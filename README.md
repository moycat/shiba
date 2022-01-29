# Shiba（柴）

Shiba is a minimalist Kubernetes network plugin, as a replacement for [flannel](https://github.com/flannel-io/flannel), [Calico](https://www.tigera.io/project-calico/), etc.

It provides the basic networking capabilities for Kubernetes, including:

- Pod address assignment (with the help of `host-local` CNI plugin)
- Overlay network (via Linux built-in IP tunnels) with **dual-stack** support (!)

It doesn't have advanced features like:

- Network policy
- Floating IPs
- BGP routing (who likes it?)

In a stable cluster, the functionality completely relies on the bundled CNI plugins and native Linux modules; Shiba daemon only works when a node has joined or left the cluster, or the node itself has rebooted.

Thus, the architecture is very simple, lightweight and friendly to debugging.

## Prerequisites

At the current stage, Shiba has the following requirements and limitations:

- The cluster must be configured with IPv6 support.
- Each node must have a routable IPv6 address as its `InternalIP` for tunneling.
- Some settings (like NAT) are hardcoded.
- Only Kubernetes 1.22.0+ & Linux kernels 4.19+ are tested and supported.

## Installation

1. Make sure it's a new cluster, or the previous network plugin has been completely purged.
2. Assign an IPv6 `InternalIP` to each node if not already, by adding a `--node-ip` parameter to `kubelet`.
3. If the cluster is NOT set up with `kubeadm`, fill in the `SHIBA_CLUSTERPODCIDRS` env in `installation.yaml`.
4. Run `kubectl apply -f installation.yaml` and enjoy.
