package app

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/moycat/shiba/model"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	cniConfigName      = "10-shiba.conflist"
	cniNetName         = "shiba-net"
	executeGracePeriod = time.Second
	fireInterval       = time.Minute
	iptablesChain      = "SHIBA"
	nodeMapFilename    = "shiba-node-map"
	tunnelPrefix       = "shiba."
)

// Shiba is the main app.
type Shiba struct {
	client          kubernetes.Interface
	cniConfigPath   string
	clusterPodCIDRs []*net.IPNet
	nodeName        string
	nodeIP          net.IP // IPv6 only.
	nodePodCIDRs    []*net.IPNet
	nodeGateways    []net.IP
	nodeGatewayMap  map[string]bool
	nodeMap         model.NodeMap // When a map reaches here, it's immutable.
	nodeMapLock     sync.Mutex
	fireCh          chan struct{}
	apiTimeout      time.Duration
}

// ShibaOptions specifies the non-essential options for Shiba.
type ShibaOptions struct {
	APITimeout      time.Duration
	ClusterPodCIDRs []*net.IPNet
}

// NewShiba returns a new instance of Shiba.
func NewShiba(client kubernetes.Interface, nodeName, cniConfigPath string, options ShibaOptions) (*Shiba, error) {
	shiba := &Shiba{
		client:          client,
		cniConfigPath:   cniConfigPath,
		nodeName:        nodeName,
		nodeMap:         make(model.NodeMap),
		nodeGatewayMap:  make(map[string]bool),
		fireCh:          make(chan struct{}, 1),
		apiTimeout:      options.APITimeout,
		clusterPodCIDRs: options.ClusterPodCIDRs,
	}
	if err := shiba.initSelf(); err != nil {
		return nil, fmt.Errorf("failed to get info about self: %w", err)
	}
	if err := shiba.initCluster(); err != nil {
		return nil, fmt.Errorf("failed to get info about the cluster: %w", err)
	}
	if err := shiba.initCNI(); err != nil {
		return nil, fmt.Errorf("failed to init cni: %w", err)
	}
	if err := shiba.initNAT(); err != nil {
		return nil, fmt.Errorf("failed to init nat: %w", err)
	}
	shiba.loadNodeMap()
	shiba.fireCh <- struct{}{} // Trigger a sync for the loaded configuration.
	log.Info("shiba initialized")
	return shiba, nil
}

// Run starts the main routine until stopCh is closed.
func (shiba *Shiba) Run(stopCh <-chan struct{}) error {
	go shiba.execute(stopCh)
	go shiba.periodicFire(stopCh)
	for {
		watcher, err := shiba.client.CoreV1().Nodes().Watch(context.Background(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to watch node list: %w", err)
		}
		watcherCh := watcher.ResultChan()
		log.Info("shiba started listening")
		for {
			select {
			case <-stopCh:
				watcher.Stop()
				return nil
			case event, ok := <-watcherCh:
				if !ok {
					log.Info("watch channel closed")
					break
				}
				shiba.processEvent(event)
			}
		}
	}
}

// periodicFire triggers a sync every fireInterval, in case of external corruption.
func (shiba *Shiba) periodicFire(stopCh <-chan struct{}) {
	ticker := time.NewTicker(fireInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			select {
			case shiba.fireCh <- struct{}{}:
			default:
			}
		}
	}
}
