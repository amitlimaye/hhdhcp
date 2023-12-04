package hhdhcp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/bits-and-blooms/bitset"
)

// This is a range of IP Addresses that is managed as a unit.

type ipv4range struct {
	Start  uint32
	End    uint32
	Mask   net.IPMask
	Count  int
	bitmap *bitset.BitSet
	sync.Mutex
}

func NewIPv4Range(start, end net.IP, count int, prefixLen uint32) (*ipv4range, error) {
	if start.To4() == nil || end.To4() == nil {
		return nil, fmt.Errorf("invalid IPv4 addresses given to create the range: [%s,%s]", start, end)
	}
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}
	if binary.BigEndian.Uint32(start.To4()) > binary.BigEndian.Uint32(end.To4()) {
		return nil, errors.New("no IPs in the given range to allocate")
	}
	if prefixLen > 32 {
		return nil, errors.New("prefix Length must be less than 32")
	}

	r := &ipv4range{
		Start:  binary.BigEndian.Uint32(start.To4()),
		End:    binary.BigEndian.Uint32(end.To4()),
		Count:  count,
		Mask:   net.CIDRMask(int(prefixLen), 32),
		bitmap: bitset.New(uint(binary.BigEndian.Uint32(end.To4()) - binary.BigEndian.Uint32(start.To4()) + 1)),
	}
	if r.Start-r.End+1 != uint32(count) {
		return nil, errors.New("count does not match range")
	}

	return r, nil
}

// Allocate allocates the next availabe IP in range
func (r *ipv4range) Allocate() (net.IPNet, error) {
	r.Lock()
	defer r.Unlock()

	var next uint
	// Then any available address
	avail, ok := r.bitmap.NextClear(0)
	if !ok {
		return net.IPNet{}, errors.New("no IPs in the range to allocate")
	}

	next = avail

	r.bitmap.Set(next)
	return net.IPNet{
		IP:   r.toIP(uint32(next)),
		Mask: r.Mask,
	}, nil

}

// AllocateIP allocates a specific IP in the range if it is available. else return the next available IP
func (r *ipv4range) AllocateIP(ip net.IPNet) (net.IPNet, error) {
	mask := net.CIDRMask(32, 32)

	if ip.IP.To4() != nil {
		mask = ip.Mask
	}

	r.Lock()
	defer r.Unlock()
	// first try to get the exact ip
	if !r.bitmap.Test(uint(binary.BigEndian.Uint32(ip.IP.To4()))) {
		return net.IPNet{
			IP:   r.toIP(uint32(binary.BigEndian.Uint32(ip.IP.To4()))),
			Mask: mask,
		}, nil
	}
	// Allocate the next available IP
	return r.Allocate()
}

// Free release the ip address back to the range
func (r *ipv4range) Free(ip net.IPNet) error {
	if !r.bitmap.Test(uint(binary.BigEndian.Uint32(ip.IP.To4()))) {
		return errors.New("ip address is not allocated in this range")
	}
	r.bitmap.Clear(uint(binary.BigEndian.Uint32(ip.IP.To4()))) // IP released
	return nil
}

func (r *ipv4range) toIP(offset uint32) net.IP {
	if offset > r.End-r.Start {
		panic("BUG: offset out of bounds")
	}
	ip := make(net.IP, net.IPv4len)
	binary.BigEndian.PutUint32(ip, uint32(r.Start)+offset)
	return ip
}

func (r *ipv4range) toOffset(ip net.IP) (uint, error) {
	ipaddr := binary.BigEndian.Uint32(ip.To4())
	if ipaddr < r.Start || ipaddr > r.End {
		return 0, errors.New("IP address out of range")
	}

	return uint(ipaddr - r.Start), nil
}
