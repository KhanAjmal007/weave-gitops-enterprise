package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for required values

const validTrackEKS = `
track: "eks"
clusterName: ""
dockerIOUser: "TheodoreLogan"
dockerIOPasswordFile: "testdata/passwordFile"
`

const validTrackEKSWithGitURL = `
track: "eks"
clusterName: ""
dockerIOUser: "TheodoreLogan"
dockerIOPasswordFile: "testdata/passwordFile"
`

const validTrackSSH = `
track: "wks-ssh"
clusterName: ""
dockerIOUser: "TheodoreLogan"
dockerIOPasswordFile: "testdata/passwordFile"
`

const validTrackFootloose = `
track: "wks-footloose"
clusterName: ""
dockerIOUser: "TheodoreLogan"
dockerIOPasswordFile: "testdata/passwordFile"
`

const invalidTrack = `
track: "footlose"
clusterName: ""
dockerIOUser: "TheodoreLogan"
dockerIOPasswordFile: "testdata/passwordFile"
`

const missingTrack = `
track: ""
clusterName: ""
dockerIOUser: "TheodoreLogan"
dockerIOPasswordFile: "testdata/passwordFile"
`

const missingUser = `
track: "wks-ssh"
clusterName: ""
dockerIOUser: ""
dockerIOPasswordFile: "testdata/passwordFile"
`

const missingPasswordFile = `
track: "wks-ssh"
clusterName: ""
dockerIOUser: "TheodoreLogan"
dockerIOPasswordFile: ""
`

func TestRequiredGlobals(t *testing.T) {
	testinput := []struct {
		config   string
		errorMsg string
	}{
		{validTrackEKS, "<nil>"},
		{validTrackEKSWithGitURL, "<nil>"},
		{validTrackSSH, "<nil>"},
		{validTrackFootloose, "<nil>"},
		{invalidTrack, "track must be one of: 'eks', 'wks-ssh', or 'wks-footloose'"},
		{missingTrack, "track must be specified"},
		{missingUser, "dockerIOUser must be specified"},
		{missingPasswordFile, "dockerIOPasswordFile must be specified"}}

	for _, testvals := range testinput {
		conf, err := unmarshalConfig([]byte(testvals.config))
		require.NoError(t, err)
		err = checkRequiredGlobalValues(conf)
		assert.Equal(t, testvals.errorMsg, fmt.Sprintf("%v", err))
	}
}

const gitUrl = `
gitUrl: "foo"
`

const gitlabWithUrl = `
gitProvider: "gitlab"
gitUrl: "foo"
`

const githubWithOrg = `
gitProvider: "github"
gitProviderOrg: "foo"
`

const gitlabNoUrl = `
gitProvider: "gitlab"
`

const githubNoOrg = `
gitProvider: "github"
`

const badGitProvider = `
gitProvider: "bitbucket"
`

const emptyGitProvider = `
gitProvider: ""
`

func TestValidateGitValues(t *testing.T) {
	testinput := []struct {
		config   string
		errorMsg string
	}{
		{gitUrl, "gitProvider must be one of: 'github' or 'gitlab'"},
		{gitlabWithUrl, "<nil>"},
		{githubWithOrg, "<nil>"},
		{gitlabNoUrl, "Please provide the url to your gitlab git repository in: gitUrl"},
		{githubNoOrg, "Please provide the gitProviderOrg where the repository will be created"},
		{badGitProvider, "gitProvider must be one of: 'github' or 'gitlab'"},
		{emptyGitProvider, "gitProvider must be one of: 'github' or 'gitlab'"},
	}

	for _, testvals := range testinput {
		conf, err := unmarshalConfig([]byte(testvals.config))
		require.NoError(t, err)
		err = checkRequiredGitValues(conf)
		assert.Equal(t, testvals.errorMsg, fmt.Sprintf("%v", err))
	}
}

const noKeyNoCert = `
sealedSecretsCertificate: ""
sealedSecretsPrivateKey: ""
`

