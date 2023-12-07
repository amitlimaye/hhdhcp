package hhdhcp

import (
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
	return Handler4, nil
}

func Handler4(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool) {
	// First check and extract all possible keys from the dhcpv4 request
	// Which interface did the packet came on
	// Do we have a mac to IP address mapping?
	// do we have a gateway or relay interface information
	//
	// Do we have a map with the ip of the interface ?
	//
	pluginHdl.Lock()
	defer pluginHdl.Unlock()
	// if val, err := pluginHdl.mactoIPMap.Get(req.ClientHWAddr.String()); err == nil {
	// 	resp.Options.Update(dhcpv4.OptIPAddressLeaseTime(leaseTime))
	// 	resp.YourIPAddr = val.(net.IP)

	// }
	// Is there a learnt subnet for (VrfName,VlanName)
	switch req.MessageType() {
	case dhcpv4.MessageTypeDiscover:
		// Find the IP address that was is avialable for this (VrfName,VlanName) combination
		// Send the DHCP offer
		if err := handleDiscover4(req, resp); err == nil {
			return resp, false
		} else {
			log.Errorf("handleDiscover4 error: %s", err)
			return resp, true
		}
	case dhcpv4.MessageTypeRequest:
		// Check if the ip was actually offered and commit if the the offer came from the right client
		handleDiscover4Request(req, resp)
	case dhcpv4.MessageTypeRelease:
		// Find the IP address. Check if the ip was actually offered to the client and release the lease.
	case dhcpv4.MessageTypeDecline:
		// Client declined the offer. Release the reservation. Client accepted another servers offer.
	default:
		return resp, false
	}

	//vrfName := dhcpv4.RelayOptions.Get(dhcpv4)
	return resp, false
}
