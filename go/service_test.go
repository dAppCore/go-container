// SPDX-License-Identifier: EUPL-1.2

package container

import (
	"testing"

	core "dappco.re/go"
)

// TestNewService_NilProvider_RegistersWithStateOnly — empty options registers
// Service with nil Provider but always-non-nil State, per the canon shape.
func TestNewService_NilProvider_RegistersWithStateOnly(t *testing.T) {
	c := core.New(core.WithService(NewService(ServiceOptions{})))
	r := c.Service("container")
	if !r.OK {
		t.Fatal("container service not registered")
	}
	svc := r.Value.(*Service)
	if svc.Provider != nil {
		t.Fatal("expected nil Provider with empty options")
	}
	if svc.State == nil {
		t.Fatal("expected non-nil State (always constructed)")
	}
}

// TestRegister_DefaultsRegistersContainer — imperative Register(c) shorthand.
func TestRegister_DefaultsRegistersContainer(t *testing.T) {
	c := core.New(core.WithService(Register))
	if !c.Service("container").OK {
		t.Fatal("container service not registered via Register")
	}
}
