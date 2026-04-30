package vm

import (
	// Note: AX-6 — context.Context is structural because container manager APIs use it for cancellation; no core primitive.
	"context"
	// Note: AX-6 — io.Copy is structural for streaming process log output to stdout without buffering; core.io Copy is medium/path based.
	goio "io"
	// Note: AX-6 — text/tabwriter is structural for CLI table formatting; no core primitive.
	"text/tabwriter"
	// Note: AX-6 — time is structural for elapsed container duration formatting; no core primitive.
	"time"

	core "dappco.re/go"
	"dappco.re/go/cli/pkg/i18n"
	"dappco.re/go/container"
	"dappco.re/go/container/internal/proc"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

var (
	runName         string
	runDetach       bool
	runMemory       int
	runCPUs         int
	runSSHPort      int
	runTemplateName string
	runVarFlags     []string
	runRuntime      string
	runGPU          bool
)

// addVMRunCommand adds the 'run' command under vm.
func addVMRunCommand(c *core.Core) {
	registerVMCommand(c, "vm/run", core.Command{
		Description: i18n.T("cmd.vm.run.short"),
		Flags: core.NewOptions(
			core.Option{Key: "name", Value: ""},
			core.Option{Key: "detach", Value: false},
			core.Option{Key: "memory", Value: 0},
			core.Option{Key: "cpus", Value: 0},
			core.Option{Key: "ssh-port", Value: 0},
			core.Option{Key: "template", Value: ""},
			core.Option{Key: "runtime", Value: ""},
			core.Option{Key: "gpu", Value: false},
		),
		Action: func(opts core.Options) core.Result {
			runOpts := container.RunOptions{
				Name:    opts.String("name"),
				Detach:  opts.Bool("detach"),
				Memory:  opts.Int("memory"),
				CPUs:    opts.Int("cpus"),
				SSHPort: opts.Int("ssh-port"),
				GPU:     opts.Bool("gpu"),
			}

			// If template is specified, build and run from template
			if templateName := opts.String("template"); templateName != "" {
				vars := ParseVarFlags(optionStrings(opts, "var"))
				return resultFromError(RunFromTemplate(templateName, vars, runOpts))
			}

			// Otherwise, require an image path
			args := optionArgs(opts)
			if len(args) == 0 {
				return core.Fail(coreerr.E("vm run", i18n.T("cmd.vm.run.error.image_required"), nil))
			}
			image := args[0]

			return resultFromError(runContainer(
				image,
				opts.String("name"),
				opts.Bool("detach"),
				opts.Int("memory"),
				opts.Int("cpus"),
				opts.Int("ssh-port"),
				opts.String("runtime"),
				opts.Bool("gpu"),
			))
		},
	})
}

// resolveRuntime maps the --runtime flag onto a RuntimeType. Empty string
// triggers auto-detection via container.Detect().
//
// Usage:
//
//	rt, err := resolveRuntime("apple")
func resolveRuntime(flag string) (container.RuntimeType, error) {
	switch core.Lower(flag) {
	case "", "auto":
		rt := container.Detect()
		if rt.Type == container.RuntimeNone {
			return container.RuntimeNone, coreerr.E("resolveRuntime", i18n.T("cmd.vm.run.error.no_runtime"), nil)
		}
		return rt.Type, nil
	case "apple":
		return container.RuntimeApple, nil
	case "docker":
		return container.RuntimeDocker, nil
	case "podman":
		return container.RuntimePodman, nil
	case "linuxkit":
		return container.RuntimeLinuxKit, nil
	case "tim":
		// TIM is an image format; route it to the LinuxKit provider.
		return container.RuntimeLinuxKit, nil
	default:
		return container.RuntimeNone, coreerr.E("resolveRuntime", "unknown runtime: "+flag, nil)
	}
}

func runContainer(image, name string, detach bool, memory, cpus, sshPort int, runtimeFlag string, gpu bool) error {
	rtType, err := resolveRuntime(runtimeFlag)
	if err != nil {
		return err
	}

	opts := container.RunOptions{
		Name:    name,
		Detach:  detach,
		Memory:  memory,
		CPUs:    cpus,
		SSHPort: sshPort,
		GPU:     gpu,
	}

	core.Print(nil, "%s %s", dimStyle.Render(i18n.Label("image")), image)
	if name != "" {
		core.Print(nil, "%s %s", dimStyle.Render(i18n.T("cmd.vm.label.name")), name)
	}
	core.Print(nil, "%s %s", dimStyle.Render(i18n.T("cmd.vm.label.runtime")), string(rtType))

	if rtType == container.RuntimeApple {
		return runContainerApple(image, name, detach, memory, cpus, gpu)
	}

	// LinuxKit (default) path — also used for Docker/Podman which route through
	// the LinuxKit manager's hypervisor wrapper until native providers land.
	manager, err := container.NewLinuxKitManager(io.Local)
	if err != nil {
		return coreerr.E("runContainer", i18n.T("i18n.fail.init", "container manager"), err)
	}
	core.Print(nil, "%s %s", dimStyle.Render(i18n.T("cmd.vm.label.hypervisor")), manager.Hypervisor().Name())
	core.Println()

	ctx := context.Background()
	c, err := manager.Run(ctx, image, opts)
	if err != nil {
		return coreerr.E("runContainer", i18n.T("i18n.fail.run", "container"), err)
	}

	if detach {
		core.Print(nil, "%s %s", successStyle.Render(i18n.Label("started")), c.ID)
		core.Print(nil, "%s %d", dimStyle.Render(i18n.T("cmd.vm.label.pid")), c.PID)
		core.Println()
		core.Println(i18n.T("cmd.vm.hint.view_logs", map[string]any{"ID": c.ID[:8]}))
		core.Println(i18n.T("cmd.vm.hint.stop", map[string]any{"ID": c.ID[:8]}))
	} else {
		core.Println()
		core.Print(nil, "%s %s", dimStyle.Render(i18n.T("cmd.vm.label.container_stopped")), c.ID)
	}

	return nil
}

// runContainerApple boots an image through the AppleProvider. Ports and
// volumes are omitted — they are handled via the Apple CLI directly when
// declared on the image's source config.
func runContainerApple(image, name string, detach bool, memory, cpus int, gpu bool) error {
	p := container.NewAppleProvider()
	if !p.Available() {
		return coreerr.E("runContainerApple", i18n.T("cmd.vm.run.error.apple_unavailable"), nil)
	}
	core.Println()

	img := &container.Image{
		Name:     name,
		Path:     image,
		Format:   container.DetectImageFormat(image),
		Provider: string(container.RuntimeApple),
	}

	opts := []container.RunOption{
		container.WithName(name),
		container.WithMemory(memory),
		container.WithCPUs(cpus),
		container.WithDetach(detach),
	}
	if gpu {
		opts = append(opts, container.WithGPU(true))
	}

	c, err := p.Run(img, opts...)
	if err != nil {
		return coreerr.E("runContainerApple", i18n.T("i18n.fail.run", "container"), err)
	}

	core.Print(nil, "%s %s", successStyle.Render(i18n.Label("started")), c.ID)
	core.Print(nil, "%s %d", dimStyle.Render(i18n.T("cmd.vm.label.pid")), c.PID)
	core.Println()
	return nil
}

var psAll bool

// addVMPsCommand adds the 'ps' command under vm.
func addVMPsCommand(c *core.Core) {
	registerVMCommand(c, "vm/ps", core.Command{
		Description: i18n.T("cmd.vm.ps.short"),
		Flags:       core.NewOptions(core.Option{Key: "all", Value: false}),
		Action: func(opts core.Options) core.Result {
			return resultFromError(listContainers(opts.Bool("all")))
		},
	})
}

func listContainers(all bool) error {
	manager, err := container.NewLinuxKitManager(io.Local)
	if err != nil {
		return coreerr.E("listContainers", i18n.T("i18n.fail.init", "container manager"), err)
	}

	ctx := context.Background()
	containers, err := manager.List(ctx)
	if err != nil {
		return coreerr.E("listContainers", i18n.T("i18n.fail.list", "containers"), err)
	}

	// Filter if not showing all
	if !all {
		filtered := make([]*container.Container, 0)
		for _, c := range containers {
			if c.Status == container.StatusRunning {
				filtered = append(filtered, c)
			}
		}
		containers = filtered
	}

	if len(containers) == 0 {
		if all {
			core.Println(i18n.T("cmd.vm.ps.no_containers"))
		} else {
			core.Println(i18n.T("cmd.vm.ps.no_running"))
		}
		return nil
	}

	w := tabwriter.NewWriter(proc.Stdout, 0, 0, 2, ' ', 0)
	core.Print(w, "%s", i18n.T("cmd.vm.ps.header"))
	core.Print(w, "%s", "--\t----\t-----\t------\t-------\t---")

	for _, c := range containers {
		// Shorten image path
		imageName := c.Image
		if len(imageName) > 30 {
			imageName = "..." + imageName[len(imageName)-27:]
		}

		// Format duration
		duration := formatDuration(time.Since(c.StartedAt))

		// Status with color
		status := string(c.Status)
		switch c.Status {
		case container.StatusRunning:
			status = successStyle.Render(status)
		case container.StatusStopped:
			status = dimStyle.Render(status)
		case container.StatusError:
			status = errorStyle.Render(status)
		}

		core.Print(w, "%s\t%s\t%s\t%s\t%s\t%d",
			c.ID[:8], c.Name, imageName, status, duration, c.PID)
	}

	return w.Flush()
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return core.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return core.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return core.Sprintf("%dh", int(d.Hours()))
	}
	return core.Sprintf("%dd", int(d.Hours()/24))
}

