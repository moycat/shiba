package app

import (
	"github.com/moycat/shiba/model"
	"github.com/moycat/shiba/util"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// processEvent parses the node event and updates the node map if necessary.
func (shiba *Shiba) processEvent(event watch.Event) {
	log.Debugf("received an event of type [%s]", event.Type)
	node, ok := event.Object.(*corev1.Node)
	if !ok {
		log.Warningf("received an event of type [%s] with unexpected object type [%T]",
			event.Type, event.Object.GetObjectKind().GroupVersionKind())
		return
	}
	if node.Name == shiba.nodeName {
		log.Debugf("ignoring an [%s] event of myself", event.Type)
		return
	}
	var needFiring bool
	switch event.Type {
	case watch.Added:
		needFiring = shiba.addNode(node)
	case watch.Modified:
		needFiring = shiba.updateNode(node)
	case watch.Deleted:
		needFiring = shiba.deleteNode(node)
	default:
		log.Warningf("received an unwanted event of type [%s] of node [%s]", event.Type, node.Name)
		return
	}
	if needFiring {
		log.Infof("processed %s event of node [%s]", event.Type, node.Name)
		select {
		case shiba.fireCh <- struct{}{}:
		default:
		}
	} else {
		log.Debugf("processed %s event of node [%s] witch didn't trigger firing", event.Type, node.Name)
	}
}

func (shiba *Shiba) addNode(node *corev1.Node) bool {
	nodeMap := shiba.cloneNodeMap()
	if _, ok := nodeMap[node.Name]; ok {
		log.Debugf("adding a existing node [%s]", node.Name)
		return shiba.updateNode(node)
	}
	nodeIP := util.FindNodeIPv6(node)
	if nodeIP == nil {
		log.Errorf("failed to find ipv6 address of node [%s]", node.Name)
		return false
	}
	nodePodCIDRs, err := util.ParseNodePodCIDRs(node)
	if err != nil {
		log.Errorf("failed to parse pod cidrs of node [%s]: %v", node.Name, err)
		return false
	}
	parsedNode := &model.Node{
		Name:     node.Name,
		IP:       nodeIP,
		PodCIDRs: nodePodCIDRs,
		Tunnel:   tunnelPrefix + util.NewUID(),
	}
	nodeMap[node.Name] = parsedNode
	shiba.saveNodeMap(nodeMap)
	shiba.dumpNodeMap()
	log.Debugf("added node [%s] and dumped map", node.Name)
	return true
}

func (shiba *Shiba) deleteNode(node *corev1.Node) bool {
	nodeMap := shiba.cloneNodeMap()
	if _, ok := nodeMap[node.Name]; !ok {
		log.Warningf("deleting node [%s] which is not present", node.Name)
		return false
	}
	delete(nodeMap, node.Name)
	shiba.saveNodeMap(nodeMap)
	shiba.dumpNodeMap()
	log.Debugf("deleted node [%s] and dumped map", node.Name)
	return true
}

func (shiba *Shiba) updateNode(node *corev1.Node) bool {
	nodeMap := shiba.cloneNodeMap()
	oldNode, ok := nodeMap[node.Name]
	if !ok {
		log.Warningf("updating node [%s] which is not present", node.Name)
		return shiba.addNode(node)
	}
	nodeIP := util.FindNodeIPv6(node)
	if nodeIP == nil {
		log.Errorf("failed to find ipv6 address of node [%s]", node.Name)
		return false
	}
	nodePodCIDRs, err := util.ParseNodePodCIDRs(node)
	if err != nil {
		log.Errorf("failed to parse pod cidrs of node [%s]: %v", node.Name, err)
		return false
	}
	parsedNode := &model.Node{
		Name:     node.Name,
		IP:       nodeIP,
		PodCIDRs: nodePodCIDRs,
		Tunnel:   tunnelPrefix + util.NewUID(),
	}
	if !parsedNode.DiffersFrom(oldNode) {
		log.Debugf("node [%s] have no actual updates", node.Name)
		return false
	}
	nodeMap[node.Name] = parsedNode
	shiba.saveNodeMap(nodeMap)
	shiba.dumpNodeMap()
	log.Debugf("updated node [%s] and saved map", node.Name)
	return true
}
