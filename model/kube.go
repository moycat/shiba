package model

// KubeadmClusterConfiguration is to retrieve the cluster pod CIDRs configured by kubeadm.
type KubeadmClusterConfiguration struct {
	Networking struct {
		PodSubnet string `yaml:"podSubnet"`
	} `yaml:"networking"`
}
