package hhdhcp

import (
	"fmt"
	"net"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

var pendingDiscoverTimeout = 1000

func handleDiscover4(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) error {
	relayAgentInfo := req.RelayAgentInfo()
	var ipnet net.IPNet
	var val *allocations
	var ok bool
	var err error
	if relayAgentInfo != nil {
		circuitID := relayAgentInfo.Get(dhcpv4.AgentCircuitIDSubOption)
		vrfName := relayAgentInfo.Get(dhcpv4.VirtualSubnetSelectionSubOption)
		if val, ok = pluginHdl.ranges[string(vrfName)+string(circuitID)]; !ok {
			// Call record backend to see if the backend can retrieve this
			backendKey := map[string]string{
				"vrfName":   string(vrfName),
				"circuitID": string(circuitID),
			}
			record, err := pluginHdl.backend.GetRange(backendKey)
			if err != nil {
				return fmt.Errorf("Unknow vrf %s circuiId %s", string(vrfName), string(circuitID))
			}

			start, end, gateway, count := parseRecord(record.subnet)
			prefixLen, _ := record.subnet.Mask.Size()
			iprange, err := NewIPv4Range(start, end, gateway, prefixLen, count)
			if err != nil {
				log.Errorf("Unable to create range for vrf %s circuiId %s", string(vrfName))
				return err
			}
			val = &allocations{
				pool: iprange,
				pending: &pendingAllocations{
					allocation: make(map[string]string),
				},
			}

			pluginHdl.ranges[string(vrfName)+string(circuitID)] = val
		}

		ipnet, err = val.pool.Allocate()
		if err != nil {
			return fmt.Errorf("Unable to allocate IP for vrf %s circuiId %s error %s", string(vrfName), string(circuitID), err)
		}

		// Update the pending IP address list
		val.pending.Lock()
		val.pending.allocation[ipnet.IP.String()] = req.ClientHWAddr.String()
		val.pending.Unlock()
		time.AfterFunc(time.Duration(pendingDiscoverTimeout)*time.Millisecond, func() {
			val.pending.Lock()
			defer val.pending.Unlock()
			delete(val.pending.allocation, ipnet.IP.String())
		})

		resp.YourIPAddr = ipnet.IP
		net.ParseCIDR(ipnet.String())
		resp.Options.Update(dhcpv4.OptIPAddressLeaseTime(leaseTime))
		resp.Options.Update(dhcpv4.OptSubnetMask(ipnet.Mask))
		resp.Options.Update(dhcpv4.OptRouter(val.pool.GatewayIP()))
	}
	return nil
}

func handleDiscover4Request(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) error {
	return nil
}
