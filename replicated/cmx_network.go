package main

import (
	"context"
	"encoding/json"
	"fmt"
)

// networkJSON is an unexported middleman structfor unmarshaling JSON with the dagger reserved "id" field
type networkJSON struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	ExpiresAt       string `json:"expires_at"`
	TTL             string `json:"ttl"`
	OverlayEndpoint string `json:"overlay_endpoint"`
	OverlayToken    string `json:"overlay_token"`
	Policy          string `json:"policy"`
}

// Network represents a single network in the response
type Network struct {
	ItemID          string
	Name            string
	Status          string
	CreatedAt       string
	ExpiresAt       string
	TTL             string
	OverlayEndpoint string
	OverlayToken    string
	Policy          string
}

// fromJSON creates a Network from a networkJSON struct
func networkFromJSON(nj networkJSON) Network {
	return Network{
		ItemID:          nj.ID,
		Name:            nj.Name,
		Status:          nj.Status,
		CreatedAt:       nj.CreatedAt,
		ExpiresAt:       nj.ExpiresAt,
		TTL:             nj.TTL,
		OverlayEndpoint: nj.OverlayEndpoint,
		OverlayToken:    nj.OverlayToken,
		Policy:          nj.Policy,
	}
}

// Create a new CMX network
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN network-create --name=my-network --wait=10m --ttl=20m
func (m *Replicated) NetworkCreate(
	ctx context.Context,
	// Name of the network
	// +optional
	name string,
	// How long to wait for the network to be ready
	// +default="15m"
	wait string,
	// TTL of the network
	// +default="20m"
	ttl string,
) ([]Network, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"network",
		"create",
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

	containerWithCmd := replicated.With(cacheBustingExec(cmd))

	stdout, err := containerWithCmd.Stdout(ctx)
	if err != nil {
		return nil, err
	}

	var networkJSONList []networkJSON
	if err := json.Unmarshal([]byte(stdout), &networkJSONList); err != nil {
		return nil, err
	}

	// Convert all networks from the JSON response
	networkList := make([]Network, len(networkJSONList))
	for i, networkJSON := range networkJSONList {
		networkList[i] = networkFromJSON(networkJSON)
	}

	return networkList, nil
}

// Remove a CMX network
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN network-remove --network-id=my-network-id
// dagger call --token=env:REPLICATED_API_TOKEN network-remove --network-name=my-network-name
func (m *Replicated) NetworkRemove(
	ctx context.Context,
	// Network ID of the network to remove
	// +optional
	networkID string,
	// Network name of the network to remove
	// +optional
	networkName string,
) (string, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"network",
		"rm",
	}

	if networkID != "" {
		cmd = append(cmd, networkID)
	} else if networkName != "" {
		cmd = append(cmd, "--name", networkName)
	} else {
		return "", fmt.Errorf("either network-id or network-name must be specified")
	}

	return replicated.With(cacheBustingExec(cmd)).Stdout(ctx)
}

// Update a CMX network policy
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN network-update-policy --network-id=my-network-id --policy=airgap
// dagger call --token=env:REPLICATED_API_TOKEN network-update-policy --network-name=my-network-name --policy=airgap
func (m *Replicated) NetworkUpdatePolicy(
	ctx context.Context,
	// Network ID to update
	// +optional
	networkID string,
	// Network name to update
	// +optional
	networkName string,
	// New policy for the network (e.g., "airgap")
	policy string,
) (string, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"network",
		"update",
		"policy",
	}

	if networkID != "" {
		cmd = append(cmd, networkID)
	} else if networkName != "" {
		cmd = append(cmd, "--name", networkName)
	} else {
		return "", fmt.Errorf("either network-id or network-name must be specified")
	}

	cmd = append(cmd, "--policy", policy, "--output", "json")

	return replicated.With(cacheBustingExec(cmd)).Stdout(ctx)
}

// List all CMX networks
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN network-list
func (m *Replicated) NetworkList(
	ctx context.Context,
) ([]Network, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"network",
		"ls",
		"--output", "json",
	}

	networks := replicated.With(cacheBustingExec(cmd))

	stdout, err := networks.Stdout(ctx)
	if err != nil {
		return nil, err
	}

	var networkJSONList []networkJSON
	if err := json.Unmarshal([]byte(stdout), &networkJSONList); err != nil {
		return nil, err
	}

	networkList := make([]Network, len(networkJSONList))
	for i, networkJSON := range networkJSONList {
		networkList[i] = networkFromJSON(networkJSON)
	}

	return networkList, nil
}
