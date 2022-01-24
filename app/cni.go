package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moycat/shiba/util"
	log "github.com/sirupsen/logrus"
)

// initCNI writes the CNI configuration file for the container runtime.
func (shiba *Shiba) initCNI() error {
	config := shiba.generateCNIConfig()
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cni config: %w", err)
	}
	path := filepath.Join(shiba.cniConfigPath, cniConfigName)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("failed to write cni config [%s]: %w", path, err)
	}
	entries, err := os.ReadDir(shiba.cniConfigPath)
	if err != nil {
		return fmt.Errorf("failed to open cni config path [%s] for checking: %w", shiba.cniConfigPath, err)
	}
	log.Infof("cni config is written to [%s]", path)
	// Check whether there are additional configs and give warnings.
	for _, entry := range entries {
		if entryName := entry.Name(); entryName != cniConfigName &&
			(strings.HasSuffix(entryName, ".conf") || strings.HasSuffix(entryName, ".conflist")) {
			log.Warningf("another cni config [%s] found in [%s], there could be a problem",
				entryName, shiba.cniConfigPath)
		}
	}
	return nil
}

func (shiba *Shiba) generateCNIConfig() map[string]interface{} {
	var (
		hasV4, hasV6 bool
		podCIDRs     [][]map[string]interface{}
		routes       []map[string]interface{}
	)
	for _, cidr := range shiba.nodePodCIDRs {
		switch {
		case util.IsV4(cidr.IP):
			hasV4 = true
		case util.IsV6(cidr.IP):
			hasV6 = true
		default:
			continue
		}
		podCIDRs = append(podCIDRs, []map[string]interface{}{{"subnet": cidr.String()}})
	}
	if hasV4 {
		routes = append(routes, map[string]interface{}{"dst": "0.0.0.0/0"})
	}
	if hasV6 {
		routes = append(routes, map[string]interface{}{"dst": "::/0"})
	}
	return map[string]interface{}{
		"name":       cniNetName,
		"cniVersion": "0.3.1",
		"plugins": []map[string]interface{}{
			{
				"type": "ptp", // Create a veth pair for each pod.
				"ipam": map[string]interface{}{
					"type":   "host-local", // Allocate IPs locally within following ranges.
					"ranges": podCIDRs,
					"routes": routes,
				},
			},
			{
				"type":         "portmap", // Essential for HostPort.
				"snat":         true,
				"capabilities": map[string]bool{"portMappings": true},
			},
		},
	}
}
