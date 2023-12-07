package hhdhcp

import (
	"encoding/binary"
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
		if len(vrfName) > 1 {
			vrfName = vrfName[1:]
		}
		log.Infof("vrfName: %v:%s circuitID: %v,%s", vrfName[0:], string(vrfName), circuitID, string(circuitID))
		if val, ok = pluginHdl.ranges[string(vrfName)+string(circuitID)]; !ok {
			// Call record backend to see if the backend can retrieve this
			backendKey := map[string]string{
				"vrfName":   string(vrfName),
				"circuitID": string(circuitID),
			}
			record, err := pluginHdl.backend.GetRange(backendKey)
			if err != nil {
				return fmt.Errorf("unknown vrf %s circuiId %s", string(vrfName), string(circuitID))
			}

			if record == nil {
				return fmt.Errorf("unknown vrf %s circuiId %s", string(vrfName), string(circuitID))
			}

			prefixLen, _ := record.CIDRBlock.Mask.Size()
			count := binary.BigEndian.Uint32(record.EndIP) - binary.BigEndian.Uint32(record.StartIP) + 1
			iprange, err := NewIPv4Range(record.StartIP, record.EndIP, record.Gateway, count, uint32(prefixLen))
			if err != nil {
				return fmt.Errorf("unable to create range for vrf %s circuiId %s,err", string(vrfName), string(circuitID), err)

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
			return fmt.Errorf("unable to allocate IP for vrf %s circuiId %s error %s", string(vrfName), string(circuitID), err)
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
	relayAgentInfo := req.RelayAgentInfo()
	if relayAgentInfo != nil {
		circuitID := relayAgentInfo.Get(dhcpv4.AgentCircuitIDSubOption)
		vrfName := relayAgentInfo.Get(dhcpv4.VirtualSubnetSelectionSubOption)
		// if val, ok := pluginHdl.ranges[string(vrfName)+string(circuitID)]; !ok {
		// 	return fmt.Errorf("unknown vrf %s circuiId %s", string(vrfName), string(circuitID)
		// }else{
		// 	val.pending.Lock()
		// 	if mac,ok := val.pending.allocation[string(req.ClientIPAddr)];ok {}
		// 	val.pending.Unlock()
		// }
		log.Infof("vrf %s circuiId %s req Summary %s", string(vrfName), string(circuitID), req.Summary())

	}
	return nil
}
