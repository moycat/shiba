package util

import (
	"fmt"
	"net"
	"sort"

	corev1 "k8s.io/api/core/v1"
)

// FindNodeIPv6 returns the first IPv6 address of the node's InternalIPs, nil if not found.
func FindNodeIPv6(node *corev1.Node) net.IP {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			ip := net.ParseIP(address.Address)
			if IsV6(ip) {
				return ip
			}
		}
	}
	return nil
}

// ParseNodePodCIDRs returns the pod CIDRs of the node.
func ParseNodePodCIDRs(node *corev1.Node) ([]*net.IPNet, error) {
	cidrStrings := make(map[string]bool, len(node.Spec.PodCIDRs)+1)
	cidrStrings[node.Spec.PodCIDR] = true
	for _, cidr := range node.Spec.PodCIDRs {
		cidrStrings[cidr] = true
	}
	podCIDRs := make([]*net.IPNet, 0, len(cidrStrings))
	for cidr := range cidrStrings {
		if len(cidr) == 0 {
			continue
		}
		_, podCIDR, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cidr [%s]: %w", cidr, err)
		}
		podCIDRs = append(podCIDRs, podCIDR)
	}
	// Sort for later comparison.
	sort.Slice(podCIDRs, func(i, j int) bool {
		return podCIDRs[i].String() < podCIDRs[j].String()
	})
	return podCIDRs, nil
}
