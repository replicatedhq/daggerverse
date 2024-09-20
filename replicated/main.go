// Run commands using the replicated CLI
package main

import (
	"context"
	"dagger/replicated/internal/dagger"
	"time"
)

// Replicated is a Dagger module that provides access to the replicated CLI
type Replicated struct{}

// Obtain the base container with the replicated CLI installed at `/replicated`
func (m *Replicated) ReplicatedContainer(
	ctx context.Context,
	token *dagger.Secret,
	// +optional
	apiOrigin string,
	// +optional
	idOrigin string,
	// +optional
	registryOrigin string,
) *dagger.Container {
	ctr := dag.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	}).From("replicated/vendor-cli:latest").
		WithSecretVariable("REPLICATED_API_TOKEN", token)

	if apiOrigin != "" {
		ctr = ctr.WithEnvVariable("REPLICATED_API_ORIGIN", apiOrigin)
	}
	if idOrigin != "" {
		ctr = ctr.WithEnvVariable("REPLICATED_ID_ORIGIN", idOrigin)
	}
	if registryOrigin != "" {
		ctr = ctr.WithEnvVariable("REPLICATED_REGISTRY_ORIGIN", registryOrigin)
	}
	return ctr
}

func cacheBustingExec(args []string, opts ...dagger.ContainerWithExecOpts) dagger.WithContainerFunc {
	return func(c *dagger.Container) *dagger.Container {
		if c == nil {
			panic("CacheBustingExec requires a container, but container was nil")
		}
		return c.WithEnvVariable("DAGGER_CACHEBUSTER_CBE", time.Now().String()).WithExec(args, opts...)
	}
}
