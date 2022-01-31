package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/moycat/shiba/app"
	"github.com/moycat/shiba/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	if os.Geteuid() != 0 {
		log.Fatal("shiba must be run as root")
	}
	config := parseConfig()
	client := getKubernetesClient(config.KubeConfigPath)
	options := getShibaOptions(config)
	shiba, err := app.NewShiba(client, config.NodeName, config.CNIConfigPath, options)
	if err != nil {
		log.Fatalf("failed to create shiba: %v", err)
	}
	stopCh := make(chan struct{})
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go waitForSignals(signalCh, stopCh)
	go servePprof(config.PprofPort)
	if err := shiba.Run(stopCh); err != nil {
		log.Fatal(err)
	}
}

func getShibaOptions(config *Config) app.ShibaOptions {
	options := app.ShibaOptions{
		APITimeout: time.Duration(config.APITimeout) * time.Second,
	}
	if len(config.ClusterPodCIDRs) > 0 {
		cidrs, err := util.ParseIPNets(strings.Split(config.ClusterPodCIDRs, ","))
		if err != nil {
			log.Fatalf("failed to parse cluster pod cidrs: %v", err)
		}
		options.ClusterPodCIDRs = cidrs
	}
	return options
}

func getKubernetesClient(kubeConfigPath string) kubernetes.Interface {
	var (
		restConfig *rest.Config
		err        error
	)
	if len(kubeConfigPath) > 0 {
		// Use the given kube config.
		log.Debugf("loading kube config from [%s]", kubeConfigPath)
		b, err := os.ReadFile(kubeConfigPath)
		if err != nil {
			log.Fatalf("failed to read kube config [%s]: %v", kubeConfigPath, err)
		}
		config, err := clientcmd.NewClientConfigFromBytes(b)
		if err != nil {
			log.Fatalf("failed to parse kube config [%s]: %v", kubeConfigPath, err)
		}
		restConfig, err = config.ClientConfig()
		if err != nil {
			log.Fatalf("failed to get rest config from kube config [%s]: %v", kubeConfigPath, err)
		}
	} else {
		// Use the in-cluster config.
		log.Debug("loading kube config from in-cluster files")
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("failed to get in-cluster config: %v", err)
		}
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("failed to create rest client: %v", err)
	}
	return client
}

func waitForSignals(signalCh <-chan os.Signal, stopCh chan<- struct{}) {
	<-signalCh
	log.Info("signal captured, exiting")
	_ = os.Stdout.Sync()
	_ = os.Stderr.Sync()
	close(stopCh)
}

func servePprof(port int) {
	if port <= 0 {
		return
	}
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Errorf("pprof exited: %v", err)
	}
}
