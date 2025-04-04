package main

import (
	"errors"
	"flag"
	"os"

	"github.com/jinzhu/configor"
	log "github.com/sirupsen/logrus"
)

const (
	envPrefix = "SHIBA"
	debugEnv  = "SHIBA_DEBUG"
)

const (
	defaultCNIConfigPath = "/etc/cni/net.d"
	defaultAPITimeout    = 30
)

var debugMode bool

func init() {
	if len(os.Getenv(debugEnv)) > 0 {
		debugMode = true
		log.SetLevel(log.DebugLevel)
	}
}

type Config struct {
	// NodeName must be set to the name of the current node.
	NodeName string
	// CNIConfigPath is the path to CNI configuration files, usually /etc/cni/net.d.
	CNIConfigPath string
	// KubeConfigPath is the path to the kubeconfig file, using in-cluster config if empty.
	KubeConfigPath string
	// APITimeout is the timeout in seconds for non-watch API calls.
	APITimeout int
	// ClusterPodCIDRs is the pod CIDR subnets of the cluster.
	ClusterPodCIDRs string
	// PprofPort specifies the port of pprof debug server, non-positive to disable.
	PprofPort int

	// IP6tnlMTU is the MTU for ip6tnl interface. when not config, default is 1450.
	IP6tnlMTU int
}

func (c *Config) InitFlags(set *flag.FlagSet) {
	set.StringVar(&c.NodeName, "node-name", c.NodeName, "current node name")
	set.StringVar(&c.CNIConfigPath, "cni-config-path", c.CNIConfigPath, "CNI config path")
	set.StringVar(&c.KubeConfigPath, "kube-config-path", c.KubeConfigPath, "K8s config file path")
	set.IntVar(&c.APITimeout, "api-timeout", c.APITimeout, "K8s API timeout in seconds")
	set.StringVar(&c.ClusterPodCIDRs, "cluster-pod-cidrs", c.ClusterPodCIDRs, "cluster pod CIDRs")
	set.IntVar(&c.PprofPort, "pprof-port", c.PprofPort, "pprof debug server port")
	set.IntVar(&c.IP6tnlMTU, "ip6tnl-mtu", c.IP6tnlMTU, "the MTU for ip6tnl interface")
}

func (c *Config) Validate() error {
	if len(c.NodeName) == 0 {
		return errors.New("node name is empty")
	}
	if len(c.CNIConfigPath) == 0 {
		c.CNIConfigPath = defaultCNIConfigPath
	}
	if c.APITimeout <= 0 {
		c.APITimeout = defaultAPITimeout
	}
	return nil
}

func newConfig() *Config {
	var config Config
	loader := configor.New(&configor.Config{
		ENVPrefix: envPrefix,
		Debug:     debugMode,
		Verbose:   debugMode,
	})
	if err := loader.Load(&config); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return &config
}
