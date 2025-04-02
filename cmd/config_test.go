package main

import (
	"flag"
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func Test_parseConfig(t *testing.T) {
	cfg := newConfig()
	set := flag.NewFlagSet("", flag.ExitOnError)
	cfg.InitFlags(set)
	err := set.Parse([]string{"--ip6tnl-mtu=1500", "--node-name=hello"})
	assert.NilError(t, err)
	err = cfg.Validate()
	assert.NilError(t, err)
	assert.Equal(t, cfg.IP6tnlMTU, 1500)
	assert.Equal(t, cfg.NodeName, "hello")
}

func Test_parseConfig_from_env(t *testing.T) {
	os.Setenv("SHIBA_IP6TNLMTU", "1500")
	os.Setenv("SHIBA_NODENAME", "hello")
	cfg := newConfig()
	set := flag.NewFlagSet("", flag.ExitOnError)
	cfg.InitFlags(set)
	err := set.Parse(nil)
	assert.NilError(t, err)
	err = cfg.Validate()
	assert.NilError(t, err)
	assert.Equal(t, cfg.IP6tnlMTU, 1500)
	assert.Equal(t, cfg.NodeName, "hello")
}
