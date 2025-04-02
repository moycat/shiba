package main

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func Test_parseConfig(t *testing.T) {
	originArgs := os.Args
	defer func() {
		os.Args = originArgs
	}()
	os.Args = []string{"test", "--ip6tnl-mtu=1500"}
	cfg := parseConfig()
	assert.Equal(t, cfg.IP6tnlMTU, 1500)
}

func Test_parseConfig_from_env(t *testing.T) {
	os.Setenv("SHIBA_IP6TNL_MTU", "1500")
	defer os.Unsetenv("SHIBA_IP6TNL_MTU")
	cfg := parseConfig()
	assert.Equal(t, cfg.IP6tnlMTU, 1500)
}
