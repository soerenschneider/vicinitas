package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Probes      []Probe    `yaml:"probes"`
	Mqtt        MqttConfig `yaml:"mqtt"`
	MetricsAddr string     `yaml:"metrics_addr"`
}

type Probe struct {
	Name   string `yaml:"name"`
	Target string `yaml:"target"`
}

type MqttConfig struct {
	Broker         string `json:"broker"`
	Topic          string `json:"topic"`
	ClientId       string `json:"client_id"`
	CaCertFile     string `json:"tls_ca_cert"`
	ClientCertFile string `json:"tls_client_cert"`
	ClientKeyFile  string `json:"tls_client_key"`
	TlsInsecure    bool   `json:"tls_insecure"`
}

func defaultConfig() Config {
	return Config{
		MetricsAddr: "127.0.0.1:9224",
	}
}

func ReadConfig(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	conf := defaultConfig()
	err = yaml.Unmarshal(data, &conf)
	return &conf, err
}

func (conf *MqttConfig) UsesTlsClientCerts() bool {
	return len(conf.CaCertFile) > 0 && len(conf.ClientCertFile) > 0 && len(conf.ClientKeyFile) > 0
}

func (conf *MqttConfig) TlsConfig() *tls.Config {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		log.Warn().Msgf("Could not get system cert pool")
		certPool = x509.NewCertPool()
	}

	if conf.UsesTlsClientCerts() {
		pemCerts, err := os.ReadFile(conf.CaCertFile)
		if err != nil {
			log.Error().Msgf("Could not read CA cert file: %v", err)
		} else {
			certPool.AppendCertsFromPEM(pemCerts)
		}
	}

	// #nosec G402
	tlsConf := &tls.Config{
		RootCAs:            certPool,
		ClientAuth:         tls.RequestClientCert,
		InsecureSkipVerify: conf.TlsInsecure,
	}

	clientCertDefined := len(conf.ClientCertFile) > 0
	clientKeyDefined := len(conf.ClientKeyFile) > 0
	if clientCertDefined && clientKeyDefined {
		tlsConf.GetClientCertificate = func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			cert, err := tls.LoadX509KeyPair(conf.ClientCertFile, conf.ClientKeyFile)
			return &cert, err
		}
	}

	return tlsConf
}

func (conf MqttConfig) String() string {
	base := fmt.Sprintf("broker=%s, clientId=%s", conf.Broker, conf.ClientId)
	if conf.UsesTlsClientCerts() {
		base += fmt.Sprintf("ca=%s, crt=%s, key=%s", conf.CaCertFile, conf.ClientCertFile, conf.ClientKeyFile)
	}

	return base
}
