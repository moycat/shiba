package main

import (
	"flag"
	"fmt"
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

func parseConfig() *Config {
	var config Config
	loader := configor.New(&configor.Config{
		ENVPrefix: envPrefix,
		Debug:     debugMode,
		Verbose:   debugMode,
	})
	if err := loader.Load(&config); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	set := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	set.StringVar(&config.NodeName, "node-name", config.NodeName, "current node name")
	set.StringVar(&config.CNIConfigPath, "cni-config-path", config.CNIConfigPath, "CNI config path")
	set.StringVar(&config.KubeConfigPath, "kube-config-path", config.KubeConfigPath, "K8s config file path")
	set.IntVar(&config.APITimeout, "api-timeout", config.APITimeout, "K8s API timeout in seconds")
	set.StringVar(&config.ClusterPodCIDRs, "cluster-pod-cidrs", config.ClusterPodCIDRs, "cluster pod CIDRs")
	set.IntVar(&config.PprofPort, "pprof-port", config.PprofPort, "pprof debug server port")
	set.IntVar(&config.IP6tnlMTU, "ip6tnl-mtu", config.IP6tnlMTU, "the MTU for ip6tnl interface")
	_ = set.Parse(os.Args[1:])
	if len(config.NodeName) == 0 {
		fmt.Println("node name is empty!")
		fmt.Println()
		set.Usage()
		os.Exit(2)
		return nil
	}
	if len(config.CNIConfigPath) == 0 {
		config.CNIConfigPath = defaultCNIConfigPath
	}
	if config.APITimeout <= 0 {
		config.APITimeout = defaultAPITimeout
	}
	return &config
}
