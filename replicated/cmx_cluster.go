package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// nodeGroupJSON is an unexported struct for unmarshaling JSON with the reserved "id" field
type nodeGroupJSON struct {
	ID             string   `json:"id"`
	IsDefault      bool     `json:"is_default"`
	InstanceType   string   `json:"instance_type"`
	Name           string   `json:"name"`
	NodeCount      int      `json:"node_count"`
	DiskGiB        int      `json:"disk_gib"`
	CreatedAt      string   `json:"created_at"`
	RunningAt      string   `json:"running_at"`
	CreditsPerHour int      `json:"credits_per_hour"`
	MinutesBilled  int      `json:"minutes_billed"`
	Nodes          []string `json:"nodes"`
}

// NodeGroup represents a group of nodes in a cluster
type NodeGroup struct {
	ItemID         string
	IsDefault      bool
	InstanceType   string
	Name           string
	NodeCount      int
	DiskGiB        int
	CreatedAt      string
	RunningAt      string
	CreditsPerHour int
	MinutesBilled  int
	Nodes          []string
}

// nodeGroupFromJSON creates a NodeGroup from a nodeGroupJSON struct
func nodeGroupFromJSON(ng nodeGroupJSON) NodeGroup {
	return NodeGroup{
		ItemID:         ng.ID,
		IsDefault:      ng.IsDefault,
		InstanceType:   ng.InstanceType,
		Name:           ng.Name,
		NodeCount:      ng.NodeCount,
		DiskGiB:        ng.DiskGiB,
		CreatedAt:      ng.CreatedAt,
		RunningAt:      ng.RunningAt,
		CreditsPerHour: ng.CreditsPerHour,
		MinutesBilled:  ng.MinutesBilled,
		Nodes:          ng.Nodes,
	}
}

// clusterJSON is an unexported struct for unmarshaling JSON with the reserved "id" field
type clusterJSON struct {
	ID                       string          `json:"id"`
	Name                     string          `json:"name"`
	Status                   string          `json:"status"`
	KubernetesDistribution   string          `json:"kubernetes_distribution"`
	KubernetesVersion        string          `json:"kubernetes_version"`
	NodeGroups               []nodeGroupJSON `json:"node_groups"`
	NetworkID                string          `json:"network_id"`
	CreatedAt                string          `json:"created_at"`
	ExpiresAt                string          `json:"expires_at"`
	TTL                      string          `json:"ttl"`
	CreditsPerHourPerCluster int             `json:"credits_per_hour_per_cluster"`
	FlatFee                  int             `json:"flat_fee"`
	TotalCredits             int             `json:"total_credits"`
	EstimatedCost            int             `json:"estimated_cost"`
	Tags                     []string        `json:"tags"`
}

// Cluster represents a single cluster in the response
type Cluster struct {
	ItemID                   string
	Name                     string
	Status                   string
	KubernetesDistribution   string
	KubernetesVersion        string
	NodeGroups               []NodeGroup
	NetworkID                string
	CreatedAt                string
	ExpiresAt                string
	TTL                      string
	CreditsPerHourPerCluster int
	FlatFee                  int
	TotalCredits             int
	EstimatedCost            int
	Tags                     []string
	Kubeconfig               string // Not part of the API response, populated after creation
}

// clusterFromJSON creates a Cluster from a clusterJSON struct
func clusterFromJSON(cj clusterJSON) Cluster {
	nodeGroups := make([]NodeGroup, len(cj.NodeGroups))
	for i, ngJSON := range cj.NodeGroups {
		nodeGroups[i] = nodeGroupFromJSON(ngJSON)
	}

	return Cluster{
		ItemID:                   cj.ID,
		Name:                     cj.Name,
		Status:                   cj.Status,
		KubernetesDistribution:   cj.KubernetesDistribution,
		KubernetesVersion:        cj.KubernetesVersion,
		NodeGroups:               nodeGroups,
		NetworkID:                cj.NetworkID,
		CreatedAt:                cj.CreatedAt,
		ExpiresAt:                cj.ExpiresAt,
		TTL:                      cj.TTL,
		CreditsPerHourPerCluster: cj.CreditsPerHourPerCluster,
		FlatFee:                  cj.FlatFee,
		TotalCredits:             cj.TotalCredits,
		EstimatedCost:            cj.EstimatedCost,
		Tags:                     cj.Tags,
	}
}

type ClusterVersion struct {
	Name          string   `json:"short_name"`
	Versions      []string `json:"versions"`
	InstanceTypes []string `json:"instance_types"`
	NodesMax      int      `json:"nodes_max"`
}

