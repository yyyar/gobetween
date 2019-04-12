package discovery

/**
 * lxd.go - LXD API discovery implementation
 *
 * @author Joe Topjian <joe@topjian.net>
 */

import (
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	lxd "github.com/lxc/lxd/client"
	lxd_config "github.com/lxc/lxd/lxc/config"
	"github.com/lxc/lxd/shared"
	lxd_api "github.com/lxc/lxd/shared/api"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/utils"
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

	/* Get an LXD config */
	lxdConfig, err := lxdBuildConfig(cfg)
	if err != nil {
		return nil, err
	}

	/* Set the timeout for the client */
	httpClient, err := client.GetHTTPClient()
	if err != nil {
		return nil, err
	}

	httpClient.Timeout = utils.ParseDurationOrDefault(cfg.Timeout, lxdTimeout)

	log.Debug("Fetching containers from ", lxdConfig.Remotes[cfg.LXDServerRemoteName].Addr)

	/* Create backends from response */
	backends := []core.Backend{}

	/* Fetch containers */
	containers, err := client.GetContainers()
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
func lxdBuildClient(cfg config.DiscoveryConfig) (lxd.ContainerServer, error) {
	log := logging.For("lxdBuildClient")

	/* Make a client to pass around */
	var client lxd.ContainerServer

	/* Build a configuration with the requested options */
	lxdConfig, err := lxdBuildConfig(cfg)
	if err != nil {
		return client, err
	}

	if strings.HasPrefix(cfg.LXDServerAddress, "https:") {

		/* Validate or generate certificates on the client side (gobetween) */
		if cfg.LXDGenerateClientCerts {
			log.Debug("Generating LXD client certificates")
			if err := lxdConfig.GenerateClientCertificate(); err != nil {
				return nil, err
			}
		}

		/* Validate or accept certificates on the server side (LXD) */
		serverCertf := lxdConfig.ServerCertPath(cfg.LXDServerRemoteName)
		if !shared.PathExists(serverCertf) {
			/* If the server certificate was not found, either gobetween and the LXD server are set
			 * up for PKI, or gobetween must authenticate with the LXD server and accept its server
			 * certificate.
			 *
			 * First, see if communication with the LXD server is possible.
			 */
			_, err := lxdConfig.GetContainerServer(cfg.LXDServerRemoteName)
			if err != nil {
				/* If there was an error, then gobetween will try to download the server's cert. */
				if cfg.LXDAcceptServerCert {
					log.Debug("Retrieving LXD server certificate")
					err := lxdGetRemoteCertificate(lxdConfig, cfg.LXDServerRemoteName)
					if err != nil {
						return nil, fmt.Errorf("Could obtain LXD server certificate: %s", err)
					}
				} else {
					err := fmt.Errorf("Unable to communicate with LXD server. Either set " +
						"lxd_accept_server_cert to true or add the LXD server out of " +
						"band of gobetween and try again.")
					return nil, err
				}
			}
		}

		/*
		 * Finally, check and see if gobetween needs to authenticate with the LXD server.
		 * Authentication happens only once. After that, gobetween will be a trusted client
		 * as long as the exchanged certificates to not change.
		 *
		 * Authentication must happen even if PKI is in use.
		 */
		client, err = lxdConfig.GetContainerServer(cfg.LXDServerRemoteName)
		if err != nil {
			return nil, err
		}

		log.Info("Authenticating to LXD server")
		err = lxdAuthenticateToServer(client, cfg.LXDServerRemoteName, cfg.LXDServerRemotePassword)
		if err != nil {
			log.Info("Authentication unsuccessful")
			return nil, err
		}

		log.Info("Authentication successful")
	}

	/* Build a new client */
	client, err = lxdConfig.GetContainerServer(cfg.LXDServerRemoteName)
	if err != nil {
		return nil, err
	}

	/* Validate the client config and connectivity */
	if _, _, err := client.GetServer(); err != nil {
		return nil, err
	}

	return client, nil
}

/**
 * Create LXD Client Config
 */
func lxdBuildConfig(cfg config.DiscoveryConfig) (*lxd_config.Config, error) {
	log := logging.For("lxdBuildConfig")

	log.Debug("Using API: ", cfg.LXDServerAddress)

	/* Build an LXD configuration that will connect to the requested LXD server */
	var config *lxd_config.Config
	if conf, err := lxd_config.LoadConfig(cfg.LXDConfigDirectory); err != nil {
		config = &lxd_config.DefaultConfig
		config.ConfigDir = cfg.LXDConfigDirectory
	} else {
		config = conf
	}

	config.Remotes[cfg.LXDServerRemoteName] = lxd_config.Remote{Addr: cfg.LXDServerAddress}
	return config, nil
}

/**
* lxdGetRemoteCertificate will attempt to retrieve a remote LXD server's
  certificate and save it to the servercert's path.
*/
func lxdGetRemoteCertificate(config *lxd_config.Config, remote string) error {
	addr := config.Remotes[remote]
	certificate, err := shared.GetRemoteCertificate(addr.Addr)
	if err != nil {
		return err
	}

	serverCertDir := config.ConfigPath("servercerts")
	if err := os.MkdirAll(serverCertDir, 0750); err != nil {
		return fmt.Errorf("Could not create server cert dir: %s", err)
	}

	certf := fmt.Sprintf("%s/%s.crt", serverCertDir, remote)
	certOut, err := os.Create(certf)
	if err != nil {
		return err
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})
	certOut.Close()

	return nil
}

/**
 * lxdAuthenticateToServer authenticates to an LXD Server
 */
func lxdAuthenticateToServer(client lxd.ContainerServer, remote string, password string) error {
	srv, _, err := client.GetServer()
	if srv.Auth == "trusted" {
		return nil
	}

	req := lxd_api.CertificatesPost{
		Password: password,
	}
	req.Type = "client"

	err = client.CreateCertificate(req)
	if err != nil {
		return fmt.Errorf("Unable to authenticate with remote server: %s", err)
	}

	_, _, err = client.GetServer()
	if err != nil {
		return err
	}

	return nil
}

/**
 * Get container IP address depending on network interface and address type
 */
func lxdDetermineContainerIP(client lxd.ContainerServer, container, iface, addrType string) (string, error) {
	var containerIP string

	/* Convert addrType to inet */
	var inet string
	switch addrType {
	case "IPv4":
		inet = "inet"
	case "IPv6":
		inet = "inet6"
	}

	cstate, _, err := client.GetContainerState(container)
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
