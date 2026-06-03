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
	"dappco.re/go/container"
	"dappco.re/go/container/internal/proc"
	"dappco.re/go/io"
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
		Description: vmT("cmd.vm.run.short"),
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
			// --publish/--volume/--env are repeatable (read like --var); no
			// short forms (core.Option carries no short alias).
			publishRes := parsePublish(optionStrings(opts, "publish"))
			if !publishRes.OK {
				return publishRes
			}
			volumeRes := parseVolumes(optionStrings(opts, "volume"))
			if !volumeRes.OK {
				return volumeRes
			}
			envRes := parseEnv(optionStrings(opts, "env"))
			if !envRes.OK {
				return envRes
			}

			runOpts := container.RunOptions{
				Name:    opts.String("name"),
				Detach:  opts.Bool("detach"),
				Memory:  opts.Int("memory"),
				CPUs:    opts.Int("cpus"),
				SSHPort: opts.Int("ssh-port"),
				GPU:     opts.Bool("gpu"),
				Ports:   core.MustCast[map[int]int](publishRes),
				Volumes: core.MustCast[map[string]string](volumeRes),
				Env:     core.MustCast[[]string](envRes),
			}

			// If template is specified, build and run from template.
			if templateName := opts.String("template"); templateName != "" {
				vars := ParseVarFlags(optionStrings(opts, "var"))
				return resultFromError(RunFromTemplate(templateName, vars, runOpts))
			}

			// Otherwise, require an image path; trailing args are the container command.
			args := optionArgs(opts)
			if len(args) == 0 {
				return core.Fail(core.E("vm run", vmT("cmd.vm.run.error.image_required"), nil))
			}
			runOpts.Args = args[1:]

			return resultFromError(runContainer(args[0], opts.String("runtime"), runOpts))
		},
	})
}

