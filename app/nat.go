package app

import (
	"fmt"

	"github.com/moycat/shiba/util"
	log "github.com/sirupsen/logrus"
)

// initNAT sets up NAT for cluster pod CIDRs using iptables.
func (shiba *Shiba) initNAT() error {
	addRules := func(tables *util.Tables, chain string, subnets []string) error {
		if err := tables.NewChainUnique("nat", chain); err != nil {
			return fmt.Errorf("failed to create a unique chain: %w", err)
		}
		if err := tables.AppendUnique("nat", chain, "-j", "MASQUERADE"); err != nil {
			return fmt.Errorf("failed to append the nat rule: %w", err)
		}
		for _, subnet := range subnets {
			log.Debugf("adding nat rules for [%s]", subnet)
			// NAT if traffic comes from the subnet.
			if err := tables.AppendUnique("nat", "POSTROUTING", "--src", subnet, "-j", chain); err != nil {
				return fmt.Errorf("failed to redirect outgoing traffic from [%s]: %w", subnet, err)
			}
			// However, skip if traffic goes to the subnet.
			if err := tables.InsertUnique("nat", chain, 1, "--dst", subnet, "-j", "RETURN"); err != nil {
				return fmt.Errorf("failed to add nat exclusion rule for [%s]: %w", subnet, err)
			}
		}
		return nil
	}
	var v4Subnets, v6Subnets []string
	for _, cidr := range shiba.clusterPodCIDRs {
		switch {
		case util.IsV4(cidr.IP):
			v4Subnets = append(v4Subnets, cidr.String())
		case util.IsV6(cidr.IP):
			v6Subnets = append(v6Subnets, cidr.String())
		default:
			return fmt.Errorf("[%s] is neither ipv4 or ipv6 subnet", cidr.String())
		}
	}
	if len(v4Subnets) > 0 {
		if err := addRules(util.V4tables, iptablesChain, v4Subnets); err != nil {
			return fmt.Errorf("failed to setup nat for v4 subnets: %w", err)
		}
		log.Infof("ipv4 nat rules are ready")
	}
	if len(v6Subnets) > 0 {
		if err := addRules(util.V6tables, iptablesChain, v6Subnets); err != nil {
			return fmt.Errorf("failed to setup nat for v6 subnets: %w", err)
		}
		log.Infof("ipv6 nat rules are ready")
	}
	return nil
}