const KeyNoCert = `
sealedSecretsCertificate: ""
sealedSecretsPrivateKey: "testdata/sealedSecretsKey"
`

const noKeyCert = `
sealedSecretsCertificate: "testdata/sealedSecretsCert.crt"
sealedSecretsPrivateKey: ""
`

const matchingKeyCert = `
sealedSecretsCertificate: "testdata/sealedSecretsCert.crt"
sealedSecretsPrivateKey: "testdata/sealedSecretsKey"
`

const nonMatchingKeyCert = `
sealedSecretsCertificate: "testdata/nonMatchingCert.crt"
sealedSecretsPrivateKey: "testdata/sealedSecretsKey"
`

func TestValidateSealedSecretsValues(t *testing.T) {
	testinput := []struct {
		config   string
		errorMsg string
	}{
		{noKeyNoCert, "<nil>"},
		{KeyNoCert, "please provide both the private key and certificate for the sealed secrets controller"},
		{noKeyCert, "please provide both the private key and certificate for the sealed secrets controller"},
		{matchingKeyCert,
			"<nil>"},
		{nonMatchingKeyCert,
			"could not load key and certificate pair"}}

	for _, testvals := range testinput {
		conf, err := unmarshalConfig([]byte(testvals.config))
		require.NoError(t, err)
		err = validateSealedSecretsValues(conf)
		assert.Equal(t, testvals.errorMsg, fmt.Sprintf("%v", err))
	}
}

const validEKS = `
eksConfig:
  kubernetesVersion: "1.14"
  clusterRegion: "eu-north-1"
  managedNodeGroupFile: "testdata/managedNodeGroups.yaml"
`

const validEKSWithNodeGroups = `
eksConfig:
  kubernetesVersion: "1.14"
  clusterRegion: "eu-north-1"
  nodeGroups:
  - name: "my-first-node-group"
    instanceType: "m5.small"
    desiredCapacity: 1
  - name: "my-second-node-group"
    instanceType: "m5.large"
    desiredCapacity: 2
`

const invalidNodeGroup = `
eksConfig:
  kubernetesVersion: "1.14"
  clusterRegion: "eu-north-1"
  nodeGroups:
  - name: "my-first-node-group"
    instanceType: "m5.small"
    desiredCapacity: 1
  - name: "my-second-node-group"
    instanceType: "m5.large"
    desiredCapacity: -1
`

const missingK8sVersion = `
eksConfig:
  clusterRegion: "eu-north-1"
`

const missingClusterRegion = `
eksConfig:
  kubernetesVersion: "1.14"
`

const invalidK8sVersion = `
eksConfig:
  kubernetesVersion: "1.16"
  clusterRegion: "eu-north-1"
`

const invalidManagedNodeGroupFile = `
eksConfig:
  kubernetesVersion: "1.14"
  clusterRegion: "eu-north-1"
  managedNodeGroupFile: "628wanda496"
`

func TestRequiredEKSValues(t *testing.T) {
	testinput := []struct {
		config   string
		errorMsg string
	}{
		{validEKS, "<nil>"},
		{validEKSWithNodeGroups, "<nil>"},
		{invalidNodeGroup, "A node group must have a capacity of at least 1"},
		{missingK8sVersion, "A Kubernetes version must be specified"},
		{missingClusterRegion, "clusterRegion must be specified"},
		{invalidK8sVersion, `Kubernetes version must be one of: "1.14" or "1.15"`},
		{invalidManagedNodeGroupFile, `no file found at path: "628wanda496" for field: "managedNodeGroupFile"`}}
	for _, testvals := range testinput {
		conf, err := unmarshalConfig([]byte(testvals.config))
		require.NoError(t, err)
		err = checkRequiredEKSValues(&conf.EKSConfig)
		assert.Equal(t, testvals.errorMsg, fmt.Sprintf("%v", err))
	}
}

const validWKS = `
wksConfig:
  kubernetesVersion: "1.14.1"
  serviceCIDRBlocks: [10.96.0.0/12]
  podCIDRBlocks: [192.168.1.0/16]
`

