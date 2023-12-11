package hhdhcp

import (
	"time"

	"github.com/coredhcp/coredhcp/handler"
	"github.com/coredhcp/coredhcp/logger"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

var log = logger.GetLogger("plugins/hhdhcp")

// Plugin wraps the DNS plugin information.

var pluginHdl *pluginState

func setuphhdhcp4(args ...string) (handler.Handler4, error) {
	log.Infof("loaded HH plugin for DHCPv4.")
	pluginHdl = &pluginState{
		//mactoIPMap: make(map[string]net.IPNet),
		ranges:  make(map[string]*allocations),
		backend: NewBackend(),
	}

	go handleExpiredLeases()
	// Throw this away this is for debugging
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for {
			select {
			case <-ticker.C:

				log.Infof("time Ticker")
				pluginHdl.Lock()
				log.Infof("time Ticker -- Lock")
				for k, v := range pluginHdl.ranges {
					log.Infof("Reservation Length %s::: %d", k, len(v.ipReservations.allocation))
				}

				pluginHdl.Unlock()
				log.Infof("time Ticker -- Unlock")
			}

		}
	}()
	return Handler4, nil
}

func Handler4(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool) {
	pluginHdl.Lock()
	defer pluginHdl.Unlock()

	switch req.MessageType() {
	case dhcpv4.MessageTypeDiscover:
		// Find the IP address that was is avialable for this (VrfName,VlanName) combination
		// Send the DHCP offer
		if err := handleDiscover4(req, resp); err == nil {
			return resp, true
		} else {
			log.Errorf("handleDiscover4 error: %s", err)
			return resp, true
		}
	case dhcpv4.MessageTypeRequest:
		// Check if the ip was actually offered and commit if the the offer came from the right client
		if err := handleRequest(req, resp); err != nil {
			log.Errorf("handle DHCP Request error: %s", err)
		}
	case dhcpv4.MessageTypeRelease:
		// Find the IP address. Check if the ip was actually offered to the client and release the lease.
		// Check if the ip was actually offered and commit if the the offer came from the right client
		if err := handleRelease(req, resp); err != nil {
			log.Errorf("handle DHCP Release error: %s", err)
		}
	case dhcpv4.MessageTypeDecline:
		if err := handleDecline(req, resp); err != nil {
			log.Errorf("handle DHCP Decline error: %s", err)
		}
		// Client declined the offer. Release the reservation. Client accepted another servers offer.
	default:
		log.Errorf("Received DHCP unknown message type")
		return resp, false
	}

	//vrfName := dhcpv4.RelayOptions.Get(dhcpv4)
	return resp, false
}
