package vm

import (
	"context"
	goio "io"
	"text/tabwriter"
	"time"

	"dappco.re/go/core"
	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/container"
	"dappco.re/go/core/container/internal/proc"
	"dappco.re/go/core/i18n"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// addVMRunCommand registers the vm/run command.
//
// Flags: --name --detach --memory --cpus --ssh-port --template --var
// Positional arg: image path.
//
//	c vm run ./app.qcow2
//	c vm run --template core-dev --var SSH_KEY=abc
func addVMRunCommand(c *core.Core) {
	c.Command("vm/run", core.Command{
		Description: "cmd.vm.run.long",
		Action: func(opts core.Options) core.Result {
			runOpts := container.RunOptions{
				Name:    opts.String("name"),
				Detach:  opts.Bool("detach"),
				Memory:  opts.Int("memory"),
				CPUs:    opts.Int("cpus"),
				SSHPort: opts.Int("ssh-port"),
			}

			if templateName := opts.String("template"); templateName != "" {
				vars := parseVarOption(opts.String("var"))
				return resultFromError(RunFromTemplate(templateName, vars, runOpts))
			}

			image := opts.String("_arg")
			if image == "" {
				return resultFromError(coreerr.E("vm run", i18n.T("cmd.vm.run.error.image_required"), nil))
			}

			return resultFromError(runContainer(image, runOpts))
		},
	})
}

func runContainer(image string, opts container.RunOptions) error {
	manager, err := container.NewLinuxKitManager(io.Local)
	if err != nil {
		return coreerr.E("runContainer", i18n.T("i18n.fail.init", "container manager"), err)
	}

	cli.Print("%s %s\n", dimStyle.Render(i18n.Label("image")), image)
	if opts.Name != "" {
		cli.Print("%s %s\n", dimStyle.Render(i18n.T("cmd.vm.label.name")), opts.Name)
	}
	cli.Print("%s %s\n", dimStyle.Render(i18n.T("cmd.vm.label.hypervisor")), manager.Hypervisor().Name())

	ctx := context.Background()
	c, err := manager.Run(ctx, image, opts)
	if err != nil {
		return coreerr.E("runContainer", i18n.T("i18n.fail.run", "container"), err)
	}

	if opts.Detach {
		cli.Print("%s %s\n", successStyle.Render(i18n.Label("started")), c.ID)
		cli.Print("%s %d\n", dimStyle.Render(i18n.T("cmd.vm.label.pid")), c.PID)
		cli.Println("%s", i18n.T("cmd.vm.hint.view_logs", map[string]any{"ID": c.ID[:8]}))
		cli.Println("%s", i18n.T("cmd.vm.hint.stop", map[string]any{"ID": c.ID[:8]}))
	} else {
		cli.Print("%s %s\n", dimStyle.Render(i18n.T("cmd.vm.label.container_stopped")), c.ID)
	}

	return nil
}

// addVMPsCommand registers the vm/ps command.
//
// Flags: --all (include stopped containers).
//
//	c vm ps --all
func addVMPsCommand(c *core.Core) {
	c.Command("vm/ps", core.Command{
		Description: "cmd.vm.ps.long",
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
			cli.Println("%s", i18n.T("cmd.vm.ps.no_containers"))
		} else {
			cli.Println("%s", i18n.T("cmd.vm.ps.no_running"))
		}
		return nil
	}

	w := tabwriter.NewWriter(proc.Stdout, 0, 0, 2, ' ', 0)
	core.Print(w, "%s\n", i18n.T("cmd.vm.ps.header"))
	core.Print(w, "%s\n", "--\t----\t-----\t------\t-------\t---")

	for _, c := range containers {
		imageName := c.Image
		if len(imageName) > 30 {
			imageName = "..." + imageName[len(imageName)-27:]
		}

		duration := formatDuration(time.Since(c.StartedAt))

		status := string(c.Status)
		switch c.Status {
		case container.StatusRunning:
			status = successStyle.Render(status)
		case container.StatusStopped:
			status = dimStyle.Render(status)
		case container.StatusError:
			status = errorStyle.Render(status)
		}

		core.Print(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
			c.ID[:8], c.Name, imageName, status, duration, c.PID)
	}

	_ = w.Flush()
	return nil
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

// addVMStopCommand registers the vm/stop command.
//
// Positional arg: container id (partial prefix supported).
//
//	c vm stop a1b2c3
func addVMStopCommand(c *core.Core) {
	c.Command("vm/stop", core.Command{
		Description: "cmd.vm.stop.long",
		Action: func(opts core.Options) core.Result {
			id := opts.String("_arg")
			if id == "" {
				return resultFromError(coreerr.E("vm stop", i18n.T("cmd.vm.error.id_required"), nil))
			}
			return resultFromError(stopContainer(id))
		},
	})
}

func stopContainer(id string) error {
	manager, err := container.NewLinuxKitManager(io.Local)
	if err != nil {
		return coreerr.E("stopContainer", i18n.T("i18n.fail.init", "container manager"), err)
	}

	fullID, err := resolveContainerID(manager, id)
	if err != nil {
		return err
	}

	cli.Print("%s %s\n", dimStyle.Render(i18n.T("cmd.vm.stop.stopping")), fullID[:8])

	ctx := context.Background()
	if err := manager.Stop(ctx, fullID); err != nil {
		return coreerr.E("stopContainer", i18n.T("i18n.fail.stop", "container"), err)
	}

	cli.Print("%s\n", successStyle.Render(i18n.T("common.status.stopped")))
	return nil
}

// resolveContainerID resolves a partial ID prefix to a full ID.
// Errors on ambiguous or missing matches.
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

// addVMLogsCommand registers the vm/logs command.
//
// Flags: --follow (tail new entries).
// Positional arg: container id.
//
//	c vm logs a1b2c3 --follow
func addVMLogsCommand(c *core.Core) {
	c.Command("vm/logs", core.Command{
		Description: "cmd.vm.logs.long",
		Action: func(opts core.Options) core.Result {
			id := opts.String("_arg")
			if id == "" {
				return resultFromError(coreerr.E("vm logs", i18n.T("cmd.vm.error.id_required"), nil))
			}
			return resultFromError(viewLogs(id, opts.Bool("follow")))
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
	defer func() { _ = reader.Close() }()

	_, err = goio.Copy(proc.Stdout, reader)
	return err
}

// addVMExecCommand registers the vm/exec command.
//
// Positional arg: container id. Flag: --cmd='shell cmd' — Core CLI preserves a
// single positional so multi-word commands are passed via --cmd.
//
//	c vm exec a1b2c3 --cmd='ls -la /app'
func addVMExecCommand(c *core.Core) {
	c.Command("vm/exec", core.Command{
		Description: "cmd.vm.exec.long",
		Action: func(opts core.Options) core.Result {
			id := opts.String("_arg")
			command := opts.String("cmd")
			if id == "" || command == "" {
				return resultFromError(coreerr.E("vm exec", i18n.T("cmd.vm.error.id_and_cmd_required"), nil))
			}
			return resultFromError(execInContainer(id, splitCommand(command)))
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

// resultFromError adapts a Go error into a Core Result.
//
//	return resultFromError(someOperation())
func resultFromError(err error) core.Result {
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{OK: true}
}

// parseVarOption parses a single --var flag value into a map. Returns an
// empty map for empty input. For multi-variable template instantiation,
// callers invoke ParseVarFlags with an explicit slice.
//
//	parseVarOption("SSH_KEY=abc")  → {"SSH_KEY":"abc"}
func parseVarOption(value string) map[string]string {
	if core.Trim(value) == "" {
		return map[string]string{}
	}
	return ParseVarFlags([]string{value})
}

// splitCommand splits a shell-like command string on whitespace, collapsing
// consecutive spaces. Quoted arguments are not unpacked — use `--cmd` with a
// single simple invocation; compound shell expressions should be wrapped in
// `sh -c`.
//
//	splitCommand("ls -la /app")  → ["ls","-la","/app"]
func splitCommand(command string) []string {
	parts := core.Split(command, " ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := core.Trim(p)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
