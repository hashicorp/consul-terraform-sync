package controller

import (
	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/hcat"
)

// newWatcher initializes a new hcat Watcher with a Consul client
func newWatcher(conf *config.Config) *hcat.Watcher {
	return hcat.NewWatcher(hcat.WatcherInput{
		Clients: newConsulClient(conf),
		Cache:   hcat.NewStore(),
	})
}

// newConsulClient creates a new Consul client used for monitoring the Service
// Catalog
func newConsulClient(conf *config.Config) hcat.Looker {
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

	cs := hcat.NewClientSet()
	return cs.AddConsul(consul)
}
