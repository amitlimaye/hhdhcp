package hhdhcp

import (
	"fmt"
	"sync"
	"time"

	//"github.com/coredhcp/coredhcp/handler"
	"github.com/coredhcp/coredhcp/logger"
	"github.com/coredhcp/coredhcp/plugins"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

var log = logger.GetLogger("plugins/hhdhcp")

// Plugin wraps the DNS plugin information.
var Plugin = plugins.Plugin{
	Name:   "hhdhcp",
	Setup6: nil,
	Setup4: setuphhdhcp4,
}

type rangeKey struct {
	constructedKey string
}

var leaseTime = time.Duration(3600 * time.Second)

type pluginState struct {
	//	mactoIPMap cache.Cache              // mac -> ip
	ranges map[string]IPv4Allocator //
	sync.RWMutex
}

var pluginHdl *pluginState

func setuphhdhcp4(args ...string) (handler.Handler4, error) {
	log.Infof("loaded HH plugin for DHCPv4.")
	pluginHdl = &pluginState{
		//mactoIPMap: make(map[string]net.IPNet),
		ranges: make(map[string]IPv4Allocator),
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
	circuitID := dhcpv4.RelayOptions.Get(dhcpv4.RelayOptions{}, dhcpv4.OptionRelayAgentInformation)
	fmt.Println(circuitID)
	//vrfName := dhcpv4.RelayOptions.Get(dhcpv4)
	return resp, false
}

// This function returns the key that can be  used to retrieve the range that represents
func findSubnet(keys map[string]string) (string, error) {
	return "", nil
}
