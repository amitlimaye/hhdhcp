package hhdhcp

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

var pendingDiscoverTimeout = 1000 * time.Millisecond

func handleDiscover4(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) error {
	relayAgentInfo := req.RelayAgentInfo()

	if relayAgentInfo != nil {

		circuitID := relayAgentInfo.Get(dhcpv4.AgentCircuitIDSubOption)
		vrfName := relayAgentInfo.Get(dhcpv4.VirtualSubnetSelectionSubOption)
		log.Infof("handleDiscover4:: circuitID: %s, vrfName: %s", string(circuitID), string(vrfName))
		if len(vrfName) > 1 {
			vrfName = vrfName[1:]
		}
		val, err := getRange(string(vrfName), string(circuitID))
		if err != nil {
			return err
		}

		// We know we have seen this (vrf,circuitID) before.
		// Lets check if we know this client
		val.ipReservations.Lock()
		if reservation, ok := val.ipReservations.allocation[req.ClientHWAddr.String()]; ok {
			resp.YourIPAddr = reservation.address.IP
			net.ParseCIDR(reservation.address.String())
			resp.Options.Update(dhcpv4.OptIPAddressLeaseTime(leaseTime))
			resp.Options.Update(dhcpv4.OptSubnetMask(reservation.address.Mask))
			resp.Options.Update(dhcpv4.OptRouter(val.pool.GatewayIP()))
			val.ipReservations.Unlock()
			return nil
		}

		// This is a new client now allocate IP
		ipnet, err := val.pool.Allocate()
		if err != nil {
			return fmt.Errorf("handleDiscover :: unable to allocate IP for vrf %s circuiId %s error %s", string(vrfName), string(circuitID), err)
		}

		// Update the pending IP address list

		val.ipReservations.Lock()

		val.ipReservations.allocation[req.ClientHWAddr.String()] = &ipreservation{
			address:    ipnet,
			state:      pending,
			MacAddress: req.ClientHWAddr.String(),
			Hostname:   req.HostName(),
			expiry:     time.Now().Add(leaseTime),
		}
		log.Infof("handleDiscover4 :: Added IP %s to reservations list length %d", ipnet.IP, len(val.ipReservations.allocation))
		val.ipReservations.Unlock()
		time.AfterFunc(pendingDiscoverTimeout, func() {

			val.ipReservations.Lock()
			defer val.ipReservations.Unlock()
			if reservation, ok := val.ipReservations.allocation[req.ClientHWAddr.String()]; ok {
				if reservation.state == committed {
					log.Infof("handleDiscover4 :: IP %s already committed", ipnet.IP)
					return
				}
				log.Infof("handleDiscover4 :: removing IP %s from pending list", ipnet.IP)
				delete(val.ipReservations.allocation, req.ClientHWAddr.String())
				val.pool.Free(ipnet)
			}
			// Clear from the bitset

		})
		resp.YourIPAddr = ipnet.IP
		net.ParseCIDR(ipnet.String())
		resp.Options.Update(dhcpv4.OptIPAddressLeaseTime(leaseTime))
		resp.Options.Update(dhcpv4.OptSubnetMask(ipnet.Mask))
		resp.Options.Update(dhcpv4.OptRouter(val.pool.GatewayIP()))

	}
	return nil
}

