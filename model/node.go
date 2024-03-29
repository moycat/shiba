package model

import (
	"net"
)

// NodeMap is the map of Node.
type NodeMap map[string]*Node

// Node is a parsed K8s node.
type Node struct {
	Name     string
	IP       net.IP // IPv6 only.
	PodCIDRs []*net.IPNet
	Tunnel   string
}

// DiffersFrom checks if the node is different from another node, except for the tunnel name.
// The order of the pod CIDRs may be altered.
func (n *Node) DiffersFrom(nn *Node) bool {
	if n == nil && nn == nil {
		return false
	}
	if (n == nil) != (nn == nil) {
		return true
	}
	if n.Name != nn.Name {
		return true
	}
	if !n.IP.Equal(nn.IP) {
		return true
	}
	if len(n.PodCIDRs) != len(nn.PodCIDRs) {
		return true
	}
	for i := range n.PodCIDRs {
		if n.PodCIDRs[i].String() != nn.PodCIDRs[i].String() {
			return true
		}
	}
	return false
}
