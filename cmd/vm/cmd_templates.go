package vm

import (
	"context"
	"text/tabwriter"

	"dappco.re/go/core"
	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/container"
	"dappco.re/go/core/container/internal/coreutil"
	"dappco.re/go/core/container/internal/proc"
	"dappco.re/go/core/i18n"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// addVMTemplatesCommand registers the vm/templates command tree:
//
//	vm/templates             → list templates
//	vm/templates/show        → show template content (--name required)
//	vm/templates/vars        → show required/optional vars (--name required)
func addVMTemplatesCommand(c *core.Core) {
	c.Command("vm/templates", core.Command{
		Description: "cmd.vm.templates.long",
		Action: func(_ core.Options) core.Result {
			return resultFromError(listTemplates())
		},
	})

	c.Command("vm/templates/show", core.Command{
		Description: "cmd.vm.templates.show.long",
		Action: func(opts core.Options) core.Result {
			name := opts.String("_arg")
			if name == "" {
				name = opts.String("name")
			}
			if name == "" {
				return resultFromError(coreerr.E("templates show", i18n.T("cmd.vm.error.template_required"), nil))
			}
			return resultFromError(showTemplate(name))
		},
	})

	c.Command("vm/templates/vars", core.Command{
		Description: "cmd.vm.templates.vars.long",
		Action: func(opts core.Options) core.Result {
			name := opts.String("_arg")
			if name == "" {
				name = opts.String("name")
			}
			if name == "" {
				return resultFromError(coreerr.E("templates vars", i18n.T("cmd.vm.error.template_required"), nil))
			}
			return resultFromError(showTemplateVars(name))
		},
	})
}

func listTemplates() error {
	templates := container.ListTemplates()

	if len(templates) == 0 {
		cli.Println("%s", i18n.T("cmd.vm.templates.no_templates"))
		return nil
	}

	cli.Print("%s\n", repoNameStyle.Render(i18n.T("cmd.vm.templates.title")))

	w := tabwriter.NewWriter(proc.Stdout, 0, 0, 2, ' ', 0)
	core.Print(w, "%s\n", i18n.T("cmd.vm.templates.header"))
	core.Print(w, "%s\n", "----\t-----------")

	for _, tmpl := range templates {
		desc := tmpl.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		core.Print(w, "%s\t%s\n", repoNameStyle.Render(tmpl.Name), desc)
	}
	_ = w.Flush()

	cli.Println("")
	cli.Print("%s %s\n", i18n.T("cmd.vm.templates.hint.show"), dimStyle.Render("core vm templates show <name>"))
	cli.Print("%s %s\n", i18n.T("cmd.vm.templates.hint.vars"), dimStyle.Render("core vm templates vars <name>"))
	cli.Print("%s %s\n", i18n.T("cmd.vm.templates.hint.run"), dimStyle.Render("core vm run --template <name> --var SSH_KEY=\"...\""))

	return nil
}

func showTemplate(name string) error {
	content, err := container.GetTemplate(name)
	if err != nil {
		return err
	}

	cli.Print("%s %s\n", dimStyle.Render(i18n.T("common.label.template")), repoNameStyle.Render(name))
	cli.Println("%s", content)

	return nil
}

func showTemplateVars(name string) error {
	content, err := container.GetTemplate(name)
	if err != nil {
		return err
	}

	required, optional := container.ExtractVariables(content)

	cli.Print("%s %s\n", dimStyle.Render(i18n.T("common.label.template")), repoNameStyle.Render(name))

	if len(required) > 0 {
		cli.Print("%s\n", errorStyle.Render(i18n.T("cmd.vm.templates.vars.required")))
		for _, v := range required {
			cli.Print("  %s\n", varStyle.Render("${"+v+"}"))
		}
	}

	if len(optional) > 0 {
		cli.Print("%s\n", successStyle.Render(i18n.T("cmd.vm.templates.vars.optional")))
		for v, def := range optional {
			cli.Print("  %s = %s\n",
				varStyle.Render("${"+v+"}"),
				defaultStyle.Render(def))
		}
	}

	if len(required) == 0 && len(optional) == 0 {
		cli.Println("%s", i18n.T("cmd.vm.templates.vars.none"))
	}

	return nil
}

// RunFromTemplate builds and runs a LinuxKit image from a template.
//
//	err := RunFromTemplate("core-dev", map[string]string{"SSH_KEY": "..."}, runOpts)
func RunFromTemplate(templateName string, vars map[string]string, runOpts container.RunOptions) error {
	content, err := container.ApplyTemplate(templateName, vars)
	if err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("common.error.failed", map[string]any{"Action": "apply template"}), err)
	}

	tmpDir, err := coreutil.MkdirTemp("core-linuxkit-")
	if err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("common.error.failed", map[string]any{"Action": "create temp directory"}), err)
	}
	defer func() { _ = io.Local.DeleteAll(tmpDir) }()

	yamlPath := coreutil.JoinPath(tmpDir, core.Concat(templateName, ".yml"))
	if err := io.Local.Write(yamlPath, content); err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("common.error.failed", map[string]any{"Action": "write template"}), err)
	}

	cli.Print("%s %s\n", dimStyle.Render(i18n.T("common.label.template")), repoNameStyle.Render(templateName))
	cli.Print("%s %s\n", dimStyle.Render(i18n.T("cmd.vm.label.building")), yamlPath)

	outputPath := coreutil.JoinPath(tmpDir, templateName)
	if err := buildLinuxKitImage(yamlPath, outputPath); err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("common.error.failed", map[string]any{"Action": "build image"}), err)
	}

	imagePath := findBuiltImage(outputPath)
	if imagePath == "" {
		return coreerr.E("RunFromTemplate", i18n.T("cmd.vm.error.no_image_found"), nil)
	}

	cli.Print("%s %s\n", dimStyle.Render(i18n.T("common.label.image")), imagePath)

	manager, err := container.NewLinuxKitManager(io.Local)
	if err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("common.error.failed", map[string]any{"Action": "initialize container manager"}), err)
	}

	cli.Print("%s %s\n", dimStyle.Render(i18n.T("cmd.vm.label.hypervisor")), manager.Hypervisor().Name())

	ctx := context.Background()
	c, err := manager.Run(ctx, imagePath, runOpts)
	if err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("i18n.fail.run", "container"), err)
	}

	if runOpts.Detach {
		cli.Print("%s %s\n", successStyle.Render(i18n.T("common.label.started")), c.ID)
		cli.Print("%s %d\n", dimStyle.Render(i18n.T("cmd.vm.label.pid")), c.PID)
		cli.Println("%s", i18n.T("cmd.vm.hint.view_logs", map[string]any{"ID": c.ID[:8]}))
		cli.Println("%s", i18n.T("cmd.vm.hint.stop", map[string]any{"ID": c.ID[:8]}))
	} else {
		cli.Print("%s %s\n", dimStyle.Render(i18n.T("cmd.vm.label.container_stopped")), c.ID)
	}

	return nil
}

