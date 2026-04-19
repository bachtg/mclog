package tls

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

type TLSConfig struct {
	CertFile   string
	KeyFile    string
	CAFile     string
	ServerAddr string
	IsServer   bool
}

func NewTLS(config *TLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		return nil, err
	}

	caData, err := os.ReadFile(config.CAFile)
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caData) {
		return nil, err
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   config.ServerAddr,
	}

	if config.IsServer {
		tlsCfg.ClientCAs = caPool
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	} else {
		tlsCfg.RootCAs = caPool
	}

	return tlsCfg, nil
}