// resolveRuntime maps the --runtime flag onto a RuntimeType. Empty string
// triggers auto-detection via container.Detect().
//
// Usage:
//
//	rt, err := resolveRuntime("apple")
func resolveRuntime(flag string) (
	container.RuntimeType,
	error,
) {
	switch core.Lower(flag) {
	case "", "auto":
		rt := container.Detect()
		if rt.Type == container.RuntimeNone {
			return container.RuntimeNone, core.E("resolveRuntime", vmT("cmd.vm.run.error.no_runtime"), nil)
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
		return container.RuntimeNone, core.E("resolveRuntime", "unknown runtime: "+flag, nil)
	}
}

func runContainer(image, runtimeFlag string, opts container.RunOptions) (
	err error, // result
) {
	rtType, err := resolveRuntime(runtimeFlag)
	if err != nil {
		return err
	}

	core.Print(nil, "%s %s", dimStyle.Render(vmT("image")), image)
	if opts.Name != "" {
		core.Print(nil, "%s %s", dimStyle.Render(vmT("cmd.vm.label.name")), opts.Name)
	}
	core.Print(nil, "%s %s", dimStyle.Render(vmT("cmd.vm.label.runtime")), string(rtType))

	if rtType == container.RuntimeApple {
		return runContainerApple(image, opts)
	}

	// LinuxKit (default) path — also used for Docker/Podman which route through
	// the LinuxKit manager's hypervisor wrapper until native providers land.
	mgrRes := container.NewLinuxKitManager(io.Local)
	if !mgrRes.OK {
		return core.E("runContainer", vmT("i18n.fail.init", "container manager"), mgrRes.Value.(error))
	}
	manager := core.MustCast[*container.LinuxKitManager](mgrRes)
	core.Print(nil, "%s %s", dimStyle.Render(vmT("cmd.vm.label.hypervisor")), manager.Hypervisor().Name())
	core.Println()

	ctx := context.Background()
	runRes := manager.Run(ctx, image, opts)
	if !runRes.OK {
		return core.E("runContainer", vmT("i18n.fail.run", "container"), runRes.Value.(error))
	}
	c := core.MustCast[*container.Container](runRes)

	if opts.Detach {
		core.Print(nil, "%s %s", successStyle.Render(vmT("started")), c.ID)
		core.Print(nil, "%s %d", dimStyle.Render(vmT("cmd.vm.label.pid")), c.PID)
		core.Println()
		core.Println(vmT("cmd.vm.hint.view_logs", map[string]any{"ID": shortID(c.ID)}))
		core.Println(vmT("cmd.vm.hint.stop", map[string]any{"ID": shortID(c.ID)}))
	} else {
		core.Println()
		core.Print(nil, "%s %s", dimStyle.Render(vmT("cmd.vm.label.container_stopped")), c.ID)
	}

	return nil
}

// runContainerApple boots an image through the AppleProvider, forwarding the
// resolved RunOptions: name, resources, published ports, volume mounts, env,
// container args, and GPU request.
func runContainerApple(image string, opts container.RunOptions) (
	err error, // result
) {
	p := container.NewAppleProvider()
	if !p.Available() {
		return core.E("runContainerApple", vmT("cmd.vm.run.error.apple_unavailable"), nil)
	}
	core.Println()

	img := &container.Image{
		Name:     opts.Name,
		Path:     image,
		Format:   container.DetectImageFormat(image),
		Provider: string(container.RuntimeApple),
	}

	runOpts := []container.RunOption{
		container.WithName(opts.Name),
		container.WithMemory(opts.Memory),
		container.WithCPUs(opts.CPUs),
		container.WithDetach(opts.Detach),
		container.WithPorts(opts.Ports),
		container.WithVolumes(opts.Volumes),
		container.WithEnv(opts.Env...),
	}
	if len(opts.Args) > 0 {
		runOpts = append(runOpts, container.WithArgs(opts.Args...))
	}
	if opts.GPU {
		runOpts = append(runOpts, container.WithGPU(true))
	}

	runRes := p.Run(img, runOpts...)
	if !runRes.OK {
		return core.E("runContainerApple", vmT("i18n.fail.run", "container"), runRes.Value.(error))
	}
	c := core.MustCast[*container.Container](runRes)

	core.Print(nil, "%s %s", successStyle.Render(vmT("started")), c.ID)
	core.Print(nil, "%s %d", dimStyle.Render(vmT("cmd.vm.label.pid")), c.PID)
	core.Println()
	return nil
}

// shortID truncates a container id to its first 8 characters for display.
// Apple container ids are the user-chosen --name and may be shorter than 8,
// so a naive id[:8] would panic.
func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// parsePublish parses docker-style "[host-ip:]host:container[/proto]" port
// specs into a host→container map. The host-ip prefix and /proto suffix are
// dropped (tcp assumed; RunOptions.Ports is map[int]int).
//
// Usage:
//
//	ports := core.MustCast[map[int]int](parsePublish([]string{"8080:80"}))
func parsePublish(specs []string) core.Result { // Value: map[int]int
	ports := make(map[int]int, len(specs))
	for _, s := range specs {
		spec := s
		if core.Contains(spec, "/") {
			spec = core.Split(spec, "/")[0]
		}
		parts := core.Split(spec, ":")
		if len(parts) < 2 {
			return core.Fail(core.E("vm run", core.Sprintf("invalid --publish %q: want host:container", s), nil))
		}
		hr := core.Atoi(parts[len(parts)-2])
		cr := core.Atoi(parts[len(parts)-1])
		if !hr.OK || !cr.OK {
			return core.Fail(core.E("vm run", core.Sprintf("invalid --publish %q: non-numeric port", s), nil))
		}
		ports[core.MustCast[int](hr)] = core.MustCast[int](cr)
	}
	return core.Ok(ports)
}

// parseVolumes parses "host:container" mount specs into a host→container map.
//
// Usage:
//
//	vols := core.MustCast[map[string]string](parseVolumes([]string{"/data:/app"}))
func parseVolumes(specs []string) core.Result { // Value: map[string]string
	vols := make(map[string]string, len(specs))
	for _, s := range specs {
		parts := core.SplitN(s, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return core.Fail(core.E("vm run", core.Sprintf("invalid --volume %q: want host:container", s), nil))
		}
		vols[parts[0]] = parts[1]
	}
	return core.Ok(vols)
}

// parseEnv validates KEY=VALUE env specs (the value may be empty or contain '=').
//
// Usage:
//
//	env := core.MustCast[[]string](parseEnv([]string{"PORT=8080"}))
func parseEnv(specs []string) core.Result { // Value: []string
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		if !core.Contains(s, "=") {
			return core.Fail(core.E("vm run", core.Sprintf("invalid --env %q: want KEY=VALUE", s), nil))
		}
		out = append(out, s)
	}
	return core.Ok(out)
}