func handleRequest(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) error {
	relayAgentInfo := req.RelayAgentInfo()
	if relayAgentInfo != nil {
		circuitID := relayAgentInfo.Get(dhcpv4.AgentCircuitIDSubOption)
		vrfName := relayAgentInfo.Get(dhcpv4.VirtualSubnetSelectionSubOption)
		if len(vrfName) > 1 {
			vrfName = vrfName[1:]
		}

		val, err := getRange(string(vrfName), string(circuitID))
		if err != nil {
			return fmt.Errorf("handleRequest::unknown vrf %s circuiId %s", string(vrfName), string(circuitID))
		}

		val.ipReservations.Lock()
		defer val.ipReservations.Unlock()

		if ip, ok := val.ipReservations.allocation[req.ClientHWAddr.String()]; ok { // We have seen DHCP disocver from this client and we reserved an IP for it or there is already an active reservation for this client
			log.Infof("handleRequest::allocated in discover :: for vrf %s circuitID %s ip %s", string(vrfName), string(circuitID), ip.address.String())
			// Clean pending state and allocate this IP
			val.ipReservations.allocation[req.ClientHWAddr.String()].state = committed
			//everytime we see a DHCP request update the expiry time
			val.ipReservations.allocation[req.ClientHWAddr.String()].expiry = time.Now().Add(leaseTime)
			resp.YourIPAddr = ip.address.IP
			resp.Options.Update(dhcpv4.OptIPAddressLeaseTime(leaseTime))
			resp.Options.Update(dhcpv4.OptSubnetMask(ip.address.Mask))
			resp.Options.Update(dhcpv4.OptRouter(val.pool.GatewayIP()))

			pluginHdl.backend.RecordAllocation(map[string]string{"vrfName": string(vrfName), "circuitID": string(circuitID)}, &allocationRecord{IP: resp.YourIPAddr, MacAddress: req.ClientHWAddr.String(), Hostname: req.HostName(), Expiry: val.ipReservations.allocation[req.ClientHWAddr.String()].expiry})

		} else { // This is a fresh DHCP request we have did not see DHCP discover
			// Before we allocate a new IP check if there is an active allocation for this client
			ipnet, err := val.pool.Allocate()
			if err != nil {
				return fmt.Errorf("handleRequest:: unable to allocate IP for vrf %s circuiId %s error %s", string(vrfName), string(circuitID), err)
			}
			log.Infof("handleRequest:: no discover allocated ip:: for vrf %s circuitID %s ip %s", string(vrfName), string(circuitID), ipnet.String())
			resp.YourIPAddr = ipnet.IP
			resp.Options.Update(dhcpv4.OptIPAddressLeaseTime(leaseTime))
			resp.Options.Update(dhcpv4.OptSubnetMask(ipnet.Mask))
			resp.Options.Update(dhcpv4.OptRouter(val.pool.GatewayIP()))
			val.ipReservations.allocation[req.ClientHWAddr.String()] = &ipreservation{
				address:    ipnet,
				state:      committed,
				MacAddress: req.ClientHWAddr.String(),
				Hostname:   req.HostName(),
				expiry:     time.Now().Add(leaseTime),
			}
			pluginHdl.backend.RecordAllocation(map[string]string{"vrfName": string(vrfName), "circuitID": string(circuitID)}, &allocationRecord{IP: resp.YourIPAddr, MacAddress: req.ClientHWAddr.String(), Hostname: req.HostName(), Expiry: val.ipReservations.allocation[req.ClientHWAddr.String()].expiry})
		}

	}
	return nil
}

func handleRelease(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) error {
	relayAgentInfo := req.RelayAgentInfo()
	if relayAgentInfo != nil {
		circuitID := relayAgentInfo.Get(dhcpv4.AgentCircuitIDSubOption)
		vrfName := relayAgentInfo.Get(dhcpv4.VirtualSubnetSelectionSubOption)
		if len(vrfName) > 1 {
			vrfName = vrfName[1:]
		}

		val, err := getRange(string(vrfName), string(circuitID))
		if err != nil {
			return fmt.Errorf("handleRequest::unknown vrf %s circuiId %s", string(vrfName), string(circuitID))
		}

		val.ipReservations.Lock()
		defer val.ipReservations.Unlock()
		if ip, ok := val.ipReservations.allocation[req.ClientHWAddr.String()]; ok { // We have allocated an ip to this client. Release it
			log.Debugf("handleRelease::release :: for vrf %s circuitID %s ip %s", string(vrfName), string(circuitID), ip.address.String())
			val.ipReservations.allocation[req.ClientHWAddr.String()].state = pending
			val.pool.Free(ip.address)
			delete(val.ipReservations.allocation, req.ClientHWAddr.String())
			pluginHdl.backend.ReleaseAllocation(map[string]string{"vrfName": string(vrfName), "circuitID": string(circuitID)}, ip.address.IP, req.ClientHWAddr.String())
		}
		//Silently ignore the release request if there is no active reservation for this client
	}
	return nil
}