// addVMStopCommand adds the 'stop' command under vm.
func addVMStopCommand(c *core.Core) {
	registerVMCommand(c, "vm/stop", core.Command{
		Description: i18n.T("cmd.vm.stop.short"),
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) == 0 {
				return core.Fail(coreerr.E("vm stop", i18n.T("cmd.vm.error.id_required"), nil))
			}
			return resultFromError(stopContainer(args[0]))
		},
	})
}

func stopContainer(id string) error {
	manager, err := container.NewLinuxKitManager(io.Local)
	if err != nil {
		return coreerr.E("stopContainer", i18n.T("i18n.fail.init", "container manager"), err)
	}

	// Support partial ID matching
	fullID, err := resolveContainerID(manager, id)
	if err != nil {
		return err
	}

	core.Print(nil, "%s %s", dimStyle.Render(i18n.T("cmd.vm.stop.stopping")), fullID[:8])

	ctx := context.Background()
	if err := manager.Stop(ctx, fullID); err != nil {
		return coreerr.E("stopContainer", i18n.T("i18n.fail.stop", "container"), err)
	}

	core.Print(nil, "%s", successStyle.Render(i18n.T("common.status.stopped")))
	return nil
}

// resolveContainerID resolves a partial ID to a full ID.
func resolveContainerID(manager *container.LinuxKitManager, partialID string) (string, error) {
	ctx := context.Background()
	containers, err := manager.List(ctx)
	if err != nil {
		return "", err
	}

	var matches []*container.Container
	for _, c := range containers {
		if core.HasPrefix(c.ID, partialID) || core.HasPrefix(c.Name, partialID) {
			matches = append(matches, c)
		}
	}

	switch len(matches) {
	case 0:
		return "", coreerr.E("resolveContainerID", i18n.T("cmd.vm.error.no_match", map[string]any{"ID": partialID}), nil)
	case 1:
		return matches[0].ID, nil
	default:
		return "", coreerr.E("resolveContainerID", i18n.T("cmd.vm.error.multiple_match", map[string]any{"ID": partialID}), nil)
	}
}

