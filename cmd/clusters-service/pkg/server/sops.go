package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gopgp "github.com/ProtonMail/gopenpgp/v2/crypto"
	kustomizev1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	capiv1_proto "github.com/weaveworks/weave-gitops-enterprise/cmd/clusters-service/pkg/protos"
	"github.com/weaveworks/weave-gitops/core/clustersmngr"
	"github.com/weaveworks/weave-gitops/pkg/server/auth"
	"go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/age"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/keys"
	pgp "go.mozilla.org/sops/v3/pgp"
	structpb "google.golang.org/protobuf/types/known/structpb"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/client"

	goage "filippo.io/age"
)

const (
	EncryptedRegex   = "^(data|stringData)$"
	DecryptionPGPExt = ".asc"
	DecryptionAgeExt = ".agekey"
)

type Encryptor struct {
	store common.Store
}

func NewEncryptor() *Encryptor {
	return &Encryptor{
		store: common.StoreForFormat(formats.Json),
	}
}

func (e *Encryptor) Encrypt(raw []byte, keys ...keys.MasterKey) ([]byte, error) {
	branches, err := e.store.LoadPlainFile(raw)
	if err != nil {
		return nil, err
	}

	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			EncryptedRegex: EncryptedRegex,
			KeyGroups: []sops.KeyGroup{
				keys,
			},
		},
	}

	dataKey, errs := tree.GenerateDataKey()
	if errs != nil {
		return nil, fmt.Errorf("failed to get data key: %v", errs)
	}

	err = common.EncryptTree(common.EncryptTreeOpts{
		Cipher:  aes.NewCipher(),
		DataKey: dataKey,
		Tree:    &tree,
	})

	if err != nil {
		return nil, err
	}

	encrypted, err := e.store.EmitEncryptedFile(tree)
	if err != nil {
		return nil, err
	}

	return encrypted, nil
}

func (e *Encryptor) EncryptWithPGP(raw []byte, secret string) ([]byte, error) {
	privateKey, err := gopgp.NewKeyFromArmored(secret)
	if err != nil {
		return nil, err
	}

	// return only the valid private key
	privateKeyStr, err := privateKey.Armor()
	if err != nil {
		return nil, err
	}

	err = importPGPKey(privateKeyStr)
	if err != nil {
		return nil, err
	}

	masterKey := pgp.NewMasterKeyFromFingerprint(privateKey.GetFingerprint())
	return e.Encrypt(raw, masterKey)
}

func (e *Encryptor) EncryptWithAGE(raw []byte, secret string) ([]byte, error) {
	identities, err := goage.ParseIdentities(strings.NewReader(secret))
	if err != nil {
		return nil, fmt.Errorf("failed to parse age identity: %w", err)
	}

	var recipients = []string{}
	for i := range identities {
		recipients = append(recipients, identities[i].(*goage.X25519Identity).Recipient().String())
	}

	keys, err := age.MasterKeysFromRecipients(strings.Join(recipients, ","))
	if err != nil {
		return nil, fmt.Errorf("failed to create the master key: %w", err)
	}

	if keys == nil {
		return nil, errors.New("no key found")
	}

	return e.Encrypt(raw, keys[0])
}

