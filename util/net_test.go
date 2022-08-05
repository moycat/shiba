package util

import (
	"net"
	"testing"

	"gotest.tools/v3/assert"
)

type ipTestCase struct {
	ip net.IP
	ok bool
}

func TestIsV4(t *testing.T) {
	cases := []*ipTestCase{
		{ip: nil, ok: false},
		{ip: []byte("vaala"), ok: false},
		{ip: net.ParseIP("1.2.3.4"), ok: true},
		{ip: []byte{8, 8, 8, 8}, ok: true},
		{ip: []byte{8, 8, 8, 8, 8}, ok: false},
		{ip: net.ParseIP("::1"), ok: false},
		{ip: []byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}, ok: false},
	}
	for _, c := range cases {
		assert.Assert(t, IsV4(c.ip) == c.ok, `IsV4() should return %v for %v, got %v`, c.ok, c.ip, !c.ok)
	}
}

func TestIsV6(t *testing.T) {
	cases := []*ipTestCase{
		{ip: nil, ok: false},
		{ip: []byte("vaala"), ok: false},
		{ip: net.ParseIP("1.2.3.4"), ok: false},
		{ip: []byte{8, 8, 8, 8}, ok: false},
		{ip: []byte{8, 8, 8, 8, 8}, ok: false},
		{ip: net.ParseIP("::1"), ok: true},
		{ip: []byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}, ok: true},
	}
	for _, c := range cases {
		assert.Assert(t, IsV6(c.ip) == c.ok, `IsV6() should return %v for %v, got %v`, c.ok, c.ip, !c.ok)
	}
}

func TestParseIPNets(t *testing.T) {
	nets, err := ParseIPNets(nil)
	if err != nil {
		t.Error(err)
	}
	assert.Assert(t, len(nets) == 0, "unexpected parsed nets: %#v", nets)
	nets, err = ParseIPNets([]string{"192.168.0.0/16", "fdef:1234::/64"})
	assert.NilError(t, err)
	assert.Equal(t, nets[0].String(), "192.168.0.0/16")
	assert.Equal(t, nets[1].String(), "fdef:1234::/64")
	// Should sort.
	nets, err = ParseIPNets([]string{"192.168.0.0/16", "172.16.0.0/12", "10.0.0.0/8"})
	assert.NilError(t, err)
	assert.Equal(t, nets[0].String(), "10.0.0.0/8")
	assert.Equal(t, nets[1].String(), "172.16.0.0/12")
	assert.Equal(t, nets[2].String(), "192.168.0.0/16")
}