var logsFollow bool

// addVMLogsCommand adds the 'logs' command under vm.
func addVMLogsCommand(c *core.Core) {
	registerVMCommand(c, "vm/logs", core.Command{
		Description: i18n.T("cmd.vm.logs.short"),
		Flags:       core.NewOptions(core.Option{Key: "follow", Value: false}),
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) == 0 {
				return core.Fail(coreerr.E("vm logs", i18n.T("cmd.vm.error.id_required"), nil))
			}
			return resultFromError(viewLogs(args[0], opts.Bool("follow")))
		},
	})
}

func viewLogs(id string, follow bool) error {
	manager, err := container.NewLinuxKitManager(io.Local)
	if err != nil {
		return coreerr.E("viewLogs", i18n.T("i18n.fail.init", "container manager"), err)
	}

	fullID, err := resolveContainerID(manager, id)
	if err != nil {
		return err
	}

	ctx := context.Background()
	reader, err := manager.Logs(ctx, fullID, follow)
	if err != nil {
		return coreerr.E("viewLogs", i18n.T("i18n.fail.get", "logs"), err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			// Streaming has already ended; return the copy error if one exists.
		}
	}()

	_, err = goio.Copy(proc.Stdout, reader)
	return err
}

// addVMExecCommand adds the 'exec' command under vm.
func addVMExecCommand(c *core.Core) {
	registerVMCommand(c, "vm/exec", core.Command{
		Description: i18n.T("cmd.vm.exec.short"),
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) < 2 {
				return core.Fail(coreerr.E("vm exec", i18n.T("cmd.vm.error.id_and_cmd_required"), nil))
			}
			return resultFromError(execInContainer(args[0], args[1:]))
		},
	})
}

func execInContainer(id string, cmd []string) error {
	manager, err := container.NewLinuxKitManager(io.Local)
	if err != nil {
		return coreerr.E("execInContainer", i18n.T("i18n.fail.init", "container manager"), err)
	}

	fullID, err := resolveContainerID(manager, id)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return manager.Exec(ctx, fullID, cmd)
}
