//go:build !darwin

package container

// detectVZ never finds the in-process Virtualization provider off-darwin —
// the framework (and the whole vz.go surface) is darwin-only, so non-darwin
// builds carry only this stub for the runtime.go detection chain.
func detectVZ() (ContainerRuntime, bool) { return ContainerRuntime{}, false }
