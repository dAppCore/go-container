package container

import (
	"embed"
	"iter"
	"maps"
	"regexp"
	"slices"

	core "dappco.re/go"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"

	"dappco.re/go/container/internal/coreutil"
)

//go:embed templates/*.yml
var embeddedTemplates embed.FS

// Template represents a LinuxKit YAML template.
type Template struct {
	// Name is the template identifier (e.g., "core-dev", "server-php").
	Name string
	// Description is a human-readable description of the template.
	Description string
	// Path is the file path to the template (relative or absolute).
	Path string
}

// builtinTemplates defines the metadata for embedded templates.
var builtinTemplates = []Template{
	{
		Name:        "core-dev",
		Description: "Development environment with Go, Node.js, PHP, Docker-in-LinuxKit, and SSH access",
		Path:        "templates/core-dev.yml",
	},
	{
		Name:        "server-php",
		Description: "Production PHP server with FrankenPHP, Caddy reverse proxy, and health checks",
		Path:        "templates/server-php.yml",
	},
}

// ListTemplates returns all available LinuxKit templates.
// It combines embedded templates with any templates found in the user's
// .core/linuxkit directory.
//
// Usage:
//
//	templates := ListTemplates()
func ListTemplates() []Template {
	return slices.Collect(ListTemplatesIter())
}

// ListTemplatesIter returns an iterator for all available LinuxKit templates.
//
// Usage:
//
//	for template := range ListTemplatesIter() { _ = template }
func ListTemplatesIter() iter.Seq[Template] {
	return func(yield func(Template) bool) {
		// Yield builtin templates
		for _, t := range builtinTemplates {
			if !yield(t) {
				return
			}
		}

		// Check for user templates in .core/linuxkit/
		userTemplatesDir := getUserTemplatesDir()
		if userTemplatesDir != "" {
			for _, t := range scanUserTemplates(userTemplatesDir) {
				if !yield(t) {
					return
				}
			}
		}
	}
}

// GetTemplate returns the content of a template by name.
// It first checks embedded templates, then user templates.
//
// Usage:
//
//	content, err := GetTemplate("core-dev")
func GetTemplate(name string) (
	string,
	error,
) {
	// Check embedded templates first
	for _, t := range builtinTemplates {
		if t.Name == name {
			content, err := embeddedTemplates.ReadFile(t.Path)
			if err != nil {
				return "", coreerr.E("GetTemplate", "failed to read embedded template: "+name, err)
			}
			return string(content), nil
		}
	}

	// Check user templates
	userTemplatesDir := getUserTemplatesDir()
	if userTemplatesDir != "" {
		templatePath := coreutil.JoinPath(userTemplatesDir, core.Concat(name, ".yml"))
		if io.Local.IsFile(templatePath) {
			content, err := io.Local.Read(templatePath)
			if err != nil {
				return "", coreerr.E("GetTemplate", "failed to read user template: "+name, err)
			}
			return content, nil
		}
	}

	return "", coreerr.E("GetTemplate", "template not found: "+name, nil)
}

// ApplyTemplate applies variable substitution to a template.
// It supports two syntaxes:
//   - ${VAR} - required variable, returns error if not provided
//   - ${VAR:-default} - variable with default value
//
// Usage:
//
//	content, err := ApplyTemplate("core-dev", vars)
func ApplyTemplate(name string, vars map[string]string) (
	string,
	error,
) {
	content, err := GetTemplate(name)
	if err != nil {
		return "", err
	}

	return ApplyVariables(content, vars)
}

