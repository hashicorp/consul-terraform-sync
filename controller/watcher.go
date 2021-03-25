package controller

import (
	"log"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/retry"
	"github.com/hashicorp/hcat"
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
// so the retry is more liberal.
//
// Supports a situation where Consul has stopped and needs a few minutes (and
// wiggle room) to automatically restart.
func retryConsul(attempt int) (bool, time.Duration) {
	maxRetry := 8 // at 8th retry, will have waited a total of 8.5-12.8 minutes
	if attempt > maxRetry {
		log.Printf("[ERR] (hcat) error connecting with consul even after retries")
		return false, 0 * time.Second
	}

	r := retry.NewRetry(0, time.Now().UnixNano()) // 0 is not used
	wait := r.WaitTime(uint(attempt))
	dur := time.Duration(wait)
	log.Printf("[WARN] (hcat) error connecting with consul. Will retry attempt #%d after %v", attempt, dur)
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
