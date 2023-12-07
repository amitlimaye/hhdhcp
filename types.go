package hhdhcp

import (
	"net"
	"sync"
	"time"

	"github.com/coredhcp/coredhcp/plugins"
)

// DHCPSubnetSpec defines the desired state of DHCPSubnet
// type DHCPSubnetSpec struct {
// 	Subnet    string `json:"subnet"`    // e.g. vpc-0/default (vpc name + vpc subnet name)
// 	CIDRBlock string `json:"cidrBlock"` // e.g. 10.10.10.0/24
// 	Gateway   string `json:"gateway"`   // e.g. 10.10.10.1
// 	StartIP   string `json:"startIP"`   // e.g. 10.10.10.10
// 	EndIP     string `json:"endIP"`     // e.g. 10.10.10.99
// 	VRF       string `json:"vrf"`       // e.g. VrfVvpc-1 as it's named on switch
// 	CircuitID string `json:"circuitID"` // e.g. Vlan1000 as it's named on switch
// }

// // DHCPSubnetStatus defines the observed state of DHCPSubnet
// type DHCPSubnetStatus struct {
// 	AllocatedIPs map[string]DHCPAllocatedIP `json:"allocatedIPs,omitempty"`
// }

// type DHCPAllocatedIP struct {
// 	Expiry   metav1.Time `json:"expiry"`
// 	MAC      string      `json:"mac"`
// 	Hostname string      `json:"hostname"` // from dhcp request
// }

type rangeRecord struct {
	StartIP net.IP
	EndIP   net.IP
	//count     int
	Subnet    string
	Gateway   net.IP
	CIDRBlock net.IPNet
	VRF       string
	CircuitID string
	records   []*allocationRecord
}

type allocationRecord struct {
	IP         net.IP
	MacAddress string
	Hostname   string
	Expiry     time.Time
}

type recordBackend struct {
	subnets map[string]*rangeRecord // This is temporary and we should be using a kubernetes backend
}

var Plugin = plugins.Plugin{
	Name:   "hhdhcp",
	Setup6: nil,
	Setup4: setuphhdhcp4,
}

var leaseTime = time.Duration(3600 * time.Second)

type allocations struct {
	pool IPv4Allocator
	// Offers that have been made but we have not seen a request for. ip->mac address. This is temporary
	// while we wait for dhcprequest. Sync to kubernetes backend and destroy this state.
	pending *pendingAllocations
}

type pendingAllocations struct {
	allocation map[string]string
	sync.RWMutex
}
type pluginState struct {
	ranges  map[string]*allocations //(vrfName.circuitID) -> allocations (In memory on bootup we need to populate this from kubernetes)
	backend RecordBackend           // Final source of truth. This might be implemented as a kubernetes backend.
	sync.RWMutex
}
