// Run commands using the replicated CLI
package main

import (
	"dagger/replicated/internal/dagger"
	"time"
)

// Replicated is a Dagger module that provides access to the replicated CLI
type Replicated struct {
	Token          *dagger.Secret
	APIOrigin      string
	IDOrigin       string
	RegistryOrigin string
}

func New(
	token *dagger.Secret,
	// +optional
	apiOrigin string,
	// +optional
	idOrigin string,
	// +optional
	registryOrigin string,
) *Replicated {
	return &Replicated{
		Token:          token,
		APIOrigin:      apiOrigin,
		IDOrigin:       idOrigin,
		RegistryOrigin: registryOrigin,
	}
}

// Obtain the base container with the replicated CLI installed at `/replicated`
func (m *Replicated) Container() *dagger.Container {
	ctr := dag.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	}).From("replicated/vendor-cli:latest").
		WithSecretVariable("REPLICATED_API_TOKEN", m.Token)

	if m.APIOrigin != "" {
		ctr = ctr.WithEnvVariable("REPLICATED_API_ORIGIN", m.APIOrigin)
	}
	if m.IDOrigin != "" {
		ctr = ctr.WithEnvVariable("REPLICATED_ID_ORIGIN", m.IDOrigin)
	}
	if m.RegistryOrigin != "" {
		ctr = ctr.WithEnvVariable("REPLICATED_REGISTRY_ORIGIN", m.RegistryOrigin)
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
