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

// --- AX-7 canonical triplets for the canonical service shape (Mantis #1336) ---

func TestService_NewService_Good(t *testing.T) {
	// A fully-wired NewService(opts) carries the supplied Provider and a
	// constructed State, retrievable from the Core under "container".
	c := core.New(core.WithService(NewService(ServiceOptions{
		Provider:  NewAppleProvider(),
		StatePath: core.PathJoin(t.TempDir(), "state.json"),
	})))
	svc := core.MustServiceFor[*Service](c, "container")
	if svc.Provider == nil {
		t.Fatal("expected the supplied Provider to be wired")
	}
	if svc.State == nil {
		t.Fatal("expected a non-nil State")
	}
}

func TestService_NewService_Bad(t *testing.T) {
	// Registering NewService twice under the same package-derived name must
	// fail: the second WithService call hits "already registered".
	c := core.New(core.WithService(NewService(ServiceOptions{})))
	if !c.Service("container").OK {
		t.Fatal("setup: first NewService registration should have succeeded")
	}
	r := core.WithService(NewService(ServiceOptions{}))(c)
	if r.OK {
		t.Fatal("expected duplicate NewService registration to fail")
	}
}

func TestService_NewService_Ugly(t *testing.T) {
	// Edge: the NewService factory always builds a fresh, independent Service
	// even with empty options, so two Cores never share Provider/State state.
	factory := NewService(ServiceOptions{})
	r1 := factory(core.New())
	r2 := factory(core.New())
	if !r1.OK || !r2.OK {
		t.Fatalf("NewService factory must construct: r1.OK=%v r2.OK=%v", r1.OK, r2.OK)
	}
	s1 := r1.Value.(*Service)
	s2 := r2.Value.(*Service)
	if s1 == s2 {
		t.Fatal("expected independent Service instances across Cores")
	}
	if s1.State == nil || s2.State == nil || s1.State == s2.State {
		t.Fatal("expected each Service to own a distinct non-nil State")
	}
}

func TestService_Register_Good(t *testing.T) {
	// Register(c) wires the container service with empty options: it is
	// discoverable, holds a nil Provider, and always has a non-nil State.
	c := core.New(core.WithService(Register))
	r := c.Service("container")
	if !r.OK {
		t.Fatal("container service not registered via Register")
	}
	svc := r.Value.(*Service)
	if svc.Provider != nil {
		t.Fatal("expected nil Provider from Register's empty options")
	}
	if svc.State == nil {
		t.Fatal("expected non-nil State from Register")
	}
}

func TestService_Register_Bad(t *testing.T) {
	// A locked service registry rejects registration: Register through
	// WithService on a locked Core must return a failed Result.
	c := core.New(core.WithServiceLock())
	r := core.WithService(Register)(c)
	if r.OK {
		t.Fatal("expected Register to fail against a locked service registry")
	}
}

func TestService_Register_Ugly(t *testing.T) {
	// Edge: registering twice collides on the "container" name — the second
	// Register attempt fails even though the registry is unlocked.
	c := core.New(core.WithService(Register))
	if !c.Service("container").OK {
		t.Fatal("setup: first Register should have succeeded")
	}
	r := core.WithService(Register)(c)
	if r.OK {
		t.Fatal("expected duplicate Register to fail")
	}
}
