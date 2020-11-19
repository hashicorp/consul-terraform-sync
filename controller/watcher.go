package controller

import (
	"github.com/hashicorp/consul-terraform-sync/config"
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
		Clients: clients,
		Cache:   hcat.NewStore(),
	}), nil
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
