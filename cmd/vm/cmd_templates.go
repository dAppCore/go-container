package vm

import (
	"context"
	"text/tabwriter"

	core "dappco.re/go/core"
	"dappco.re/go/core/container"
	"dappco.re/go/core/container/internal/coreutil"
	"dappco.re/go/core/container/internal/proc"
	"dappco.re/go/core/i18n"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"forge.lthn.ai/core/cli/pkg/cli"
)

// addVMTemplatesCommand adds the 'templates' command under vm.
func addVMTemplatesCommand(parent *cli.Command) {
	templatesCmd := &cli.Command{
		Use:   "templates",
		Short: i18n.T("cmd.vm.templates.short"),
		Long:  i18n.T("cmd.vm.templates.long"),
		RunE: func(cmd *cli.Command, args []string) error {
			return listTemplates()
		},
	}

	// Add subcommands
	addTemplatesShowCommand(templatesCmd)
	addTemplatesVarsCommand(templatesCmd)

	parent.AddCommand(templatesCmd)
}

// addTemplatesShowCommand adds the 'templates show' subcommand.
func addTemplatesShowCommand(parent *cli.Command) {
	showCmd := &cli.Command{
		Use:   "show <template-name>",
		Short: i18n.T("cmd.vm.templates.show.short"),
		Long:  i18n.T("cmd.vm.templates.show.long"),
		RunE: func(cmd *cli.Command, args []string) error {
			if len(args) == 0 {
				return coreerr.E("templates show", i18n.T("cmd.vm.error.template_required"), nil)
			}
			return showTemplate(args[0])
		},
	}

	parent.AddCommand(showCmd)
}

// addTemplatesVarsCommand adds the 'templates vars' subcommand.
func addTemplatesVarsCommand(parent *cli.Command) {
	varsCmd := &cli.Command{
		Use:   "vars <template-name>",
		Short: i18n.T("cmd.vm.templates.vars.short"),
		Long:  i18n.T("cmd.vm.templates.vars.long"),
		RunE: func(cmd *cli.Command, args []string) error {
			if len(args) == 0 {
				return coreerr.E("templates vars", i18n.T("cmd.vm.error.template_required"), nil)
			}
			return showTemplateVars(args[0])
		},
	}

	parent.AddCommand(varsCmd)
}

func listTemplates() error {
	templates := container.ListTemplates()

	if len(templates) == 0 {
		core.Println(i18n.T("cmd.vm.templates.no_templates"))
		return nil
	}

	core.Print(nil, "%s", repoNameStyle.Render(i18n.T("cmd.vm.templates.title")))
	core.Println()

	w := tabwriter.NewWriter(proc.Stdout, 0, 0, 2, ' ', 0)
	core.Print(w, "%s", i18n.T("cmd.vm.templates.header"))
	core.Print(w, "%s", "----\t-----------")

	for _, tmpl := range templates {
		desc := tmpl.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		core.Print(w, "%s\t%s", repoNameStyle.Render(tmpl.Name), desc)
	}
	_ = w.Flush()

	core.Println()
	core.Print(nil, "%s %s", i18n.T("cmd.vm.templates.hint.show"), dimStyle.Render("core vm templates show <name>"))
	core.Print(nil, "%s %s", i18n.T("cmd.vm.templates.hint.vars"), dimStyle.Render("core vm templates vars <name>"))
	core.Print(nil, "%s %s", i18n.T("cmd.vm.templates.hint.run"), dimStyle.Render("core vm run --template <name> --var SSH_KEY=\"...\""))

	return nil
}

func showTemplate(name string) error {
	content, err := container.GetTemplate(name)
	if err != nil {
		return err
	}

	core.Print(nil, "%s %s", dimStyle.Render(i18n.T("common.label.template")), repoNameStyle.Render(name))
	core.Println()
	core.Println(content)

	return nil
}

func showTemplateVars(name string) error {
	content, err := container.GetTemplate(name)
	if err != nil {
		return err
	}

	required, optional := container.ExtractVariables(content)

	core.Print(nil, "%s %s", dimStyle.Render(i18n.T("common.label.template")), repoNameStyle.Render(name))
	core.Println()

	if len(required) > 0 {
		core.Print(nil, "%s", errorStyle.Render(i18n.T("cmd.vm.templates.vars.required")))
		for _, v := range required {
			core.Print(nil, "  %s", varStyle.Render("${"+v+"}"))
		}
		core.Println()
	}

	if len(optional) > 0 {
		core.Print(nil, "%s", successStyle.Render(i18n.T("cmd.vm.templates.vars.optional")))
		for v, def := range optional {
			core.Print(nil, "  %s = %s",
				varStyle.Render("${"+v+"}"),
				defaultStyle.Render(def))
		}
		core.Println()
	}

	if len(required) == 0 && len(optional) == 0 {
		core.Println(i18n.T("cmd.vm.templates.vars.none"))
	}

	return nil
}