func (s *server) SopsEncryptSecret(ctx context.Context, msg *capiv1_proto.SopsEncryptSecretRequest) (*capiv1_proto.SopsEncryptSecretResponse, error) {
	clustersClient, err := s.clustersManager.GetImpersonatedClientForCluster(ctx, auth.Principal(ctx), msg.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("error getting impersonating client: %w", err)
	}

	var kustomization kustomizev1beta2.Kustomization
	kustomizationKey := client.ObjectKey{
		Name:      msg.KustomizationName,
		Namespace: msg.KustomizationNamespace,
	}
	if err := clustersClient.Get(ctx, msg.ClusterName, kustomizationKey, &kustomization); err != nil {
		return nil, fmt.Errorf("failed to get kustomization: %w", err)
	}

	if kustomization.Spec.Decryption == nil {
		return nil, errors.New("kustomization missing decryption settings")
	}

	if kustomization.Spec.Decryption.SecretRef == nil {
		return nil, errors.New("kustomization doesn't have decryption secret")
	}

	var decryptionSecret v1.Secret
	decryptionSecretKey := client.ObjectKey{
		Name:      kustomization.Spec.Decryption.SecretRef.Name,
		Namespace: msg.KustomizationNamespace,
	}
	if err := clustersClient.Get(ctx, msg.ClusterName, decryptionSecretKey, &decryptionSecret); err != nil {
		return nil, fmt.Errorf("failed to get decryption key: %w", err)
	}

	rawSecret, err := generateSecret(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	encryptedKey, err := encryptSecret(rawSecret, decryptionSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	result := structpb.Value{}
	err = result.UnmarshalJSON(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
	}

	return &capiv1_proto.SopsEncryptSecretResponse{
		EncryptedSecret: &result,
		Path:            kustomization.Spec.Path,
	}, nil
}

func generateSecret(msg *capiv1_proto.SopsEncryptSecretRequest) ([]byte, error) {
	data := map[string][]byte{}
	for key := range msg.Data {
		value := msg.Data[key]
		data[key] = []byte(value)
	}

	secret := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.Identifier(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      msg.Name,
			Namespace: msg.Namespace,
			Labels:    msg.Labels,
		},
		Data:       data,
		StringData: msg.StringData,
		Immutable:  &msg.Immutable,
		Type:       v1.SecretType(msg.Type),
	}

	var buf bytes.Buffer
	serializer := json.NewSerializer(json.DefaultMetaFactory, nil, nil, false)
	if err := serializer.Encode(&secret, &buf); err != nil {
		return nil, fmt.Errorf("failed to serialize object, error: %w", err)
	}

	return buf.Bytes(), nil
}

func encryptSecret(raw []byte, decryptSecret v1.Secret) ([]byte, error) {
	encryptor := NewEncryptor()
	for name, value := range decryptSecret.Data {
		switch filepath.Ext(name) {
		case DecryptionPGPExt:
			return encryptor.EncryptWithPGP(raw, string(value))
		case DecryptionAgeExt:
			return encryptor.EncryptWithAGE(raw, strings.TrimRight(string(value), "\n"))
		default:
			return nil, errors.New("invalid secret")
		}
	}
	return nil, nil
}

func importPGPKey(pk string) error {
	binary := "gpg"
	if envBinary := os.Getenv("SOPS_GPG_EXEC"); envBinary != "" {
		binary = envBinary
	}
	args := []string{"--batch", "--import"}
	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader([]byte(pk))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	return cmd.Run()
}

func (s *server) ListSOPSKustomizations(ctx context.Context, req *capiv1_proto.ListSOPSKustomizationsRequest) (*capiv1_proto.ListSOPSKustomizationsResponse, error) {

	clustersClient, err := s.clustersManager.GetImpersonatedClientForCluster(ctx, auth.Principal(ctx), req.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("error getting impersonating client: %w", err)
	}

	if clustersClient == nil {
		return nil, fmt.Errorf("cluster %s not found", req.ClusterName)
	}

	kustomizations, err := s.listSOPSKustomizations(ctx, clustersClient, req)
	if err != nil {
		return nil, err
	}

	response := capiv1_proto.ListSOPSKustomizationsResponse{
		Kustomizations: kustomizations,
		Total:          int32(len(kustomizations)),
	}

	return &response, nil
}

func (s *server) listSOPSKustomizations(ctx context.Context, cl clustersmngr.Client, req *capiv1_proto.ListSOPSKustomizationsRequest) ([]*capiv1_proto.SOPSKustomizations, error) {

	kustomizations := []*capiv1_proto.SOPSKustomizations{}

	kustomizationList := &kustomizev1beta2.KustomizationList{}
	err := cl.List(ctx, req.ClusterName, kustomizationList)
	if err != nil {
		return nil, fmt.Errorf("failed to list kustomizations, error: %w", err)
	}

	for _, kustomization := range kustomizationList.Items {
		if kustomization.Spec.Decryption != nil && strings.EqualFold(kustomization.Spec.Decryption.Provider, "sops") {
			kustomizations = append(kustomizations, &capiv1_proto.SOPSKustomizations{
				Name:      kustomization.Name,
				Namespace: kustomization.Namespace,
			})
		}
	}

	return kustomizations, nil
}
