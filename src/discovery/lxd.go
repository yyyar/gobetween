/**
 * lxd.go - LXD API discovery implementation
 *
 * @author Joe Topjian <joe@topjian.net>
 */

package discovery

import (
	"fmt"
	"strings"
	"time"

	"../config"
	"../core"
	"../logging"
	"../utils"

	"github.com/jtopjian/lxdhelpers"

	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
)

const (
	lxdRetryWaitDuration = 2 * time.Second
	lxdTimeout           = 5 * time.Second
)

/**
 * Create new Discovery with LXD fetch func
 */
func NewLXDDiscovery(cfg config.DiscoveryConfig) interface{} {

	d := Discovery{
		opts:  DiscoveryOpts{lxdRetryWaitDuration},
		fetch: lxdFetch,
		cfg:   cfg,
	}

	return &d
}

/**
 * Fetch backends from LXD API
 */
func lxdFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {
	log := logging.For("lxdFetch")

	/* Get an LXD client */
	client, err := lxdBuildClient(cfg)
	if err != nil {
		return nil, err
	}

	/* Set the timeout for the client */
	client.Http.Timeout = utils.ParseDurationOrDefault(cfg.Timeout, lxdTimeout)

	log.Debug("Fetching containers from ", client.Config.Remotes[cfg.LXDServerRemoteName].Addr)

	/* Create backends from response */
	backends := []core.Backend{}

	/* Fetch containers */
	containers, err := client.ListContainers()
	if err != nil {
		return nil, err
	}

	for _, container := range containers {

		/* Ignore containers that aren't running */
		if container.Status != "Running" {
			continue
		}

		/* Ignore continers if not match label key and value */
		if cfg.LXDContainerLabelKey != "" {

			actualLabelValue, ok := container.Config[cfg.LXDContainerLabelKey]
			if !ok {
				continue
			}

			if cfg.LXDContainerLabelValue != "" && actualLabelValue != cfg.LXDContainerLabelValue {
				continue
			}
		}

		/* Try get container port either from label, or from discovery config */
		port := fmt.Sprintf("%v", cfg.LXDContainerPort)

		if cfg.LXDContainerPortKey != "" {
			if p, ok := container.Config[cfg.LXDContainerPortKey]; ok {
				port = p
			}
		}

		if port == "" {
			log.Warn(fmt.Sprintf("Port is not found in neither in lxd_container_port config not in %s label for %s. Skipping",
				cfg.LXDContainerPortKey, container.Name))
			continue
		}

		/* iface is the container interface to get an IP address. */
		/* This isn't exposed by the LXD API, and containers can have multiple interfaces, */
		iface := cfg.LXDContainerInterface
		if v, ok := container.Config[cfg.LXDContainerInterfaceKey]; ok {
			iface = v
		}

		ip := ""
		if ip, err = lxdDetermineContainerIP(client, container.Name, iface, cfg.LXDContainerAddressType); err != nil {
			log.Error(fmt.Sprintf("Can't determine %s container ip address: %s. Skipping", container.Name, err))
			continue
		}

		sni := ""
		if v, ok := container.Config[cfg.LXDContainerSNIKey]; ok {
			sni = v
		}

		backends = append(backends, core.Backend{
			Target: core.Target{
				Host: ip,
				Port: port,
			},
			Priority: 1,
			Weight:   1,
			Stats: core.BackendStats{
				Live: true,
			},
			Sni: sni,
		})
	}

	return &backends, nil
}

/**
 * Create new LXD Client
 */