// RunFromTemplate builds and runs a LinuxKit image from a template.
//
// Usage:
//
//	err := RunFromTemplate("core-dev", vars, runOpts)
func RunFromTemplate(templateName string, vars map[string]string, runOpts container.RunOptions) error {
	// Apply template with variables
	content, err := container.ApplyTemplate(templateName, vars)
	if err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("common.error.failed", map[string]any{"Action": "apply template"}), err)
	}

	// Create a temporary directory for the build
	tmpDir, err := coreutil.MkdirTemp("core-linuxkit-")
	if err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("common.error.failed", map[string]any{"Action": "create temp directory"}), err)
	}
	defer func() { _ = io.Local.DeleteAll(tmpDir) }()

	// Write the YAML file
	yamlPath := coreutil.JoinPath(tmpDir, core.Concat(templateName, ".yml"))
	if err := io.Local.Write(yamlPath, content); err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("common.error.failed", map[string]any{"Action": "write template"}), err)
	}

	core.Print(nil, "%s %s", dimStyle.Render(i18n.T("common.label.template")), repoNameStyle.Render(templateName))
	core.Print(nil, "%s %s", dimStyle.Render(i18n.T("cmd.vm.label.building")), yamlPath)

	// Build the image using linuxkit
	outputPath := coreutil.JoinPath(tmpDir, templateName)
	if err := buildLinuxKitImage(yamlPath, outputPath); err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("common.error.failed", map[string]any{"Action": "build image"}), err)
	}

	// Find the built image (linuxkit creates .iso or other format)
	imagePath := findBuiltImage(outputPath)
	if imagePath == "" {
		return coreerr.E("RunFromTemplate", i18n.T("cmd.vm.error.no_image_found"), nil)
	}

	core.Print(nil, "%s %s", dimStyle.Render(i18n.T("common.label.image")), imagePath)
	core.Println()

	// Run the image
	manager, err := container.NewLinuxKitManager(io.Local)
	if err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("common.error.failed", map[string]any{"Action": "initialize container manager"}), err)
	}

	core.Print(nil, "%s %s", dimStyle.Render(i18n.T("cmd.vm.label.hypervisor")), manager.Hypervisor().Name())
	core.Println()

	ctx := context.Background()
	c, err := manager.Run(ctx, imagePath, runOpts)
	if err != nil {
		return coreerr.E("RunFromTemplate", i18n.T("i18n.fail.run", "container"), err)
	}

	if runOpts.Detach {
		core.Print(nil, "%s %s", successStyle.Render(i18n.T("common.label.started")), c.ID)
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

// buildLinuxKitImage builds a LinuxKit image from a YAML file.
func buildLinuxKitImage(yamlPath, outputPath string) error {
	// Check if linuxkit is available
	lkPath, err := lookupLinuxKit()
	if err != nil {
		return err
	}

	// Build the image
	// linuxkit build --format iso-bios --name <output> <yaml>
	cmd := proc.NewCommand(lkPath, "build",
		"--format", "iso-bios",
		"--name", outputPath,
		yamlPath)

	cmd.Stdout = proc.Stdout
	cmd.Stderr = proc.Stderr

	return cmd.Run()
}

// findBuiltImage finds the built image file.
func findBuiltImage(basePath string) string {
	// LinuxKit can create different formats
	extensions := []string{".iso", "-bios.iso", ".qcow2", ".raw", ".vmdk"}

	for _, ext := range extensions {
		path := core.Concat(basePath, ext)
		if io.Local.IsFile(path) {
			return path
		}
	}

	// Check directory for any image file
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

// lookupLinuxKit finds the linuxkit binary.
func lookupLinuxKit() (string, error) {
	// Check PATH first
	if path, err := proc.LookPath("linuxkit"); err == nil {
		return path, nil
	}

	// Check common locations
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

// ParseVarFlags parses --var flags into a map.
// Format: --var KEY=VALUE or --var KEY="VALUE"
//
// Usage:
//
//	vars := ParseVarFlags([]string{"SSH_KEY=abc", "PORT=2222"})
func ParseVarFlags(varFlags []string) map[string]string {
	vars := make(map[string]string)

	for _, v := range varFlags {
		parts := core.SplitN(v, "=", 2)
		if len(parts) == 2 {
			key := core.Trim(parts[0])
			value := core.Trim(parts[1])
			// Remove surrounding quotes if present
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
