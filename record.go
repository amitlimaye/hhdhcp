package hhdhcp

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"
)

func NewBackend() RecordBackend {
	// Sync from kubernetes backend if it exists

	r := &persistentBackend{
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
	// Debug routine
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for {
			select {
			case <-ticker.C:
				log.Infof("---Recorded Leases: %d----", len(r.subnets["VrfDhcp"].records))
			}

		}
	}()
	return r
	log.Infof("Defined subnets: %v", r.subnets["VrfDhcp"])
	return r
}

func (r *persistentBackend) GetRange(meta map[string]string) (*rangeRecord, error) {
	if val, ok := meta["vrfName"]; ok {
		return r.subnets[val], nil
	}
	return nil, errors.New("no range found")
}

func (r *persistentBackend) RecordAllocation(meta map[string]string, alloc *allocationRecord) error {
	if val, ok := meta["vrfName"]; ok {
		// if allocation already exists skip allocation

		if record, ok := r.subnets[val]; ok {
			for _, rec := range record.records {
				if rec.IP.Equal(alloc.IP) {
					return nil
				}
			}
			record.records = append(record.records, alloc)
			r.subnets[val] = record
			return nil
		}
	}
	return nil
}

func (r *persistentBackend) ReleaseAllocation(key map[string]string, ip net.IP, macAddress string) error {
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
