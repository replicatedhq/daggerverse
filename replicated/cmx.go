package main

import (
	"context"
	"dagger/replicated/internal/dagger"
	"encoding/json"
	"fmt"
	"strconv"
)

// clusterResponse is a middleman struct for the JSON output of the CMX cluster create command
// because `id“ is a reserved name in Dagger - and exporting this struct will cause dagger build to fail.
type clusterResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Cluster is a struct representing a CMX cluster
type Cluster struct {
	ClusterID  string
	Status     string
	Kubeconfig string
}

// Create a new CMX cluster
//
// Example:
//
// dagger call cluster-create --name=my-cluster --wait=10m --ttl=20m --distribution=k3s --version=1.31.0 --token=env:REPLICATED_API_TOKEN
func (m *Replicated) ClusterCreate(
	ctx context.Context,
	token *dagger.Secret,
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
	// +default="1.31.0"
	version string,
	// Number of nodes to create
	// +default="1"
	nodes int,
	// +optional
	apiOrigin string,
	// +optional
	idOrigin string,
	// +optional
	registryOrigin string,
) (*Cluster, error) {
	replicated := m.ReplicatedContainer(ctx, token, apiOrigin, idOrigin, registryOrigin)

	cmd := []string{
		"/replicated",
		"cluster",
		"create",
		"--distribution", distribution,
		"--version", version,
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

	if nodes != 0 {
		cmd = append(cmd, "--nodes", fmt.Sprintf("%d", nodes))
	}

	cluster := replicated.With(cacheBustingExec(cmd))

	stdout, err := cluster.Stdout(ctx)
	if err != nil {
		return nil, err
	}

	cr := clusterResponse{}
	if err := json.Unmarshal([]byte(stdout), &cr); err != nil {
		return nil, err
	}

	kubeconfig, err := replicated.With(
		cacheBustingExec(
			[]string{
				"/replicated",
				"cluster",
				"kubeconfig",
				"--stdout",
				cr.ID,
			},
		),
	).Stdout(ctx)
	if err != nil {
		return nil, err
	}

	return &Cluster{
		ClusterID:  cr.ID,
		Status:     cr.Status,
		Kubeconfig: kubeconfig,
	}, nil
}

// Remove a CMX cluster
//
// Example:
//
// dagger call cluster-remove --cluster-id=my-cluster --token=env:REPLICATED_API_TOKEN
func (m *Replicated) ClusterRemove(
	ctx context.Context,
	token *dagger.Secret,
	// Cluster ID of the cluster to remove
	clusterID string,
	// +optional
	apiOrigin string,
	// +optional
	idOrigin string,
	// +optional
	registryOrigin string,
) (string, error) {

	replicated := m.ReplicatedContainer(ctx, token, apiOrigin, idOrigin, registryOrigin)
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
// dagger call cluster-expose-port --cluster-id=my-cluster --node-port=80 --token=env:REPLICATED_API_TOKEN
func (m *Replicated) ClusterExposePort(
	ctx context.Context,
	token *dagger.Secret,
	// Cluster ID of the cluster to remove
	clusterID string,
	// +optional
	apiOrigin string,
	// +optional
	idOrigin string,
	// +optional
	registryOrigin string,
	// Port to expose
	nodePort int,
) (string, error) {
	replicated := m.ReplicatedContainer(ctx, token, apiOrigin, idOrigin, registryOrigin)
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
