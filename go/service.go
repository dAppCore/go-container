// SPDX-License-Identifier: EUPL-1.2

// Service registration for the container package — exposes the
// canonical `NewService(opts)` + `Register(c)` shape per Mantis #1336,
// holding a pre-wired Provider + State so consumers can manage
// containers through the Core service registry.
//
//	c, _ := core.New(
//	    core.WithService(container.NewService(container.ServiceOptions{
//	        Provider:  appleProvider,
//	        StatePath: "/var/lib/core/containers/state.json",
//	    })),
//	)
//	svc := core.MustServiceFor[*container.Service](c, "container")
//	r := svc.Provider.Run(ctx, image, container.WithDetach(true))
//
// The Provider interface (apple.go, linuxkit.go, etc.) and the State
// type remain the source of truth — Service is a thin Core-side handle
// that gives the package a registerable identity the rest of the
// framework can discover via core.ServiceFor.

package container

import (
	core "dappco.re/go"
)

// ServiceOptions configures the container service. Provider is required
// for runtime use — empty Provider registers a Service but Container
// operations through it return a configuration error.
//
// Named ServiceOptions (not Options) to avoid colliding with the
// existing RunOptions / ContainerConfig structs in this package.
type ServiceOptions struct {
	// Provider is the container backend (AppleProvider, LinuxKitProvider,
	// etc.). nil → callers must inject via svc.Provider before use.
	Provider Provider
	// StatePath is the on-disk path for the persistent State store.
	// Empty → uses State only in memory; State methods that persist
	// will no-op or error per the State implementation's defaults.
	StatePath string
}

// Service is the registerable handle for the container package — embeds
// *core.ServiceRuntime[ServiceOptions] for typed options access and
// exposes the wired Provider + State.
//
// Usage example: `svc := core.MustServiceFor[*container.Service](c, "container"); _ = svc.Provider.Run(ctx, img)`
type Service struct {
	*core.ServiceRuntime[ServiceOptions]
	// Provider is the live container backend. nil if the consumer
	// constructed Options without one — inject via svc.Provider before
	// use.
	Provider Provider
	// State is the persistent container state store. Always non-nil —
	// constructed with the supplied StatePath (or empty path if none).
	State *State
}

// NewService returns a factory that constructs a *Service holding the
// supplied Provider + State and registers it under "container" via
// core.WithService.
//
//	core.WithService(container.NewService(container.ServiceOptions{
//	    Provider:  appleProvider,
//	    StatePath: "/var/lib/core/containers/state.json",
//	}))
//
// Provider may be nil — Service exposes it as svc.Provider for late
// injection. State is always constructed (defaults to empty StatePath
// if none supplied).
func NewService(opts ServiceOptions) func(*core.Core) core.Result {
	return func(c *core.Core) core.Result {
		return core.Ok(&Service{
			ServiceRuntime: core.NewServiceRuntime(c, opts),
			Provider:       opts.Provider,
			State:          NewState(opts.StatePath),
		})
	}
}

// Register wires the container service into the Core with empty
// ServiceOptions — the imperative-style alternative to NewService.
// The resulting *Service holds a nil Provider and a memory-only State,
// so consumers must inject a Provider before container operations.
//
//	c := core.New()
//	if r := container.Register(c); !r.OK { return r }
//	svc := core.MustServiceFor[*container.Service](c, "container")
//	svc.Provider = appleProvider
func Register(c *core.Core) core.Result {
	return NewService(ServiceOptions{})(c)
}
