package hhdhcp

import (
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DHCPSubnetSpec defines the desired state of DHCPSubnet
type DHCPSubnetSpec struct {
	Subnet    string `json:"subnet"`    // e.g. vpc-0/default (vpc name + vpc subnet name)
	CIDRBlock string `json:"cidrBlock"` // e.g. 10.10.10.0/24
	Gateway   string `json:"gateway"`   // e.g. 10.10.10.1
	StartIP   string `json:"startIP"`   // e.g. 10.10.10.10
	EndIP     string `json:"endIP"`     // e.g. 10.10.10.99
	VRF       string `json:"vrf"`       // e.g. VrfVvpc-1 as it's named on switch
	CircuitID string `json:"circuitID"` // e.g. Vlan1000 as it's named on switch
}

// DHCPSubnetStatus defines the observed state of DHCPSubnet
type DHCPSubnetStatus struct {
	AllocatedIPs map[string]DHCPAllocatedIP `json:"allocatedIPs,omitempty"`
}

type DHCPAllocatedIP struct {
	Expiry   metav1.Time `json:"expiry"`
	MAC      string      `json:"mac"`
	Hostname string      `json:"hostname"` // from dhcp request
}

type k8sBackend struct {
	subnets     map[string]*DHCPSubnetSpec // This is temporary and we should be using a kubernetes backend
	allocatedIP DHCPSubnetStatus
}

func NewKubernetesBackend() RecordBackend {
	// Lazy populate as we get requests for a vrf or a range
	return &k8sBackend{
		subnets: make(map[string]*DHCPSubnetSpec),
		allocatedIP: DHCPSubnetStatus{
			AllocatedIPs: make(map[string]DHCPAllocatedIP),
		},
	}
}

func (k *k8sBackend) GetRange(key map[string]string) (*rangeRecord, error) {
	// retrieve from kubernetes here the DHCPSubnetSpec and Status convert it to rangeRecord
	return &rangeRecord{}, nil
}

func (k *k8sBackend) RecordAllocation(key map[string]string, alloc *allocationRecord) error {
	// Save Allocation Record in DHCPStatus
	return nil
}

func (k *k8sBackend) ReleaseAllocation(key map[string]string, ip net.IP, macAddress string) error {
	// Delete Allocation Record in DHCPStatus
	return nil
}