// appleProvider returns an available AppleProvider, or nil when the Apple
// container runtime is not present on this host.
func appleProvider() *container.AppleProvider {
	p := container.NewAppleProvider()
	if p.Available() {
		return p
	}
	return nil
}

// resolveContainerOwner resolves a partial id/name to its full id and the
// runtime that owns it. Apple containers are consulted first when the runtime
// is available, then LinuxKit-managed containers. A non-nil returned provider
// means the container is an Apple container; nil means LinuxKit.
func resolveContainerOwner(partialID string) (
	*container.AppleProvider, // apple owner (nil for LinuxKit)
	string, // full id
	error, // result
) {
	if ap := appleProvider(); ap != nil {
		if listRes := ap.List(); listRes.OK {
			for _, c := range core.MustCast[[]*container.Container](listRes) {
				if core.HasPrefix(c.ID, partialID) || core.HasPrefix(c.Name, partialID) {
					return ap, c.ID, nil
				}
			}
		}
	}
	mgrRes := container.NewLinuxKitManager(io.Local)
	if !mgrRes.OK {
		return nil, "", core.E("resolveContainerOwner", vmT("i18n.fail.init", "container manager"), mgrRes.Value.(error))
	}
	manager := core.MustCast[*container.LinuxKitManager](mgrRes)
	fullID, err := resolveContainerID(manager, partialID)
	return nil, fullID, err
}

var psAll bool

// addVMPsCommand adds the 'ps' command under vm.
func addVMPsCommand(c *core.Core) {
	registerVMCommand(c, "vm/ps", core.Command{
		Description: vmT("cmd.vm.ps.short"),
		Flags:       core.NewOptions(core.Option{Key: "all", Value: false}),
		Action: func(opts core.Options) core.Result {
			return resultFromError(listContainers(opts.Bool("all")))
		},
	})
}

func listContainers(all bool) (
	err error, // result
) {
	mgrRes := container.NewLinuxKitManager(io.Local)
	if !mgrRes.OK {
		return core.E("listContainers", vmT("i18n.fail.init", "container manager"), mgrRes.Value.(error))
	}
	manager := core.MustCast[*container.LinuxKitManager](mgrRes)

	ctx := context.Background()
	listRes := manager.List(ctx)
	if !listRes.OK {
		return core.E("listContainers", vmT("i18n.fail.list", "containers"), listRes.Value.(error))
	}
	containers := core.MustCast[[]*container.Container](listRes)

	// Include Apple containers when the runtime is available, so a container
	// started via --runtime=apple is visible here too. Apple's `container ls`
	// reports running containers; --all stopped-set remains LinuxKit-scoped.
	if ap := appleProvider(); ap != nil {
		if appleRes := ap.List(); appleRes.OK {
			containers = append(containers, core.MustCast[[]*container.Container](appleRes)...)
		}
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
			core.Println(vmT("cmd.vm.ps.no_containers"))
		} else {
			core.Println(vmT("cmd.vm.ps.no_running"))
		}
		return nil
	}

	w := tabwriter.NewWriter(proc.Stdout, 0, 0, 2, ' ', 0)
	core.Print(w, "%s", vmT("cmd.vm.ps.header"))
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
			shortID(c.ID), c.Name, imageName, status, duration, c.PID)
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
		Description: vmT("cmd.vm.stop.short"),
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) == 0 {
				return core.Fail(core.E("vm stop", vmT("cmd.vm.error.id_required"), nil))
			}
			return resultFromError(stopContainer(args[0]))
		},
	})
}

func stopContainer(id string) (
	err error, // result
) {
	apple, fullID, err := resolveContainerOwner(id)
	if err != nil {
		return err
	}

	core.Print(nil, "%s %s", dimStyle.Render(vmT("cmd.vm.stop.stopping")), shortID(fullID))

	if apple != nil {
		if r := apple.Stop(fullID); !r.OK {
			return core.E("stopContainer", vmT("i18n.fail.stop", "container"), r.Value.(error))
		}
		core.Print(nil, "%s", successStyle.Render(vmT("common.status.stopped")))
		return nil
	}

	mgrRes := container.NewLinuxKitManager(io.Local)
	if !mgrRes.OK {
		return core.E("stopContainer", vmT("i18n.fail.init", "container manager"), mgrRes.Value.(error))
	}
	manager := core.MustCast[*container.LinuxKitManager](mgrRes)
	ctx := context.Background()
	if r := manager.Stop(ctx, fullID); !r.OK {
		return core.E("stopContainer", vmT("i18n.fail.stop", "container"), r.Value.(error))
	}

	core.Print(nil, "%s", successStyle.Render(vmT("common.status.stopped")))
	return nil
}

