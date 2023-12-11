package hhdhcp

import "net"

type IPv4Allocator interface {
	AllocateIP(hint net.IPNet) (net.IPNet, error)
	Allocate() (net.IPNet, error)
	Free(net.IPNet) error
	GatewayIP() net.IP
}

type RecordBackend interface {
	GetRange(key map[string]string) (*rangeRecord, error)
	RecordAllocation(key map[string]string, alloc *allocationRecord) error
	ReleaseAllocation(key map[string]string, ip net.IP, macAddress string) error
}
