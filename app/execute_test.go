package app

import (
	"net"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/moycat/shiba/model"
)

func TestShiba_createIp6tnl(t *testing.T) {
	s := &Shiba{
		ip6tnlMTU: 1500,
	}
	link, err := s.createIp6tnl("hello", &model.Node{
		IP: net.ParseIP("2605:340:cd52:100:39a:464d:c85c:e08a4"),
	})
	assert.NilError(t, err)
	assert.Equal(t, "hello", link.Name)
	assert.Equal(t, 1500, link.Attrs().MTU)

}
