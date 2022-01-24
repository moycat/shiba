package app

import (
	"net"
	"strings"
	"time"

	"github.com/moycat/shiba/model"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func (shiba *Shiba) execute(stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		case <-shiba.fireCh:
			log.Debugf("fire command received, waiting for %v", executeGracePeriod)
			time.Sleep(executeGracePeriod)
			select {
			case <-shiba.fireCh:
			default:
			}
			nodeMap := shiba.cloneNodeMap()
			shiba.syncTunnels(nodeMap)
			shiba.syncRoutes(nodeMap)
		}
	}
}

func (shiba *Shiba) syncTunnels(nodeMap model.NodeMap) {
	log.Infof("syncing tunnels")
	linkMap := make(map[string]*netlink.Ip6tnl)
	tunnelMap := make(model.NodeMap, len(nodeMap)) // Tunnel name -> node.
	for _, node := range nodeMap {
		tunnelMap[node.Tunnel] = node
	}

	log.Debugf("examining existing tunnels")
	links, err := netlink.LinkList()
	if err != nil {
		log.Errorf("failed to list links: %v", err)
	}
	for _, link := range links {
		link, ok := link.(*netlink.Ip6tnl)
		if !ok {
			continue
		}
		linkName := link.Attrs().Name
		if strings.HasPrefix(linkName, tunnelPrefix) {
			if _, ok := tunnelMap[linkName]; ok {
				linkMap[linkName] = link
			} else {
				log.Debugf("removing dangling tunnel %s", linkName)
				if err := netlink.LinkDel(link); err != nil {
					log.Errorf("failed to delete tunnel: %v", err)
				}
			}
		}
	}

	log.Debugf("applying tunnels")
	for linkName, node := range tunnelMap {
		link, ok := linkMap[linkName]
		if ok {
			if link.LinkAttrs.Flags|net.FlagUp > 0 && link.Local.Equal(shiba.nodeIP) && link.Remote.Equal(node.IP) {
				log.Debugf("tunnel [%s] to node [%s] is up and in sync, skipping", linkName, node.Name)
				continue
			}
			log.Debugf("tunnel [%s] to node [%s] out of sync, recreating", linkName, node.Name)
			if err := netlink.LinkDel(link); err != nil {
				log.Errorf("failed to delete stale tunnel [%s] to node [%s]: %v", linkName, node.Name, err)
				continue
			}
		}
		log.Infof("creating tunnel [%s] to node [%s] (%v)", linkName, node.Name, node.IP)
		link = &netlink.Ip6tnl{
			LinkAttrs: netlink.LinkAttrs{Name: linkName},
			Local:     shiba.nodeIP,
			Remote:    node.IP,
		}
		if err := netlink.LinkAdd(link); err != nil {
			log.Errorf("failed to create tunnel [%s]: %v", linkName, err)
			continue
		}
		if err := netlink.LinkSetUp(link); err != nil {
			log.Errorf("failed to bring tunnel [%s] up: %v", linkName, err)
			continue
		}
	}
}

func (shiba *Shiba) syncRoutes(nodeMap model.NodeMap) {
	log.Infof("syncing routes")
	for _, node := range nodeMap {
		link, err := netlink.LinkByName(node.Tunnel)
		if err != nil {
			log.Errorf("failed to get tunnel [%s] to node [%s]: %v", node.Tunnel, node.Name, err)
			continue
		}
		log.Debugf("checking routes of tunnel [%s] to node [%s]", node.Tunnel, node.Name)
		routeMap := make(map[string]*net.IPNet)
		for _, ipNet := range node.PodCIDRs {
			routeMap[ipNet.String()] = ipNet
		}
		routes, err := netlink.RouteList(link, netlink.FAMILY_ALL)
		if err != nil {
			log.Errorf("failed to list routes of tunnel [%s] to node [%s]: %v", node.Tunnel, node.Name, err)
			continue
		}
		for _, route := range routes {
			if route.Dst != nil && route.Src == nil && len(route.Gw) == 0 && routeMap[route.Dst.String()] != nil {
				log.Debugf("route to [%s] on node [%s] via tunnel [%s] exists",
					route.Dst.String(), node.Name, node.Tunnel)
				delete(routeMap, route.Dst.String())
				continue
			}
			log.Debugf("deleting unexpected route on tunnel [%s]: %v", node.Tunnel, route)
			if err := netlink.RouteDel(&route); err != nil {
				log.Errorf("failed to delete route on tunnel [%s]: %v", node.Tunnel, err)
				continue
			}
		}
		for _, routeToAdd := range routeMap {
			log.Infof("adding route to [%s] on node [%s] via tunnel [%s]",
				routeToAdd.String(), node.Name, node.Tunnel)
			route := netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst:       routeToAdd,
			}
			if err := netlink.RouteAdd(&route); err != nil {
				log.Errorf("failed to add route to [%s] on node [%s] via tunnel [%s]: %v",
					routeToAdd.String(), node.Name, node.Tunnel, err)
				continue
			}
		}
	}
}