const missingWKSK8sVersion = `
wksConfig:
  serviceCIDRBlocks: [10.96.0.0/12]
  podCIDRBlocks: [192.168.1.0/16]
`

const missingServiceCIDRBlocks = `
wksConfig:
  kubernetesVersion: "1.14.1"
  podCIDRBlocks: [192.168.1.0/16]
`

const missingPodCIDRBlocks = `
wksConfig:
  kubernetesVersion: "1.14.1"
  serviceCIDRBlocks: [10.96.0.0/12]
`

const invalidWKSK8sVersion = `
wksConfig:
  kubernetesVersion: "1.16.1"
  serviceCIDRBlocks: [10.96.0.0/12]
  podCIDRBlocks: [192.168.1.0/16]
`

const invalidServiceCIDRBlock = `
wksConfig:
  kubernetesVersion: "1.14.1"
  serviceCIDRBlocks: [1000.96.0.0/12]
  podCIDRBlocks: [192.168.1.0/16]
`

const invalidPodCIDRBlock = `
wksConfig:
  kubernetesVersion: "1.14.1"
  serviceCIDRBlocks: [10.96.0.0/12]
  podCIDRBlocks: [192.1680.1.0/16]
`

// invalid ipv4 address
const invalidControlPlaneLbAddress1 = `
wksConfig:
  kubernetesVersion: "1.14.1"
  serviceCIDRBlocks: [10.96.0.0/12]
  podCIDRBlocks: [192.168.1.0/16]
  controlPlaneLbAddress: 192.1680.1.0
`

// valid ipv4 address
const validControlPlaneLbAddress1 = `
wksConfig:
  kubernetesVersion: "1.14.1"
  serviceCIDRBlocks: [10.96.0.0/12]
  podCIDRBlocks: [192.168.1.0/16]
  controlPlaneLbAddress: 192.168.1.0
`

// invalid domain
const invalidControlPlaneLbAddress2 = `
wksConfig:
  kubernetesVersion: "1.14.1"
  serviceCIDRBlocks: [10.96.0.0/12]
  podCIDRBlocks: [192.168.1.0/16]
  controlPlaneLbAddress: "hello-World-.com"
`

// valid domain
const validControlPlaneLbAddress2 = `
wksConfig:
  kubernetesVersion: "1.14.1"
  serviceCIDRBlocks: [10.96.0.0/12]
  podCIDRBlocks: [192.168.1.0/16]
  controlPlaneLbAddress: "hello-World.com"
`

func TestInvalidWKSValues(t *testing.T) {
	testinput := []struct {
		config   string
		errorMsg string
	}{
		{validWKS, "<nil>"},
		{missingWKSK8sVersion, "A Kubernetes version must be specified"},
		{missingServiceCIDRBlocks, "At least one service CIDR block must be specified"},
		{missingPodCIDRBlocks, "At least one pod CIDR block must be specified"},
		{invalidWKSK8sVersion,
			"1.16.1 is not a valid Kubernetes version; must be 1.14.x-1.15.x"},
		{invalidServiceCIDRBlock, "1000.96.0.0/12 is not a valid CIDR specification"},
		{invalidPodCIDRBlock, "192.1680.1.0/16 is not a valid CIDR specification"},
		{invalidControlPlaneLbAddress1, "192.1680.1.0 is not a valid control plane load balancer address; must be a valid IP address or a domain name"},
		{invalidControlPlaneLbAddress2, "hello-World-.com is not a valid control plane load balancer address; must be a valid IP address or a domain name"},
	}

	for _, testvals := range testinput {
		conf, err := unmarshalConfig([]byte(testvals.config))
		require.NoError(t, err)
		err = checkRequiredWKSValues(&conf.WKSConfig)
		assert.Equal(t, testvals.errorMsg, fmt.Sprintf("%v", err))
	}
}

func TestValidWKSValues(t *testing.T) {
	testinput := []struct {
		config string
	}{
		{validControlPlaneLbAddress1},
		{validControlPlaneLbAddress2},
	}

	for _, testvals := range testinput {
		conf, err := unmarshalConfig([]byte(testvals.config))
		require.NoError(t, err)
		err = checkRequiredWKSValues(&conf.WKSConfig)
		require.NoError(t, err)
	}
}

