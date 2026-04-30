// Package vm provides LinuxKit VM management commands.
package vm

import (
	core "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/cli/pkg/i18n"
)

func init() {
	cli.RegisterCommands(AddVMCommands)
}

// Style aliases from shared
var (
	repoNameStyle = cli.RepoStyle
	successStyle  = cli.SuccessStyle
	errorStyle    = cli.ErrorStyle
	dimStyle      = cli.DimStyle
)

// VM-specific styles
var (
	varStyle     = cli.NewStyle().Foreground(cli.ColourAmber500)
	defaultStyle = cli.NewStyle().Foreground(cli.ColourGray500).Italic()
)

// AddVMCommands adds container-related commands under 'vm' to the CLI.
//
// Usage:
//
//	AddVMCommands(c)
func AddVMCommands(c *core.Core) {
	registerVMCommand(c, "vm", core.Command{Description: i18n.T("cmd.vm.short")})
	addVMRunCommand(c)
	addVMPsCommand(c)
	addVMStopCommand(c)
	addVMLogsCommand(c)
	addVMExecCommand(c)
	addVMTemplatesCommand(c)
}

func registerVMCommand(c *core.Core, path string, cmd core.Command) {
	if r := c.Command(path, cmd); !r.OK {
		core.Error("failed to register vm command", "path", path, "err", r.Error())
	}
}

func resultFromError(err error) core.Result {
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}

func optionArgs(opts core.Options) []string {
	if r := opts.Get("_args"); r.OK {
		switch args := r.Value.(type) {
		case []string:
			return args
		case []any:
			out := make([]string, 0, len(args))
			for _, arg := range args {
				out = append(out, core.Sprintf("%v", arg))
			}
			return out
		case string:
			if args != "" {
				return core.Split(args, " ")
			}
		}
	}
	if arg := opts.String("_arg"); arg != "" {
		return []string{arg}
	}
	return nil
}

func optionStrings(opts core.Options, key string) []string {
	if r := opts.Get(key); r.OK {
		switch values := r.Value.(type) {
		case []string:
			return values
		case []any:
			out := make([]string, 0, len(values))
			for _, value := range values {
				out = append(out, core.Sprintf("%v", value))
			}
			return out
		case string:
			if values != "" {
				return []string{values}
			}
		}
	}
	return nil
}
