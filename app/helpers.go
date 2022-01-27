package app

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"syscall"

	"github.com/moycat/shiba/model"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func (shiba *Shiba) getAPIContext() (context.Context, func()) {
	if shiba.apiTimeout > 0 {
		return context.WithTimeout(context.Background(), shiba.apiTimeout)
	}
	return context.WithCancel(context.Background())
}

func (shiba *Shiba) isTunnelInSync(link *netlink.Ip6tnl, node *model.Node) bool {
	if link.LinkAttrs.Flags|net.FlagUp == 0 {
		log.Debugf("tunnel [%s] is not up", link.Name)
		return false
	}
	if !link.Local.Equal(shiba.nodeIP) || !link.Remote.Equal(node.IP) {
		log.Debugf("tunnel [%s] has bad peer config", link.Name)
		return false
	}
	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		log.Errorf("failed to get addr list of tunnel [%s]: %v", link.Name, err)
		return false
	}
	addrMap := make(map[string]bool)
	for _, addr := range addrs {
		if addr.Scope != syscall.RT_SCOPE_UNIVERSE {
			continue
		}
		if ones, bits := addr.Mask.Size(); ones != bits {
			log.Debugf("tunnel [%s] has non-single address [%v]", link.Name, addr.IPNet.String())
		}
		addrMap[addr.IP.String()] = true
	}
	if !reflect.DeepEqual(addrMap, shiba.nodeGatewayMap) {
		log.Debugf("tunnel [%s] has bad ips: %v", link.Name, addrMap)
		return false
	}
	return true
}

func (shiba *Shiba) loadNodeMap() {
	path := filepath.Join(os.TempDir(), nodeMapFilename)
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Errorf("failed to open node map file [%s] for reading: %v", path, err)
		}
		return
	}
	defer func() { _ = f.Close() }()
	decoder := json.NewDecoder(f)
	nodeMap := make(model.NodeMap)
	if err := decoder.Decode(&nodeMap); err != nil {
		log.Errorf("failed to unmarshal node map file [%s]: %v", path, err)
		return
	}
	shiba.saveNodeMap(nodeMap)
}

func (shiba *Shiba) dumpNodeMap() {
	path := filepath.Join(os.TempDir(), nodeMapFilename)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		log.Errorf("failed to open node map file [%s] for writing: %v", path, err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("failed to close node map file [%s]: %v", path, err)
		}
	}()
	encoder := json.NewEncoder(f)
	nodeMap := shiba.cloneNodeMap()
	if err := encoder.Encode(nodeMap); err != nil {
		log.Errorf("failed to marshal node map to [%s]: %v", path, err)
	}
}

func (shiba *Shiba) cloneNodeMap() model.NodeMap {
	shiba.nodeMapLock.Lock()
	nodeMap := shiba.nodeMap
	shiba.nodeMapLock.Unlock()
	newNodeMap := make(model.NodeMap, len(nodeMap))
	for k, v := range nodeMap {
		node := *v
		newNodeMap[k] = &node
	}
	return newNodeMap
}

func (shiba *Shiba) saveNodeMap(nodeMap model.NodeMap) {
	shiba.nodeMapLock.Lock()
	shiba.nodeMap = nodeMap
	shiba.nodeMapLock.Unlock()
}