func lxdBuildClient(cfg config.DiscoveryConfig) (*lxd.Client, error) {
	log := logging.For("lxdBuildClient")

	/* Make a client to pass around */
	var client *lxd.Client

	/* Build a configuration with the requested options */
	lxdConfig, err := lxdBuildConfig(cfg)
	if err != nil {
		return client, err
	}

	if strings.HasPrefix(cfg.LXDServerAddress, "https:") {

		/* Validate or generate certificates on the client side (gobetween) */
		if err := lxdhelpers.ValidateClientCertificates(lxdConfig, cfg.LXDGenerateClientCerts); err != nil {
			return nil, err
		}

		/* Validate or accept certificates on the server side (LXD) */
		serverCertf := lxdConfig.ServerCertPath(cfg.LXDServerRemoteName)
		if !shared.PathExists(serverCertf) {

			/* If the server certificate was not found, either gobetween and the LXD server are set
			 * up for PKI, or gobetween must authenticate with the LXD server and accept its server
			 * certificate.
			 *
			 * First, create a simple LXD client
			 */
			client, err = lxd.NewClient(&lxdConfig, cfg.LXDServerRemoteName)
			if err != nil {
				return nil, err
			}

			/* Next, check if the client is able to communicate with the LXD server. If it can,
			 * this means that gobetween and the LXD server are configured with PKI certificates
			 * from a private CA.
			 *
			 * But if there's an error, then gobetween will try to download the server's cert.
			 */
			if _, err := client.GetServerConfig(); err != nil {
				if cfg.LXDAcceptServerCert {
					var err error
					client, err = lxdhelpers.GetRemoteCertificate(client, cfg.LXDServerRemoteName)
					if err != nil {
						return nil, fmt.Errorf("Could not add the LXD server: ", err)
					}
				} else {
					err := fmt.Errorf("Unable to communicate with LXD server. Either set " +
						"lxd_accept_server_cert to true or add the LXD server out of " +
						"band of gobetween and try again.")
					return nil, err
				}
			}

			/*
			 * Finally, check and see if gobetween needs to authenticate with the LXD server.
			 * Authentication happens only once. After that, gobetween will be a trusted client
			 * as long as the exchanged certificates to not change.
			 *
			 * Authentication must happen even if PKI is in use.
			 */
			log.Info("Attempting to authenticate")
			err = lxdhelpers.ValidateRemoteConnection(client, cfg.LXDServerRemoteName, cfg.LXDServerRemotePassword)
			if err != nil {
				log.Info("Authentication unsuccessful")
				return nil, err
			}

			log.Info("Authentication successful")
		}
	}

	/* Build a new client */
	client, err = lxd.NewClient(&lxdConfig, cfg.LXDServerRemoteName)
	if err != nil {
		return nil, err
	}

	/* Validate the client config and connectivity */
	if _, err := client.GetServerConfig(); err != nil {
		return nil, err
	}

	return client, nil
}

/**
 * Create LXD Client Config
 */
func lxdBuildConfig(cfg config.DiscoveryConfig) (lxd.Config, error) {
	log := logging.For("lxdBuildConfig")

	log.Debug("Using API: ", cfg.LXDServerAddress)

	/* Build an LXD configuration that will connect to the requested LXD server */
	config := lxd.Config{
		ConfigDir: cfg.LXDConfigDirectory,
		Remotes:   make(map[string]lxd.RemoteConfig),
	}
	config.Remotes[cfg.LXDServerRemoteName] = lxd.RemoteConfig{Addr: cfg.LXDServerAddress}

	return config, nil
}

/**
 * Get container IP address depending on network interface and address type
 */
func lxdDetermineContainerIP(client *lxd.Client, container, iface, addrType string) (string, error) {
	var containerIP string

	/* Convert addrType to inet */
	var inet string
	switch addrType {
	case "IPv4":
		inet = "inet"
	case "IPv6":
		inet = "inet6"
	}

	cstate, err := client.ContainerState(container)
	if err != nil {
		return "", err
	}

	for i, network := range cstate.Network {
		if i != iface {
			continue
		}

		for _, ip := range network.Addresses {
			if ip.Family == inet {
				containerIP = ip.Address
				break
			}
		}
	}

	/* If IPv6, format correctly */
	if inet == "inet6" {
		containerIP = fmt.Sprintf("[%s]", containerIP)
	}

	if containerIP == "" {
		return "", fmt.Errorf("Unable to determine IP address for LXD container %s", container)
	}

	return containerIP, nil
}