const validSSH = `
wksConfig:
  sshConfig:
    machines:
    - role: master
      publicAddress: 172.17.20.5
    - role: worker
      publicAddress: 172.17.20.6
`

const validSSHWithKey = `
wksConfig:
  sshConfig:
    sshKeyFile: "testdata/sshKey"
    machines:
    - role: master
      publicAddress: 172.17.20.5
    - role: worker
      publicAddress: 172.17.20.6
`

const missingMachines = `
wksConfig:
  sshConfig:
`

const missingWorker = `
wksConfig:
  sshConfig:
    machines:
    - role: master
      publicAddress: 172.17.20.5
`

const missingMaster = `
wksConfig:
  sshConfig:
    machines:
    - role: worker
      publicAddress: 172.17.20.5
`

const missingRole = `
wksConfig:
  sshConfig:
    machines:
    - publicAddress: 172.17.20.5
    - role: worker
      publicAddress: 172.17.20.6
`

const invalidRole = `
wksConfig:
  sshConfig:
    machines:
    - role: supervisor
      publicAddress: 172.17.20.5
    - role: worker
      publicAddress: 172.17.20.6
`

const invalidSSHKeyFile = `
wksConfig:
  sshConfig:
    sshKeyFile: "8128goober"
    machines:
    - role: master
      publicAddress: 172.17.20.5
    - role: worker
      publicAddress: 172.17.20.6
`

func TestRequiredSSHValues(t *testing.T) {
	testinput := []struct {
		config   string
		errorMsg string
	}{
		{validSSH, "<nil>"},
		{validSSHWithKey, "<nil>"},
		{missingMachines, "No machine information provided"},
		{missingWorker,
			"Invalid machine set. At least one master and one worker must be specified."},
		{missingMaster,
			"Invalid machine set. At least one master and one worker must be specified."},
		{missingRole,
			"A role ('master' or 'worker') must be specified for each machine"},
		{invalidRole,
			"Invalid machine role: 'supervisor'. Only 'master' and 'worker' are valid."},
		{invalidSSHKeyFile, `no file found at path: "8128goober" for field: "sshKeyFile"`}}

	for _, testvals := range testinput {
		conf, err := unmarshalConfig([]byte(testvals.config))
		require.NoError(t, err)
		err = checkRequiredSSHValues(&conf.WKSConfig.SSHConfig)
		assert.Equal(t, testvals.errorMsg, fmt.Sprintf("%v", err))
	}
}

const validFootlooseDocker = `
wksConfig:
  footlooseConfig:
    backend: docker
    controlPlaneNodes: 1
    workerNodes: 1
`

const validFootlooseIgnite = `
wksConfig:
  footlooseConfig:
    backend: ignite
    controlPlaneNodes: 1
    workerNodes: 1
`

const missingFootlooseBackend = `
wksConfig:
  footlooseConfig:
    controlPlaneNodes: 1
    workerNodes: 1
`

const invalidFootlooseBackend = `
wksConfig:
  footlooseConfig:
    backend: igniter
    controlPlaneNodes: 1
    workerNodes: 1
`

const invalidNegativeFootlooseControlPlaneCount = `
wksConfig:
  footlooseConfig:
    backend: ignite
    controlPlaneNodes: -1
    workerNodes: 1
`

const invalidZeroFootlooseControlPlaneCount = `
wksConfig:
  footlooseConfig:
    backend: ignite
    controlPlaneNodes: 0
    workerNodes: 1
`

const invalidNegativeFootlooseWorkerCount = `
wksConfig:
  footlooseConfig:
    backend: ignite
    controlPlaneNodes: 1
    workerNodes: -1
`

const invalidZeroFootlooseWorkerCount = `
wksConfig:
  footlooseConfig:
    backend: ignite
    controlPlaneNodes: 1
    workerNodes: 0
`