// ApplyVariables applies variable substitution to content string.
// It supports two syntaxes:
//   - ${VAR} - required variable, returns error if not provided
//   - ${VAR:-default} - variable with default value
//
// Usage:
//
//	content, err := ApplyVariables(raw, vars)
func ApplyVariables(content string, vars map[string]string) (
	string,
	error,
) {
	// Pattern for ${VAR:-default} syntax
	defaultPattern := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*):-([^}]*)\}`)

	// Pattern for ${VAR} syntax (no default)
	requiredPattern := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

	// Track missing required variables
	var missingVars []string

	// First pass: replace variables with defaults
	result := defaultPattern.ReplaceAllStringFunc(content, func(match string) string {
		submatch := defaultPattern.FindStringSubmatch(match)
		if len(submatch) != 3 {
			return match
		}
		varName := submatch[1]
		defaultVal := submatch[2]

		if val, ok := vars[varName]; ok {
			return val
		}
		return defaultVal
	})

	// Second pass: replace required variables and track missing ones
	result = requiredPattern.ReplaceAllStringFunc(result, func(match string) string {
		submatch := requiredPattern.FindStringSubmatch(match)
		if len(submatch) != 2 {
			return match
		}
		varName := submatch[1]

		if val, ok := vars[varName]; ok {
			return val
		}
		missingVars = append(missingVars, varName)
		return match // Keep original if missing
	})

	if len(missingVars) > 0 {
		return "", coreerr.E("ApplyVariables", core.Concat("missing required variables: ", core.Join(", ", missingVars...)), nil)
	}

	return result, nil
}

// ExtractVariables extracts all variable names from a template.
// Returns two slices: required variables and optional variables (with defaults).
//
// Usage:
//
//	required, optional := ExtractVariables(content)
func ExtractVariables(content string) (required []string, optional map[string]string) {
	optional = make(map[string]string)
	requiredSet := make(map[string]bool)

	// Pattern for ${VAR:-default} syntax
	defaultPattern := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*):-([^}]*)\}`)

	// Pattern for ${VAR} syntax (no default)
	requiredPattern := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

	// Find optional variables with defaults
	matches := defaultPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			optional[match[1]] = match[2]
		}
	}

	// Find required variables
	matches = requiredPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 2 {
			varName := match[1]
			// Only add if not already in optional (with default)
			if _, hasDefault := optional[varName]; !hasDefault {
				requiredSet[varName] = true
			}
		}
	}

	// Convert set to slice
	required = slices.Sorted(maps.Keys(requiredSet))

	return required, optional
}

// getUserTemplatesDir returns the path to user templates directory.
// Returns empty string if the directory doesn't exist.
func getUserTemplatesDir() string {
	// Try workspace-relative .core/linuxkit first
	wsDir := coreutil.JoinPath(coreutil.CurrentDir(), ".core", "linuxkit")
	if io.Local.IsDir(wsDir) {
		return wsDir
	}

	// Try home directory
	home := coreutil.HomeDir()
	if home == "" {
		return ""
	}

	homeDir := coreutil.JoinPath(home, ".core", "linuxkit")
	if io.Local.IsDir(homeDir) {
		return homeDir
	}

	return ""
}

// scanUserTemplates scans a directory for .yml template files.
func scanUserTemplates(dir string) []Template {
	var templates []Template

	entries, err := io.Local.List(dir)
	if err != nil {
		return templates
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !core.HasSuffix(name, ".yml") && !core.HasSuffix(name, ".yaml") {
			continue
		}

		// Extract template name from filename
		templateName := core.TrimSuffix(core.TrimSuffix(name, ".yml"), ".yaml")

		// Skip if this is a builtin template name (embedded takes precedence)
		isBuiltin := false
		for _, bt := range builtinTemplates {
			if bt.Name == templateName {
				isBuiltin = true
				break
			}
		}
		if isBuiltin {
			continue
		}

		// Read file to extract description from comments
		description := extractTemplateDescription(coreutil.JoinPath(dir, name))
		if description == "" {
			description = "User-defined template"
		}

		templates = append(templates, Template{
			Name:        templateName,
			Description: description,
			Path:        coreutil.JoinPath(dir, name),
		})
	}

	return templates
}

// extractTemplateDescription reads the first comment block from a YAML file
// to use as a description.
func extractTemplateDescription(path string) string {
	content, err := io.Local.Read(path)
	if err != nil {
		return ""
	}

	lines := core.Split(content, "\n")
	var descLines []string

	for _, line := range lines {
		trimmed := core.Trim(line)
		if core.HasPrefix(trimmed, "#") {
			// Remove the # and trim
			comment := core.Trim(core.TrimPrefix(trimmed, "#"))
			if comment != "" {
				descLines = append(descLines, comment)
				// Only take the first meaningful comment line as description
				if len(descLines) == 1 {
					return comment
				}
			}
		} else if trimmed != "" {
			// Hit non-comment content, stop
			break
		}
	}

	if len(descLines) > 0 {
		return descLines[0]
	}
	return ""
}
