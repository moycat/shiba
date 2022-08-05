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
	"github.com/moycat/shiba/util"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	shiba.validateNodeMap()
}

func (shiba *Shiba) validateNodeMap() {
	ctx, cancel := context.WithTimeout(context.Background(), shiba.apiTimeout)
	defer cancel()
	nodes, err := shiba.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Errorf("failed to list nodes: %v", err)
		shiba.nodeMap = nil // Drop the map since we can't validate it.
		return
	}
	nodeMap := make(map[string]corev1.Node, nodes.Size())
	for _, node := range nodes.Items {
		nodeMap[node.Name] = node
	}
	var badNodes []string
	for name, node := range shiba.nodeMap {
		n, ok := nodeMap[name]
		if !ok {
			log.Warningf("node [%s] loaded from cache doesn't exist, removing", name)
			badNodes = append(badNodes, name)
			continue
		}
		nodeIP := util.FindNodeIPv6(&n)
		if nodeIP == nil {
			log.Warningf("node [%s] loaded from cache no longer has an IPv6 address, removing", name)
			badNodes = append(badNodes, name)
			continue
		}
		nodePodCIDRs, err := util.ParseNodePodCIDRs(&n)
		if err != nil {
			log.Warningf("failed to parse pod cidrs of node [%s]: %v", name, err)
			badNodes = append(badNodes, name)
			continue
		}
		if node.DiffersFrom(&model.Node{Name: name, IP: nodeIP, PodCIDRs: nodePodCIDRs}) {
			log.Warningf("node [%s] IP or pod CIDRs changed, removing", name)
			log.Debugf("IP: [%v]/[%v], CIDRs:%s/%s", node.IP, nodeIP, node.PodCIDRs, util.FormatIPNets(nodePodCIDRs))
			badNodes = append(badNodes, name)
			continue
		}
	}
	for _, badNode := range badNodes {
		delete(shiba.nodeMap, badNode)
	}
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
