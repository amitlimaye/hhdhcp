package hhdhcp

import (
	"net"
	"sync"
	"time"

	"github.com/coredhcp/coredhcp/plugins"
)

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

type persistentBackend struct {
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
	ipReservations *ipallocations
}

type reservationState uint32

const (
	unassigned reservationState = iota
	pending    reservationState = 1
	committed  reservationState = 2
)

type ipreservation struct {
	address    net.IPNet
	MacAddress string
	expiry     time.Time
	Hostname   string
	state      reservationState
}

type ipallocations struct {
	allocation map[string]*ipreservation
	sync.RWMutex
}
type pluginState struct {
	ranges  map[string]*allocations //(vrfName.circuitID) -> allocations (In memory on bootup we need to populate this from kubernetes)
	backend RecordBackend           // Final source of truth. This might be implemented as a kubernetes backend.
	sync.RWMutex
}
