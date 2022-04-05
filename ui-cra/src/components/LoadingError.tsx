import React, { useEffect, useState } from 'react';
import { LoadingPage } from '@weaveworks/weave-gitops';
import { Refresh } from '@material-ui/icons';
import { createStyles, makeStyles } from '@material-ui/styles';
import Alert from '@material-ui/lab/Alert';
import styled from 'styled-components';

export interface ILoadingError {
  fetchFn: () => Promise<any>;
  children?: any;
}

const useStyles = makeStyles(() =>
  createStyles({
    retry: {
      marginLeft: '4px',
      cursor: 'pointer',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    },
  }),
);

const FlexCenter = styled.div`
  display: flex;
  lign-items: center;
  justify-content: center;
`;

const FlexStart = styled.div`
  display: flex;
  align-items: center;
  justify-content: start;
`;

const LoadingError: React.FC<any> = ({ children, fetchFn }: ILoadingError) => {
  const classes = useStyles();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [data, setData] = useState<any>();

  const fetchLoad = (fn: Promise<any>) => {
    setLoading(true);
    setError(false);
    return fn
      .then(res => {
        setData(res);
      })
      .catch(err => {
        setErrorMessage(err.message || 'Something Went wrong');
        setError(true);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(() => {
    setLoading(true);
    setError(false);
    fetchLoad(fetchFn());

    return () => {
      setData(null);
    };
  }, [fetchFn]);

  return (
    <>
      {loading && (
        <FlexCenter>
          <LoadingPage />
        </FlexCenter>
      )}
      {!loading && error && (
        <div>
          <Alert severity="error">
            <FlexStart>
              {errorMessage}
              <span
                onClick={() => fetchLoad(fetchFn())}
                className={classes.retry}
              >
                <Refresh />
              </span>
            </FlexStart>
          </Alert>
        </div>
      )}
      {!loading && !error && children({ value: data })}
    </>
  );
};

export default LoadingError;
