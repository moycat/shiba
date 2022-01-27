package util

import (
	"fmt"
	"net"
	"sort"
)

// IsV4 checks if a net.IP is an IPv4 address.
func IsV4(ip net.IP) bool {
	return ip.To4() != nil
}

// IsV6 checks if a net.IP is an IPv6 address.
func IsV6(ip net.IP) bool {
	return ip.To4() == nil && ip.To16() != nil
}

// ParseIPNets returns the parsed subnets.
func ParseIPNets(ipNetStrings []string) ([]*net.IPNet, error) {
	var ipNets []*net.IPNet
	for _, ipNetString := range ipNetStrings {
		_, ipNet, err := net.ParseCIDR(ipNetString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse [%s]: %w", ipNetString, err)
		}
		ipNets = append(ipNets, ipNet)
	}
	// Sort for later comparison.
	sort.Slice(ipNets, func(i, j int) bool {
		return ipNets[i].String() < ipNets[j].String()
	})
	return ipNets, nil
}

// FormatIPNets returns the string representation of a slice of net.IPNet.
func FormatIPNets(ipNets []*net.IPNet) string {
	var netStrings []string
	for _, ipNet := range ipNets {
		netStrings = append(netStrings, ipNet.String())
	}
	return fmt.Sprintf("%v", netStrings)
}
