package hhdhcp

import (
	"errors"
	"fmt"
	"net"
	"reflect"
)

type RecordBackend interface {
	GetRange(key map[string]string) (*rangeRecord, error)
	RecordAllocation(key map[string]string, alloc *allocationRecord) error
	ReleaseAllocation(key map[string]string, ip net.IP) error
}

func NewBackend() RecordBackend {
	// Sync from kubernetes backend if it exists

	r := &recordBackend{
		subnets: map[string]*rangeRecord{
			"VrfDhcp": {
				StartIP: net.ParseIP("10.10.30.2"),
				EndIP:   net.ParseIP("10.10.30.254"),
				Subnet:  "VrfDhcp/default",
				CIDRBlock: net.IPNet{
					IP:   net.ParseIP("10.10.30.0"),
					Mask: net.CIDRMask(24, 32),
				},
				Gateway:   net.ParseIP("10.10.30.1"),
				VRF:       "VrfDhcp",
				CircuitID: "Vlan3000",
			},
		},
	}
	log.Infof("Defined subnets: %v", r.subnets["VrfDhcp"])
	return r
}

func (r *recordBackend) GetRange(meta map[string]string) (*rangeRecord, error) {
	if val, ok := meta["vrfName"]; ok {
		log.Infof("found vrfName %s %v", val, r.subnets["VrfDhcp"])

		return r.subnets[val], nil
	}
	return nil, errors.New("no range found")
}

func (r *recordBackend) RecordAllocation(meta map[string]string, alloc *allocationRecord) error {
	if val, ok := meta["vrfName"]; ok {
		if record, ok := r.subnets[val]; ok {
			record.records = append(record.records, alloc)
		}
	}
	return nil
}

func (r *recordBackend) ReleaseAllocation(key map[string]string, ip net.IP) error {
	if val, ok := key["vrfName"]; ok {
		if record, ok := r.subnets[val]; ok {
			for i, rec := range record.records {
				if rec.IP.Equal(ip) {
					record.records = append(record.records[:i], record.records[i+1:]...)
					return nil
				} else {
					return fmt.Errorf("record not found %s", ip)
				}
			}
		} else {
			return fmt.Errorf("vrf is not managed by this backend %s", val)
		}

	}
	return fmt.Errorf("vrfName not found in key %v", reflect.ValueOf(key).MapKeys())
}

// func parseRecord(ipnet net.IPNet) (net.IP, net.IP, net.IP, uint32) {
// 	start := make(net.IP, net.IPv4len)
// 	end := make(net.IP, net.IPv4len)
// 	gateway := make(net.IP, net.IPv4len)
// 	var count uint32 = 0
// 	size, max := ipnet.Mask.Size()
// 	if max-size < 3 {
// 		binary.BigEndian.PutUint32(start, binary.BigEndian.Uint32(ipnet.IP)) //first IP
// 		binary.BigEndian.PutUint32(end, binary.BigEndian.Uint32(ipnet.IP)+1) //last IP
// 		count = 1
// 	} else {
// 		binary.BigEndian.PutUint32(gateway, binary.BigEndian.Uint32(ipnet.IP)+1)             //gateway IP
// 		binary.BigEndian.PutUint32(start, binary.BigEndian.Uint32(ipnet.IP)+2)               //first
// 		binary.BigEndian.PutUint32(end, binary.BigEndian.Uint32(ipnet.IP)+(1<<(max-size))-2) // lastIP
// 		count = (1 << (max - size)) - 4
// 	}
// 	return start, end, gateway, count
// }
