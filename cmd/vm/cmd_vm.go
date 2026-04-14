// Package vm provides LinuxKit VM management commands.
//
// Commands register on the shared Core instance via path-based routing:
//
//	vm             → group (no action)
//	vm/run         → start a VM from image or template
//	vm/ps          → list running containers
//	vm/stop        → stop a container by id
//	vm/logs        → view container logs
//	vm/exec        → execute a command inside a container over SSH
//	vm/templates   → list available templates
package vm

import (
	"dappco.re/go/core"
	"dappco.re/go/core/cli/pkg/cli"
)

func init() {
	cli.RegisterCommands(AddVMCommands)
}

// Style aliases from shared package.
var (
	repoNameStyle = cli.RepoStyle
	successStyle  = cli.SuccessStyle
	errorStyle    = cli.ErrorStyle
	dimStyle      = cli.DimStyle
)

// VM-specific styles.
var (
	varStyle     = cli.NewStyle().Foreground(cli.ColourAmber500)
	defaultStyle = cli.NewStyle().Foreground(cli.ColourGray500).Italic()
)

// AddVMCommands registers the vm command tree on the Core instance.
//
//	vm.AddVMCommands(c)
func AddVMCommands(c *core.Core) {
	// Group placeholder (no Action). Sub-commands below set their own Actions.
	c.Command("vm", core.Command{Description: "cmd.vm.long"})

	addVMRunCommand(c)
	addVMPsCommand(c)
	addVMStopCommand(c)
	addVMLogsCommand(c)
	addVMExecCommand(c)
	addVMTemplatesCommand(c)
}
