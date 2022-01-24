package util

import (
	"github.com/coreos/go-iptables/iptables"
)

// Tables wraps iptables.IPTables.
type Tables struct {
	*iptables.IPTables
}

var (
	V4tables *Tables
	V6tables *Tables
)

func init() {
	v4tables, _ := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	v6tables, _ := iptables.NewWithProtocol(iptables.ProtocolIPv6)
	V4tables = &Tables{IPTables: v4tables}
	V6tables = &Tables{IPTables: v6tables}
}

// NewChainUnique creates a chain if it doesn't already exist.
func (t *Tables) NewChainUnique(table, chain string) error {
	exists, err := t.ChainExists(table, chain)
	if err != nil {
		return err
	}
	if !exists {
		if err := t.NewChain(table, chain); err != nil {
			return err
		}
	}
	return nil
}

// InsertUnique inserts a rule if it doesn't already exist.
func (t *Tables) InsertUnique(table, chain string, pos int, rulespec ...string) error {
	exists, err := t.Exists(table, chain, rulespec...)
	if err != nil {
		return err
	}
	if !exists {
		return t.Insert(table, chain, pos, rulespec...)
	}
	return nil
}
