package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/moycat/shiba/model"
	"github.com/moycat/shiba/util"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// initSelf populates the IP address and pod CIDRs of the current node.
func (shiba *Shiba) initSelf() error {
	ctx, cancel := context.WithTimeout(context.Background(), shiba.apiTimeout)
	defer cancel()
	node, err := shiba.client.CoreV1().Nodes().Get(ctx, shiba.nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node [%s]: %w", shiba.nodeName, err)
	}
	// Find the first IPv6 internal address of the node.
	shiba.nodeIP = util.FindNodeIPv6(node)
	if len(shiba.nodeIP) == 0 {
		return fmt.Errorf("node [%s] does not have an ipv6 address", shiba.nodeName)
	}
	log.Debugf("node [%s] has ip [%s]", shiba.nodeName, shiba.nodeIP)
	// Find the pod CIDRs of the node.
	shiba.nodePodCIDRs, err = util.ParseNodePodCIDRs(node)
	if err != nil {
		return fmt.Errorf("failed to parse pod cidrs of node [%s]: %w", shiba.nodeName, err)
	}
	if len(shiba.nodePodCIDRs) == 0 {
		return fmt.Errorf("node [%s] does not have a pod CIDR", shiba.nodeName)
	}
	log.Infof("node [%s] has pod cidrs %v", shiba.nodeName, shiba.nodePodCIDRs)
	return nil
}

// initCluster gets the cluster information from kubeadm config map if not provided.
func (shiba *Shiba) initCluster() error {
	if len(shiba.clusterPodCIDRs) > 0 {
		log.Infof("cluster has privided pod cidrs %v", util.FormatIPNets(shiba.clusterPodCIDRs))
		return nil
	}
	ctx, cancel := shiba.getAPIContext()
	defer cancel()
	kubeadmConfig, err := shiba.client.CoreV1().ConfigMaps("kube-system").Get(ctx, "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cluster pod cidrs not provided, and failed to get kubeadm config")
	}
	clusterConfig := &model.KubeadmClusterConfiguration{}
	if err := yaml.Unmarshal([]byte(kubeadmConfig.Data["ClusterConfiguration"]), clusterConfig); err != nil {
		return fmt.Errorf("failed to unmarshal kubeadm cluster config: %w", err)
	}
	podSubnet := clusterConfig.Networking.PodSubnet
	if len(podSubnet) == 0 {
		return fmt.Errorf("kubeadm cluster config has empty pod subnet: %w", err)
	}
	podCIDRs, err := util.ParseIPNets(strings.Split(podSubnet, ","))
	if err != nil {
		return fmt.Errorf("failed to parse pod subnet from kubeadm config: %w", err)
	}
	if len(podCIDRs) == 0 {
		return fmt.Errorf("kubeadm cluster config has empty pod subnet")
	}
	shiba.clusterPodCIDRs = podCIDRs
	log.Infof("cluster has pod cidrs %v by kubeadm", shiba.clusterPodCIDRs)
	return nil
}
