package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

type VMVersion struct {
	Name          string   `json:"short_name"`
	Versions      []string `json:"versions"`
	InstanceTypes []string `json:"instance_types"`
}

// vmJSON is an unexported middleman structfor unmarshaling JSON with the reserved "id" field
type vmJSON struct {
	ItemID            string   `json:"id"`
	Name              string   `json:"name"`
	Status            string   `json:"status"`
	Distribution      string   `json:"distribution"`
	Version           string   `json:"version"`
	CreatedAt         string   `json:"created_at"`
	NetworkID         string   `json:"network_id"`
	ExpiresAt         string   `json:"expires_at"`
	TTL               string   `json:"ttl"`
	CreditsPerHour    int      `json:"credits_per_hour"`
	FlatFee           int      `json:"flat_fee"`
	TotalCredits      int      `json:"total_credits"`
	EstimatedCost     int      `json:"estimated_cost"`
	DirectSSHPort     int      `json:"direct_ssh_port"`
	DirectSSHEndpoint string   `json:"direct_ssh_endpoint"`
	Tags              []string `json:"tags"`
}

// VM represents a single VM in the response
type VM struct {
	ItemID            string
	Name              string
	Status            string
	Distribution      string
	Version           string
	CreatedAt         string
	NetworkID         string
	ExpiresAt         string
	TTL               string
	CreditsPerHour    int
	FlatFee           int
	TotalCredits      int
	EstimatedCost     int
	DirectSSHPort     int
	DirectSSHEndpoint string
	Tags              []string
}

// vmFromJSON creates a VM from a vmJSON struct
func vmFromJSON(vj vmJSON) VM {
	//nolint:gosimple // We're using a middleman struct here to work around
	// the dagger "id" field reservation.
	return VM{
		ItemID:            vj.ItemID,
		Name:              vj.Name,
		Status:            vj.Status,
		Distribution:      vj.Distribution,
		Version:           vj.Version,
		CreatedAt:         vj.CreatedAt,
		NetworkID:         vj.NetworkID,
		ExpiresAt:         vj.ExpiresAt,
		TTL:               vj.TTL,
		CreditsPerHour:    vj.CreditsPerHour,
		FlatFee:           vj.FlatFee,
		TotalCredits:      vj.TotalCredits,
		EstimatedCost:     vj.EstimatedCost,
		DirectSSHPort:     vj.DirectSSHPort,
		DirectSSHEndpoint: vj.DirectSSHEndpoint,
		Tags:              vj.Tags,
	}
}

// Create a new CMX VM
//
// Example:
//
// dagger call -m github.com/replicatedhq/daggerverse/replicated --token=env:REPLICATED_API_TOKEN vm-create --name=my-vm --wait=10m --ttl=20m --distribution=ubuntu --version=22.04
func (m *Replicated) VmCreate(
	ctx context.Context,
	// Name of the VM
	// +optional
	name string,
	// How long to wait for the VM to be ready
	// +default="15m"
	wait string,
	// TTL of the VM
	// +default="20m"
	ttl string,
	// Distribution to use
	// +default="ubuntu"
	distribution string,
	// Version of the distribution to use
	// +optional
	version string,
	// Number of VMs to create, each share a network
	// +default="1"
	count int,
	// Disk size in GiB
	// +default="50"
	disk int,
	// Instance type to use
	// +optional
	instanceType string,
) (*VM, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"vm",
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

	if count != 0 {
		cmd = append(cmd, "--count", fmt.Sprintf("%d", count))
	}

	if disk != 0 {
		cmd = append(cmd, "--disk", fmt.Sprintf("%d", disk))
	}

	if instanceType != "" {
		cmd = append(cmd, "--instance-type", instanceType)
	}

	containerWithCmd := replicated.With(cacheBustingExec(cmd))

	stdout, err := containerWithCmd.Stdout(ctx)
	if err != nil {
		return nil, err
	}

	var vmJsonData vmJSON
	if err := json.Unmarshal([]byte(stdout), &vmJsonData); err != nil {
		return nil, err
	}

	vm := vmFromJSON(vmJsonData)
	return &vm, nil
}

// Remove a CMX VM
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN vm-remove --vm-id=my-vm-id
// dagger call --token=env:REPLICATED_API_TOKEN vm-remove --vm-name=my-vm-name
func (m *Replicated) VmRemove(
	ctx context.Context,
	// VM ID of the VM to remove
	// +optional
	vmID string,
	// VM name of the VM to remove
	// +optional
	vmName string,
) (string, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"vm",
		"rm",
	}

	if vmID != "" {
		cmd = append(cmd, vmID)
	} else if vmName != "" {
		cmd = append(cmd, "--name", vmName)
	} else {
		return "", fmt.Errorf("either vm-id or vm-name must be specified")
	}

	return replicated.With(cacheBustingExec(cmd)).Stdout(ctx)
}

// Expose a port on a CMX VM, returning the hostname of the exposed port
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN vm-expose-port --vm-id=my-vm-id --vm-port=80
func (m *Replicated) VmExposePort(
	ctx context.Context,
	// VM ID of the VM to expose port on
	vmID string,
	// Port to expose
	vmPort int,
) (string, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"vm",
		"port",
		"expose",
		vmID,
		"--port", strconv.Itoa(vmPort),
		"--protocol", "https",
		"--output", "json",
	}

	portExposeOutput, err := replicated.With(cacheBustingExec(cmd)).Stdout(ctx)
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

// Get the available VM versions
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN vm-versions
func (m *Replicated) VmVersions(ctx context.Context) (*[]VMVersion, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"vm",
		"versions",
		"--output", "json",
	}

	versions := replicated.With(cacheBustingExec(cmd))

	stdout, err := versions.Stdout(ctx)
	if err != nil {
		return nil, err
	}

	vv := []VMVersion{}
	if err := json.Unmarshal([]byte(stdout), &vv); err != nil {
		return nil, err
	}

	return &vv, nil
}

// List all CMX VMs
//
// Example:
//
// dagger call --token=env:REPLICATED_API_TOKEN vm-list --show-terminated=true --start-time=2023-01-01T00:00:00Z
func (m *Replicated) VmList(
	ctx context.Context,
	// When set, only show terminated VMs
	// +optional
	showTerminated bool,
	// Start time for the query (Format: 2006-01-02T15:04:05Z)
	// +optional
	startTime string,
	// End time for the query (Format: 2006-01-02T15:04:05Z)
	// +optional
	endTime string,
) ([]VM, error) {
	replicated := m.Container()

	cmd := []string{
		"/replicated",
		"vm",
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

	vms := replicated.With(cacheBustingExec(cmd))

	stdout, err := vms.Stdout(ctx)
	if err != nil {
		return nil, err
	}

	var vmJSONList []vmJSON
	if err := json.Unmarshal([]byte(stdout), &vmJSONList); err != nil {
		return nil, err
	}

	vmList := make([]VM, len(vmJSONList))
	for i, vmData := range vmJSONList {
		vmList[i] = vmFromJSON(vmData)
	}

	return vmList, nil
}