func TestRequiredFootlooseValues(t *testing.T) {
	testinput := []struct {
		config   string
		errorMsg string
	}{
		{validFootlooseDocker, "<nil>"},
		{validFootlooseIgnite, "<nil>"},
		{missingFootlooseBackend, "A footloose backend must be specified"},
		{invalidFootlooseBackend, "A footloose backend must be either 'docker' or 'ignite'"},
		{invalidNegativeFootlooseControlPlaneCount,
			"A footloose specification must have at least one control plane node"},
		{invalidZeroFootlooseControlPlaneCount,
			"A footloose specification must have at least one control plane node"},
		{invalidNegativeFootlooseWorkerCount,
			"A footloose specification must have at least one worker node"},
		{invalidZeroFootlooseWorkerCount,
			"A footloose specification must have at least one worker node"}}
	for _, testvals := range testinput {
		conf, err := unmarshalConfig([]byte(testvals.config))
		require.NoError(t, err)
		err = checkRequiredFootlooseValues(&conf.WKSConfig.FootlooseConfig)
		assert.Equal(t, testvals.errorMsg, fmt.Sprintf("%v", err))
	}
}

// Tests for default values

func TestDefaultGlobals(t *testing.T) {
	conf, err := unmarshalConfig([]byte(validTrackEKS))
	require.NoError(t, err)
	setDefaultGlobalValues(conf)
	nameComponent := os.Getenv("USER")
	if nameComponent == "" {
		nameComponent = "cluster"
	}
	assert.Equal(t, "wk-"+nameComponent, conf.ClusterName)
}

const nodeGroupNeedsDefaults = `
eksConfig:
  kubernetesVersion: "1.14"
  clusterRegion: "eu-north-1"
  nodeGroups:
  - instanceType: "m5.small"
  - instanceType: "m5.large"
`

func TestDefaultEKSValues(t *testing.T) {
	conf, err := unmarshalConfig([]byte(validEKS))
	require.NoError(t, err)
	setDefaultEKSValues(&conf.EKSConfig)
	ng := conf.EKSConfig.NodeGroups[0]
	assert.Equal(t, "ng-0", ng.Name)
	assert.Equal(t, "m5.large", ng.InstanceType)
	assert.Equal(t, int64(3), ng.DesiredCapacity)

	conf, err = unmarshalConfig([]byte(nodeGroupNeedsDefaults))
	require.NoError(t, err)
	setDefaultEKSValues(&conf.EKSConfig)
	ng0 := conf.EKSConfig.NodeGroups[0]
	assert.Equal(t, "ng-0", ng0.Name)
	assert.Equal(t, "m5.small", ng0.InstanceType)
	assert.Equal(t, int64(3), ng0.DesiredCapacity)
	ng1 := conf.EKSConfig.NodeGroups[1]
	assert.Equal(t, "ng-1", ng1.Name)
	assert.Equal(t, "m5.large", ng1.InstanceType)
	assert.Equal(t, int64(3), ng1.DesiredCapacity)
}

func TestDefaultSSHValues(t *testing.T) {
	conf, err := unmarshalConfig([]byte(validSSH))
	require.NoError(t, err)
	setDefaultSSHValues(&conf.WKSConfig.SSHConfig)
	assert.Equal(t, "root", conf.WKSConfig.SSHConfig.SSHUser)
	assert.Equal(t, fmt.Sprintf("%s/.ssh/id_rsa", os.Getenv("HOME")), conf.WKSConfig.SSHConfig.SSHKeyFile)
	machines := conf.WKSConfig.SSHConfig.Machines
	m0 := machines[0]
	assert.Equal(t, int64(22), m0.PublicPort)
	assert.Equal(t, int64(22), m0.PrivatePort)
	assert.Equal(t, "172.17.20.5", m0.PrivateAddress)
	m1 := machines[1]
	assert.Equal(t, int64(22), m1.PublicPort)
	assert.Equal(t, int64(22), m1.PrivatePort)
	assert.Equal(t, "172.17.20.6", m1.PrivateAddress)
}
