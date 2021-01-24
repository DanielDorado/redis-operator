package k8sutils

import (
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// Node represents each pod in a StatefulSet
type Node struct {
	Name string
	IP   string
}

// Nodes ordered by StatefulSet Node Order
type ByNodeIndex []Node

const (
	podIndexSeparator = "-"
)

func NewOrderedNodes(pods []corev1.Pod) ByNodeIndex {
	nodes := ByNodeIndex{}
	for _, pod := range pods {
		nodes = append(nodes, Node{Name: pod.Name, IP: pod.Status.HostIP})
	}
	sort.Sort(nodes)
	return nodes
}

func (n ByNodeIndex) IPs() []string {
	ips := []string{}
	for _, p := range n {
		ips = append(ips, p.IP)
	}
	return ips
}

func (n ByNodeIndex) PodNames() []string {
	names := []string{}
	for _, p := range n {
		names = append(names, p.Name)
	}
	return names
}

func (n ByNodeIndex) Len() int           { return len(n) }
func (n ByNodeIndex) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n ByNodeIndex) Less(i, j int) bool { return nodeIndex(n[i]) < nodeIndex(n[j]) }

func nodeIndex(node Node) int {
	parts := strings.Split(node.Name, podIndexSeparator)
	i, _ := strconv.Atoi(parts[len(parts)-1])
	return i
}
