// SPDX-License-Identifier: EUPL-1.2

package container

import (
	core "dappco.re/go"
)

// ExampleNewService wires the container service into a Core with a Provider
// and an on-disk State path, then retrieves the typed handle.
func ExampleNewService() {
	c := core.New(core.WithService(NewService(ServiceOptions{
		Provider:  NewAppleProvider(),
		StatePath: "/var/lib/core/containers/state.json",
	})))
	svc := core.MustServiceFor[*Service](c, "container")
	_ = svc.Provider
	_ = svc.State
}

// ExampleRegister wires the container service with empty options — the
// imperative shorthand — leaving a nil Provider for late injection.
func ExampleRegister() {
	c := core.New(core.WithService(Register))
	svc := core.MustServiceFor[*Service](c, "container")
	svc.Provider = NewAppleProvider()
}