// buildLinuxKitImage runs `linuxkit build --format iso-bios --name <output> <yaml>`.
func buildLinuxKitImage(yamlPath, outputPath string) error {
	lkPath, err := lookupLinuxKit()
	if err != nil {
		return err
	}

	cmd := proc.NewCommand(lkPath, "build",
		"--format", "iso-bios",
		"--name", outputPath,
		yamlPath)

	cmd.Stdout = proc.Stdout
	cmd.Stderr = proc.Stderr

	return cmd.Run()
}

// findBuiltImage locates the built image file produced by linuxkit.
func findBuiltImage(basePath string) string {
	extensions := []string{".iso", "-bios.iso", ".qcow2", ".raw", ".vmdk"}

	for _, ext := range extensions {
		path := core.Concat(basePath, ext)
		if io.Local.IsFile(path) {
			return path
		}
	}

	dir := core.PathDir(basePath)
	base := core.PathBase(basePath)

	entries, err := io.Local.List(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		name := entry.Name()
		if core.HasPrefix(name, base) {
			for _, ext := range []string{".iso", ".qcow2", ".raw", ".vmdk"} {
				if core.HasSuffix(name, ext) {
					return coreutil.JoinPath(dir, name)
				}
			}
		}
	}

	return ""
}

// lookupLinuxKit finds the linuxkit binary on PATH or in well-known locations.
func lookupLinuxKit() (string, error) {
	if path, err := proc.LookPath("linuxkit"); err == nil {
		return path, nil
	}

	paths := []string{
		"/usr/local/bin/linuxkit",
		"/opt/homebrew/bin/linuxkit",
	}

	for _, p := range paths {
		if io.Local.Exists(p) {
			return p, nil
		}
	}

	return "", coreerr.E("lookupLinuxKit", i18n.T("cmd.vm.error.linuxkit_not_found"), nil)
}

// ParseVarFlags parses repeated --var flags into a map. Accepted form:
//
//	--var KEY=VALUE
//	--var KEY="VALUE"
//
//	vars := ParseVarFlags([]string{"SSH_KEY=abc", "PORT=2222"})
func ParseVarFlags(varFlags []string) map[string]string {
	vars := make(map[string]string)

	for _, v := range varFlags {
		parts := core.SplitN(v, "=", 2)
		if len(parts) == 2 {
			key := core.Trim(parts[0])
			value := core.Trim(parts[1])
			value = stripWrappingQuotes(value)
			vars[key] = value
		}
	}

	return vars
}

func stripWrappingQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	if core.HasPrefix(value, "\"") && core.HasSuffix(value, "\"") {
		return core.TrimSuffix(core.TrimPrefix(value, "\""), "\"")
	}
	if core.HasPrefix(value, "'") && core.HasSuffix(value, "'") {
		return core.TrimSuffix(core.TrimPrefix(value, "'"), "'")
	}
	return value
}