// resolveContainerID resolves a partial ID to a full ID.
func resolveContainerID(manager *container.LinuxKitManager, partialID string) (
	string,
	error,
) {
	ctx := context.Background()
	listRes := manager.List(ctx)
	if !listRes.OK {
		return "", listRes.Value.(error)
	}
	containers := core.MustCast[[]*container.Container](listRes)

	var matches []*container.Container
	for _, c := range containers {
		if core.HasPrefix(c.ID, partialID) || core.HasPrefix(c.Name, partialID) {
			matches = append(matches, c)
		}
	}

	switch len(matches) {
	case 0:
		return "", core.E("resolveContainerID", vmT("cmd.vm.error.no_match", map[string]any{"ID": partialID}), nil)
	case 1:
		return matches[0].ID, nil
	default:
		return "", core.E("resolveContainerID", vmT("cmd.vm.error.multiple_match", map[string]any{"ID": partialID}), nil)
	}
}

var logsFollow bool

// addVMLogsCommand adds the 'logs' command under vm.
func addVMLogsCommand(c *core.Core) {
	registerVMCommand(c, "vm/logs", core.Command{
		Description: vmT("cmd.vm.logs.short"),
		Flags:       core.NewOptions(core.Option{Key: "follow", Value: false}),
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) == 0 {
				return core.Fail(core.E("vm logs", vmT("cmd.vm.error.id_required"), nil))
			}
			return resultFromError(viewLogs(args[0], opts.Bool("follow")))
		},
	})
}

func viewLogs(id string, follow bool) (
	err error, // result
) {
	apple, fullID, err := resolveContainerOwner(id)
	if err != nil {
		return err
	}

	if apple != nil {
		// Apple logs are a snapshot via the container CLI's -n; --follow
		// streaming is a LinuxKit capability.
		logsRes := apple.Logs(fullID, 0)
		if !logsRes.OK {
			return core.E("viewLogs", vmT("i18n.fail.get", "logs"), logsRes.Value.(error))
		}
		core.Print(nil, "%s", core.MustCast[string](logsRes))
		return nil
	}

	mgrRes := container.NewLinuxKitManager(io.Local)
	if !mgrRes.OK {
		return core.E("viewLogs", vmT("i18n.fail.init", "container manager"), mgrRes.Value.(error))
	}
	manager := core.MustCast[*container.LinuxKitManager](mgrRes)

	ctx := context.Background()
	logsRes := manager.Logs(ctx, fullID, follow)
	if !logsRes.OK {
		return core.E("viewLogs", vmT("i18n.fail.get", "logs"), logsRes.Value.(error))
	}
	reader := core.MustCast[container.ReadCloser](logsRes)
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
		Description: vmT("cmd.vm.exec.short"),
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) < 2 {
				return core.Fail(core.E("vm exec", vmT("cmd.vm.error.id_and_cmd_required"), nil))
			}
			return resultFromError(execInContainer(args[0], args[1:]))
		},
	})
}

func execInContainer(id string, cmd []string) (
	err error, // result
) {
	apple, fullID, err := resolveContainerOwner(id)
	if err != nil {
		return err
	}

	if apple != nil {
		execRes := apple.Exec(fullID, cmd[0], cmd[1:]...)
		if !execRes.OK {
			return execRes.Value.(error)
		}
		core.Print(nil, "%s", core.MustCast[string](execRes))
		return nil
	}

	mgrRes := container.NewLinuxKitManager(io.Local)
	if !mgrRes.OK {
		return core.E("execInContainer", vmT("i18n.fail.init", "container manager"), mgrRes.Value.(error))
	}
	manager := core.MustCast[*container.LinuxKitManager](mgrRes)

	ctx := context.Background()
	if r := manager.Exec(ctx, fullID, cmd); !r.OK {
		return r.Value.(error)
	}
	return nil
}