// Create a new CMX cluster
//
// Example:
//
// dagger call -m github.com/replicatedhq/daggerverse/replicated --token=env:REPLICATED_API_TOKEN cluster-create --name=my-cluster --wait=10m --ttl=20m --distribution=k3s --version=1.31.0
func (m *Replicated) ClusterCreate(
	ctx context.Context,
	// Name of the cluster
	// +optional
	name string,
	// How long to wait for the cluster to be ready
	// +default="15m"
	wait string,
	// TTL of the cluster
	// +default="20m"
	ttl string,
	// Distribution to use
	// +default="k3s"
	distribution string,
	// Version of the distribution to use
	// +optional
	version string,
	// Number of nodes to create
	// +default="1"
	nodes int,
) (*Cluster, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"cluster",
		"create",
		"--distribution", distribution,
		"--output", "json",
	}

	if name != "" {
		cmd = append(cmd, "--name", name)
	}

	if wait != "" {
		cmd = append(cmd, "--wait", wait)
	}

	if ttl != "" {
		cmd = append(cmd, "--ttl", ttl)
	}

	if version != "" {
		cmd = append(cmd, "--version", version)
	}

	if nodes != 0 {
		cmd = append(cmd, "--nodes", fmt.Sprintf("%d", nodes))
	}

	containerWithCmd := replicated.With(cacheBustingExec(cmd))

	stdout, err := containerWithCmd.Stdout(ctx)
	if err != nil {
		return nil, err
	}

	var clusterJsonData clusterJSON
	if err := json.Unmarshal([]byte(stdout), &clusterJsonData); err != nil {
		return nil, err
	}

	cluster := clusterFromJSON(clusterJsonData)

	kubeconfig, err := replicated.With(
		cacheBustingExec(
			[]string{
				"/replicated",
				"cluster",
				"kubeconfig",
				"--stdout",
				clusterJsonData.ID,
			},
		),
	).Stdout(ctx)
	if err != nil {
		return nil, err
	}

	// Set the kubeconfig
	cluster.Kubeconfig = kubeconfig

	return &cluster, nil
}

// Remove a CMX cluster
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN cluster-remove --cluster-id=my-cluster
func (m *Replicated) ClusterRemove(
	ctx context.Context,
	// Cluster ID of the cluster to remove
	clusterID string,
) (string, error) {
	replicated := m.Container()
	return replicated.With(
		cacheBustingExec(
			[]string{
				"/replicated",
				"cluster",
				"rm",
				clusterID,
			},
		),
	).Stdout(ctx)
}

// Expose a port on a CMX cluster, returning the hostname of the exposed port
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN cluster-expose-port --cluster-id=my-cluster --node-port=80
func (m *Replicated) ClusterExposePort(
	ctx context.Context,
	// Cluster ID of the cluster to remove
	clusterID string,
	// Port to expose
	nodePort int,
) (string, error) {
	replicated := m.Container()
	portExposeOutput, err := replicated.With(
		cacheBustingExec(
			[]string{
				"/replicated",
				"cluster",
				"port",
				"expose",
				clusterID,
				"--port", strconv.Itoa(nodePort),
				"--protocol", "https",
				"--output", "json",
			},
		),
	).Stdout(ctx)
	if err != nil {
		return "", err
	}

	type PortExpose struct {
		HostName string `json:"hostname"`
		State    string `json:"state"`
	}

	postExposeOutput := PortExpose{}
	if err := json.Unmarshal([]byte(portExposeOutput), &postExposeOutput); err != nil {
		return "", err
	}

	return postExposeOutput.HostName, nil
}

// Get the available cluster versions
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN cluster-versions
func (m *Replicated) ClusterVersions(ctx context.Context) (*[]ClusterVersion, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"cluster",
		"versions",
		"--output", "json",
	}

	versions := replicated.With(cacheBustingExec(cmd))

	stdout, err := versions.Stdout(ctx)
	if err != nil {
		return nil, err
	}

	cv := []ClusterVersion{}
	if err := json.Unmarshal([]byte(stdout), &cv); err != nil {
		return nil, err
	}

	return &cv, nil
}

// List all CMX clusters
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN cluster-list --show-terminated=true --start-time=2023-01-01T00:00:00Z
func (m *Replicated) ClusterList(
	ctx context.Context,
	// When set, only show terminated clusters
	// +optional
	showTerminated bool,
	// Start time for the query (Format: 2006-01-02T15:04:05Z)
	// +optional
	startTime string,
	// End time for the query (Format: 2006-01-02T15:04:05Z)
	// +optional
	endTime string,
) ([]Cluster, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"cluster",
		"ls",
		"--output", "json",
	}

	if showTerminated {
		cmd = append(cmd, "--show-terminated")
	}

	if startTime != "" {
		cmd = append(cmd, "--start-time", startTime)
	}

	if endTime != "" {
		cmd = append(cmd, "--end-time", endTime)
	}

	clusters := replicated.With(cacheBustingExec(cmd))

	stdout, err := clusters.Stdout(ctx)
	if err != nil {
		return nil, err
	}

	var clusterJSONList []clusterJSON
	if err := json.Unmarshal([]byte(stdout), &clusterJSONList); err != nil {
		return nil, err
	}

	clusterList := make([]Cluster, len(clusterJSONList))
	for i, clusterData := range clusterJSONList {
		clusterList[i] = clusterFromJSON(clusterData)
	}

	return clusterList, nil
}
