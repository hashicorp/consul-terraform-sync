// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/retry"
	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/events"
)

const (
	hcatLogSystemName = "hcat"
)

// newWatcher initializes a new hcat Watcher with a Consul client and optional
// Vault client if configured.
func newWatcher(conf *config.Config, maxRetries int) (*hcat.Watcher, error) {
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

	wr := watcherRetry{
		maxRetries: maxRetries,
		waitFunc:   retry.WaitTime,
	}

	return hcat.NewWatcher(hcat.WatcherInput{
		Clients:         clients,
		Cache:           hcat.NewStore(),
		ConsulRetryFunc: wr.retryConsul,
		EventHandler:    newWatcherEventHandler(logging.Global().Named(hcatLogSystemName)),
	}), nil
}

type watcherRetry struct {
	maxRetries int
	waitFunc   func(attempt int, random *rand.Rand, maxWaitTime time.Duration) time.Duration
}

// retryConsul will be used by hashicat watcher to retry polling Consul for
// catalog changes. If polling Consul fails after retries, CTS will actually
// exit.
//
// retryCount parameter is passed in by hcat. It starts at 0 (there have been
// zero retries).
func (wr watcherRetry) retryConsul(retryCount int) (bool, time.Duration) {
	max := wr.maxRetries
	logger := logging.Global().Named(hcatLogSystemName)
	if max >= 0 && retryCount > max {
		logger.Error("error connecting with Consul even after retries", "retries", retryCount)
		return false, 0 * time.Second
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	wait := wr.waitFunc(retryCount, random, retry.DefaultMaxWaitTime) // max wait time of retry.maxWaitTime minutes
	logger.Debug("couldn't connect with Consul. Waiting to retry",
		"wait_duration", wait, "attempt_number", retryCount+1)
	return true, wait
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

func newWatcherEventHandler(logger logging.Logger) events.EventHandler {
	return func(e events.Event) {
		// Log events at different log levels based on the type
		var level logging.Level
		switch e.(type) {
		case events.Trace:
			// Only show hcat Trace events when trace logging is enabled.
			level = logging.Trace
		default:
			// Everything else is shown when debug is enabled.
			level = logging.Debug
		}

		// Default to emitting the go format for the event.
		event := fmt.Sprintf("%+v", e)

		// If the output log level is Trace, then emit the json format for the event.
		// This could be very large in certain circumstances, because it displays
		// nested data structures.
		if logger.IsTrace() {
			b, err := json.Marshal(e)
			if err != nil {
				logger.Warn("Unexpected error marshalling event to json", "error", err, "event", e)
				return
			}
			event = string(b)
		}

		// Emit the log and include the type name, if possible.
		tName := "nil"
		if t := reflect.TypeOf(e); t != nil {
			tName = t.Name()
		}
		logger.Log(level, "event received", "type", tName, "event", event)
	}
}
