// Package vm provides container/VM management commands across the LinuxKit
// (qemu/hyperkit) and Apple (`container`) runtimes.
//
// Container lifecycle:
//   - run: Run from an image (.iso/.qcow2/.vmdk/.raw or OCI ref) or template;
//     supports --publish/--volume/--env
//   - ps: List running containers (aggregated across runtimes)
//   - stop: Stop a running container
//   - kill: Kill a running container (SIGKILL)
//   - rm: Remove a container
//   - logs: View container logs
//   - exec: Execute a command inside a container
//   - inspect: Show detailed container information (JSON)
//   - templates: Manage LinuxKit templates (list, show, vars)
//
// Apple image management (requires the macOS `container` runtime):
//   - build: Build an OCI image from a Containerfile
//   - pull: Pull an image from a registry
//   - push: Push a locally-tagged image to a registry
//   - images: List images
//   - rmi: Remove an image
//
// Apple system management (requires the macOS `container` runtime):
//   - system start: Start the apiserver + default kernel (--no-kernel-install to skip)
//   - system status: Show system status
//   - system stop: Stop the system services
//
// kill/rm/inspect dispatch to whichever runtime owns the container id; the
// image and system commands are Apple-only (LinuxKit has no OCI image management).
package vm
