//go:build !linux

package main

// Non-Linux hosts can build vzagent (so `go build ./...` stays green on the
// darwin development host) but never run it — the agent's vsock listener is
// guest-side Linux only.

import "errors"

// run refuses to serve outside a Linux guest.
func run(_ uint32) error {
	return errors.New("vzagent runs inside a Linux guest only (build with GOOS=linux GOARCH=arm64)")
}
