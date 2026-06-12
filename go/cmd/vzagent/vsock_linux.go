//go:build linux

package main

// Linux vsock listener — the only platform-specific part of the agent. The
// guest kernel must carry virtio-vsock support (CONFIG_VIRTIO_VSOCKETS=y,
// standard in LinuxKit kernels) for AF_VSOCK sockets to bind.

import (
	"errors"
	"os"
	"os/exec"
	"time"

	"golang.org/x/sys/unix"

	"dappco.re/go/container/internal/vzproto"
)

// listenBacklog is the pending-connection queue depth for the control port.
const listenBacklog = 4

// run binds the vsock control port and serves control connections until the
// listener fails. Each connection is served sequentially per the §5 single
// host-controller model; an acked stop powers the guest off.
func run(port uint32) error {
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)
	if err != nil {
		return errors.New("vsock socket: " + err.Error())
	}
	defer unix.Close(fd)

	addr := &unix.SockaddrVM{CID: unix.VMADDR_CID_ANY, Port: port}
	if err := unix.Bind(fd, addr); err != nil {
		return errors.New("vsock bind port " + itoa(port) + ": " + err.Error())
	}
	if err := unix.Listen(fd, listenBacklog); err != nil {
		return errors.New("vsock listen: " + err.Error())
	}

	for {
		connFd, _, err := unix.Accept(fd)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return errors.New("vsock accept: " + err.Error())
		}
		serveConnFd(connFd)
	}
}

// serveConnFd serves one accepted control connection. A stop acknowledged on
// this connection powers the guest off after the response drains.
func serveConnFd(connFd int) {
	conn := os.NewFile(uintptr(connFd), "vsock-control")
	if conn == nil {
		unix.Close(connFd)
		return
	}
	handler := &agentHandler{}
	_ = vzproto.Serve(conn, handler)
	_ = conn.Close()
	if handler.stopRequested {
		time.Sleep(stopFlushDelay)
		powerOff()
	}
}

// powerOff syncs filesystems and powers the guest off. The agent runs with
// CAP_SYS_BOOT (see testdata/vz/linuxkit-vzagent.yml); a refused reboot
// falls back to a best-effort poweroff binary so the verb still acts.
func powerOff() {
	unix.Sync()
	if err := unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF); err != nil {
		// CAP_SYS_BOOT missing — try the userspace path before giving up.
		fallbackPowerOff()
	}
}

// fallbackPowerOff shells the conventional poweroff binaries in PATH order.
// Reached only when the direct reboot syscall is refused.
func fallbackPowerOff() {
	for _, bin := range []string{"/sbin/poweroff", "poweroff", "/sbin/halt"} {
		if err := exec.Command(bin).Run(); err == nil {
			return
		}
	}
}

// itoa renders a port number without pulling fmt into the static binary.
func itoa(v uint32) string {
	if v == 0 {
		return "0"
	}
	var digits [10]byte
	i := len(digits)
	for v > 0 {
		i--
		digits[i] = byte('0' + v%10)
		v /= 10
	}
	return string(digits[i:])
}
