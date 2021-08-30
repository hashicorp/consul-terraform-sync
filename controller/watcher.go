package controller

import (
	"math/rand"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/retry"
	"github.com/hashicorp/hcat"
)

const (
	hcatLogSystemName = "hcat"
)

// newWatcher initializes a new hcat Watcher with a Consul client and optional
// Vault client if configured.
func newWatcher(conf *config.Config) (*hcat.Watcher, error) {
	consulConf := conf.Consul
	transport := hcat.TransportInput{
		SSLEnabled: *consulConf.TLS.Enabled,
		SSLVerify:  *consulConf.TLS.Verify,
		SSLCert:    *consulConf.TLS.Cert,
		SSLKey:     *consulConf.TLS.Key,
		SSLCACert:  *consulConf.TLS.CACert,
		SSLCAPath:  *consulConf.TLS.CAPath,
		ServerName: *consulConf.TLS.ServerName,

		DialKeepAlive:       *consulConf.Transport.DialKeepAlive,
		DialTimeout:         *consulConf.Transport.DialTimeout,
		DisableKeepAlives:   *consulConf.Transport.DisableKeepAlives,
		IdleConnTimeout:     *consulConf.Transport.IdleConnTimeout,
		MaxIdleConns:        *consulConf.Transport.MaxIdleConns,
		MaxIdleConnsPerHost: *consulConf.Transport.MaxIdleConnsPerHost,
		TLSHandshakeTimeout: *consulConf.Transport.TLSHandshakeTimeout,
	}

	consul := hcat.ConsulInput{
		Address:      *consulConf.Address,
		Token:        *consulConf.Token,
		AuthEnabled:  *consulConf.Auth.Enabled,
		AuthUsername: *consulConf.Auth.Username,
		AuthPassword: *consulConf.Auth.Password,
		Transport:    transport,
	}

	clients := hcat.NewClientSet()
	if err := clients.AddConsul(consul); err != nil {
		return nil, err
	}

	if err := setVaultClient(clients, conf); err != nil {
		return nil, err
	}

	return hcat.NewWatcher(hcat.WatcherInput{
		Clients:         clients,
		Cache:           hcat.NewStore(),
		ConsulRetryFunc: retryConsul,
	}), nil
}

// retryConsul will be used by hashicat watcher to retry polling Consul for
// catalog changes. If polling Consul fails after retries, CTS will actually
// exit.
//
// retryCount parameter is passed in by hcat. It starts at 0 (there have been
// zero retries).
func retryConsul(retryCount int) (bool, time.Duration) {
	max := 8 // 8+1 retries. total wait of 8.5-12.8 min
	logger := logging.Global().Named(hcatLogSystemName)
	if retryCount > max {
		logger.Error("error connecting with Consul even after retries", "retries", retryCount)
		return false, 0 * time.Second
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	wait := retry.WaitTime(uint(retryCount), random)
	dur := time.Duration(wait)
	logger.Debug("couldn't connect with Consul. Waiting to retry",
		"wait_duration", dur, "attempt_number", retryCount+1)
	return true, dur
}

func setVaultClient(clients *hcat.ClientSet, conf *config.Config) error {
	vaultConf := conf.Vault
	if !*vaultConf.Enabled {
		return nil
	}

	vault := hcat.VaultInput{
		Address:     *vaultConf.Address,
		Namespace:   *vaultConf.Namespace,
		Token:       *vaultConf.Token,
		UnwrapToken: *vaultConf.UnwrapToken,
		Transport: hcat.TransportInput{
			SSLEnabled: *vaultConf.TLS.Enabled,
			SSLVerify:  *vaultConf.TLS.Verify,
			SSLCert:    *vaultConf.TLS.Cert,
			SSLKey:     *vaultConf.TLS.Key,
			SSLCACert:  *vaultConf.TLS.CACert,
			SSLCAPath:  *vaultConf.TLS.CAPath,
			ServerName: *vaultConf.TLS.ServerName,

			DialKeepAlive:       *vaultConf.Transport.DialKeepAlive,
			DialTimeout:         *vaultConf.Transport.DialTimeout,
			DisableKeepAlives:   *vaultConf.Transport.DisableKeepAlives,
			IdleConnTimeout:     *vaultConf.Transport.IdleConnTimeout,
			MaxIdleConns:        *vaultConf.Transport.MaxIdleConns,
			MaxIdleConnsPerHost: *vaultConf.Transport.MaxIdleConnsPerHost,
			TLSHandshakeTimeout: *vaultConf.Transport.TLSHandshakeTimeout,
		},
	}

	return clients.AddVault(vault)
}