func handleDecline(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) error {
	relayAgentInfo := req.RelayAgentInfo()
	if relayAgentInfo != nil {
		circuitID := relayAgentInfo.Get(dhcpv4.AgentCircuitIDSubOption)
		vrfName := relayAgentInfo.Get(dhcpv4.VirtualSubnetSelectionSubOption)
		if len(vrfName) > 1 {
			vrfName = vrfName[1:]
		}

		val, err := getRange(string(vrfName), string(circuitID))
		if err != nil {
			return fmt.Errorf("handleRequest::unknown vrf %s circuiId %s", string(vrfName), string(circuitID))
		}

		val.ipReservations.Lock()
		defer val.ipReservations.Unlock()
		if ip, ok := val.ipReservations.allocation[req.ClientHWAddr.String()]; ok { // We have allocated an ip to this client. Release it
			log.Debugf("handleRelease::release :: for vrf %s circuitID %s ip %s", string(vrfName), string(circuitID), ip.address.String())
			val.ipReservations.allocation[req.ClientHWAddr.String()].state = pending
			val.pool.Free(ip.address)
			delete(val.ipReservations.allocation, req.ClientHWAddr.String())
			pluginHdl.backend.ReleaseAllocation(map[string]string{"vrfName": string(vrfName), "circuitID": string(circuitID)}, ip.address.IP, req.ClientHWAddr.String())
		}

	}
	return nil

}

// This function does not lock anything and assumes it is the only one operating on a range
func getRange(vrfName string, circuitID string) (*allocations, error) {
	backendKey := map[string]string{
		"vrfName":   string(vrfName),
		"circuitID": string(circuitID),
	}
	val, ok := pluginHdl.ranges[string(vrfName)+string(circuitID)]
	if ok { // We have neveer seen this vrf,vlan before. Lets check if our backend can retrieve this
		return val, nil

	}
	record, err := pluginHdl.backend.GetRange(backendKey)
	if err != nil {
		return nil, fmt.Errorf("unknown vrf %s circuiId %s", string(vrfName), string(circuitID))
	}

	prefixLen, _ := record.CIDRBlock.Mask.Size()
	count := binary.BigEndian.Uint32(record.EndIP.To4()) - binary.BigEndian.Uint32(record.StartIP.To4()) + 1
	iprange, err := NewIPv4Range(record.StartIP, record.EndIP, record.Gateway, count, uint32(prefixLen))
	if err != nil {
		return nil, fmt.Errorf("unable to create range for vrf %s circuiId %s,err", string(vrfName), string(circuitID), err)

	}

	val = &allocations{
		pool: iprange,
		ipReservations: &ipallocations{
			allocation: make(map[string]*ipreservation),
		},
	}
	for _, allocatedIPs := range record.records {
		if time.Now().After(allocatedIPs.Expiry) {
			// Lease expired while we were down
			pluginHdl.backend.ReleaseAllocation(backendKey, allocatedIPs.IP, allocatedIPs.MacAddress)
			continue
		}
		val.pool.AllocateIP(net.IPNet{IP: allocatedIPs.IP, Mask: record.CIDRBlock.Mask})
		val.ipReservations.allocation[allocatedIPs.MacAddress] = &ipreservation{address: net.IPNet{IP: allocatedIPs.IP, Mask: record.CIDRBlock.Mask},
			state:      committed,
			MacAddress: allocatedIPs.MacAddress,
			Hostname:   allocatedIPs.Hostname,
			expiry:     allocatedIPs.Expiry,
		}

	}

	pluginHdl.ranges[string(vrfName)+string(circuitID)] = val
	return val, nil

}

func handleExpiredLeases() {
	pluginHdl.Lock()
	defer pluginHdl.Unlock()

	for k, v := range pluginHdl.ranges {
		v.ipReservations.Lock()
		for k1, v1 := range v.ipReservations.allocation {
			if time.Now().After(v1.expiry) {
				log.Debugf("handleExpiredLeases::release :: for vrf %s circuitID %s ip %s", string(k), string(k1), v1.address.String())
				v.ipReservations.allocation[k1].state = pending
				v.pool.Free(v1.address)
				delete(v.ipReservations.allocation, k1)
				pluginHdl.backend.ReleaseAllocation(map[string]string{"vrfName": string(k), "circuitID": string(k1)}, v1.address.IP, v1.MacAddress)
			}
		}
		v.ipReservations.Unlock()
	}
}
