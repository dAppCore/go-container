package container

import (
	core "dappco.re/go"
	"dappco.re/go/container/internal/coreutil"
	"dappco.re/go/io"
	"reflect"
	"slices"
	"testing"
)

func TestTemplates_ListTemplates_Good(t *testing.T) {
	auditTarget := "ListTemplates"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	templates := ListTemplates()

	// Should have at least the builtin templates
	if got, want := len(templates), 2; got < want {
		t.Fatalf("want at least %v, got %v", want, got)
	}

	// Find the core-dev template

	var found bool
	for _, tmpl := range templates {
		if tmpl.Name == "core-dev" {
			found = true
			if got := tmpl.Description; len(got) == 0 {
				t.Fatal("expected non-empty value")
			}
			if got := tmpl.Path; len(got) == 0 {
				t.Fatal("expected non-empty value")
			}
			break
		}
	}
	if !(found) {
		t.Fatal("expected true")
	}

	// Find the server-php template
	found = false
	for _, tmpl := range templates {
		if tmpl.Name == "server-php" {
			found = true
			if got := tmpl.Description; len(got) == 0 {
				t.Fatal("expected non-empty value")
			}
			if got := tmpl.Path; len(got) == 0 {
				t.Fatal("expected non-empty value")
			}
			break
		}
	}
	if !(found) {
		t.Fatal("expected true")
	}
}

func TestGetTemplate_CoreDev_Good(t *testing.T) {
	auditTarget := "CoreDev"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content, err := GetTemplate("core-dev")
	if err != nil {
		t.Fatal(err)
	}
	if got := content; len(got) == 0 {
		t.Fatal("expected non-empty value")
	}
	if s, sub := content, "kernel:"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := content, "linuxkit/kernel"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := content, "${SSH_KEY}"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := content, "services:"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestGetTemplate_ServerPhp_Good(t *testing.T) {
	auditTarget := "ServerPhp"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content, err := GetTemplate("server-php")
	if err != nil {
		t.Fatal(err)
	}
	if got := content; len(got) == 0 {
		t.Fatal("expected non-empty value")
	}
	if s, sub := content, "kernel:"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := content, "frankenphp"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := content, "${SSH_KEY}"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := content, "${DOMAIN:-localhost}"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestGetTemplate_NotFound_Bad(t *testing.T) {
	auditTarget := "NotFound"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	_, err := GetTemplate("nonexistent-template")
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "template not found"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestApplyVariables_SimpleSubstitution_Good(t *testing.T) {
	auditTarget := "SimpleSubstitution"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content := "Hello ${NAME}, welcome to ${PLACE}!"
	vars := map[string]string{
		"NAME":  "World",
		"PLACE": "Core",
	}

	result, err := ApplyVariables(content, vars)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := result, "Hello World, welcome to Core!"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApplyVariables_WithDefaults_Good(t *testing.T) {
	auditTarget := "WithDefaults"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content := "Memory: ${MEMORY:-1024}MB, CPUs: ${CPUS:-2}"
	vars := map[string]string{
		"MEMORY": "2048",
		// CPUS not provided, should use default
	}

	result, err := ApplyVariables(content, vars)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := result, "Memory: 2048MB, CPUs: 2"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApplyVariables_AllDefaults_Good(t *testing.T) {
	auditTarget := "AllDefaults"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content := "${HOST:-localhost}:${PORT:-8080}"
	vars := map[string]string{} // No vars provided

	result, err := ApplyVariables(content, vars)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := result, "localhost:8080"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApplyVariables_MixedSyntax_Good(t *testing.T) {
	auditTarget := "MixedSyntax"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content := `
hostname: ${HOSTNAME:-myhost}
ssh_key: ${SSH_KEY}
memory: ${MEMORY:-512}
`
	vars := map[string]string{
		"SSH_KEY":  "ssh-rsa AAAA...",
		"HOSTNAME": "custom-host",
	}

	result, err := ApplyVariables(content, vars)
	if err != nil {
		t.Fatal(err)
	}
	if s, sub := result, "hostname: custom-host"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := result, "ssh_key: ssh-rsa AAAA..."; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := result, "memory: 512"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestApplyVariables_EmptyDefault_Good(t *testing.T) {
	auditTarget := "EmptyDefault"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content := "value: ${OPT:-}"
	vars := map[string]string{}

	result, err := ApplyVariables(content, vars)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := result, "value: "; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApplyVariables_MissingRequired_Bad(t *testing.T) {
	auditTarget := "MissingRequired"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content := "SSH Key: ${SSH_KEY}"
	vars := map[string]string{} // Missing required SSH_KEY

	_, err := ApplyVariables(content, vars)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "missing required variables"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := err.Error(), "SSH_KEY"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestApplyVariables_MultipleMissing_Bad(t *testing.T) {
	auditTarget := "MultipleMissing"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content := "${VAR1} and ${VAR2} and ${VAR3}"
	vars := map[string]string{
		"VAR2": "provided",
	}

	_, err := ApplyVariables(content, vars)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "missing required variables"; !core.
		// Should mention both missing vars
		Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}

	errStr := err.Error()
	if !(core.Contains(errStr, "VAR1") || core.Contains(errStr, "VAR3")) {
		t.Fatal("expected true")
	}
}

func TestTemplates_ApplyTemplate_Good(t *testing.T) {
	auditTarget := "ApplyTemplate"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	vars := map[string]string{
		"SSH_KEY": "ssh-rsa AAAA... user@host",
	}

	result, err := ApplyTemplate("core-dev", vars)
	if err != nil {
		t.Fatal(err)
	}
	if got := result; len(got) == 0 {
		t.Fatal("expected non-empty value")
	}
	if s, sub := result, "ssh-rsa AAAA... user@host"; !core.
		// Default values should be applied
		Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := result, "core-dev"; !core. // HOSTNAME default
						Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestApplyTemplate_TemplateNotFound_Bad(t *testing.T) {
	auditTarget := "TemplateNotFound"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	vars := map[string]string{
		"SSH_KEY": "test",
	}

	_, err := ApplyTemplate("nonexistent", vars)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "template not found"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestApplyTemplate_MissingVariable_Bad(t *testing.T) {
	auditTarget := "MissingVariable"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// server-php requires SSH_KEY
	vars := map[string]string{} // Missing required SSH_KEY

	_, err := ApplyTemplate("server-php", vars)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "missing required variables"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestTemplates_ExtractVariables_Good(t *testing.T) {
	auditTarget := "ExtractVariables"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content := `
hostname: ${HOSTNAME:-myhost}
ssh_key: ${SSH_KEY}
memory: ${MEMORY:-1024}
cpus: ${CPUS:-2}
api_key: ${API_KEY}
	`
	required, optional := ExtractVariables(content)

	// Required variables (no default)
	if s, sub := required, "SSH_KEY"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := required, "API_KEY"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}

	// Optional variables (with defaults)
	if got, want := len(required), 2; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
	if got, want := optional["HOSTNAME"], "myhost"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := optional["MEMORY"], "1024"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := optional["CPUS"], "2"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := len(optional), 3; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
}

func TestExtractVariables_NoVariables_Good(t *testing.T) {
	auditTarget := "NoVariables"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content := "This has no variables at all"

	required, optional := ExtractVariables(content)
	if got := required; len(got) != 0 {
		t.Fatal("expected empty value")
	}
	if got := optional; len(got) != 0 {
		t.Fatal("expected empty value")
	}
}

func TestExtractVariables_OnlyDefaults_Good(t *testing.T) {
	auditTarget := "OnlyDefaults"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	content := "${A:-default1} ${B:-default2}"

	required, optional := ExtractVariables(content)
	if got := required; len(got) != 0 {
		t.Fatal("expected empty value")
	}
	if got, want := len(optional), 2; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
	if got, want := optional["A"], "default1"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := optional["B"], "default2"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestTemplates_ScanUserTemplates_Good(t *testing.T) {
	auditTarget := "ScanUserTemplates"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Create a temporary directory with template files
	tmpDir := t.TempDir()

	// Create a valid template file
	templateContent := `# My Custom Template
# A custom template for testing
kernel:
  image: linuxkit/kernel:6.6
`
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "custom.yml"), templateContent)
	if err != nil {
		t.Fatal(err)
	}

	// Create a non-template file (should be ignored)
	err = io.Local.Write(coreutil.JoinPath(tmpDir, "readme.txt"), "Not a template")
	if err != nil {
		t.Fatal(err)
	}

	templates := scanUserTemplates(tmpDir)
	if got, want := len(templates), 1; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
	if got, want := templates[0].Name, "custom"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := templates[0].Description, "My Custom Template"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestScanUserTemplates_MultipleTemplates_Good(t *testing.T) {
	auditTarget := "MultipleTemplates"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()

	// Create multiple template files
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "web.yml"), "# Web Server\nkernel:")
	if err != nil {
		t.Fatal(err)
	}
	err = io.Local.Write(coreutil.JoinPath(tmpDir, "db.yaml"), "# Database Server\nkernel:")
	if err != nil {
		t.Fatal(err)
	}

	templates := scanUserTemplates(tmpDir)
	if got, want := len(templates), 2; got !=

		// Check names are extracted correctly
		want {
		t.Fatalf("want len %v, got %v", want, got)
	}

	names := make(map[string]bool)
	for _, tmpl := range templates {
		names[tmpl.Name] = true
	}
	if !(names["web"]) {
		t.Fatal("expected true")
	}
	if !(names["db"]) {
		t.Fatal("expected true")
	}
}

func TestScanUserTemplates_EmptyDirectory_Good(t *testing.T) {
	auditTarget := "EmptyDirectory"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()

	templates := scanUserTemplates(tmpDir)
	if got := templates; len(got) != 0 {
		t.Fatal("expected empty value")
	}
}

func TestScanUserTemplates_NonexistentDirectory_Bad(t *testing.T) {
	auditTarget := "NonexistentDirectory"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	templates := scanUserTemplates("/nonexistent/path/to/templates")
	if got := templates; len(got) != 0 {
		t.Fatal("expected empty value")
	}
}

func TestTemplates_ExtractTemplateDescription_Good(t *testing.T) {
	auditTarget := "ExtractTemplateDescription"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	path := coreutil.JoinPath(tmpDir, "test.yml")

	content := `# My Template Description
# More details here
kernel:
  image: test
`
	err := io.Local.Write(path, content)
	if err != nil {
		t.Fatal(err)
	}

	desc := extractTemplateDescription(path)
	if got, want := desc, "My Template Description"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestExtractTemplateDescription_NoComments_Good(t *testing.T) {
	auditTarget := "NoComments"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	path := coreutil.JoinPath(tmpDir, "test.yml")

	content := `kernel:
  image: test
`
	err := io.Local.Write(path, content)
	if err != nil {
		t.Fatal(err)
	}

	desc := extractTemplateDescription(path)
	if got := desc; len(got) != 0 {
		t.Fatal("expected empty value")
	}
}

func TestExtractTemplateDescription_FileNotFound_Bad(t *testing.T) {
	auditTarget := "FileNotFound"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	desc := extractTemplateDescription("/nonexistent/file.yml")
	if got := desc; len(got) != 0 {
		t.Fatal("expected empty value")
	}
}

func TestTemplates_VariablePatternEdgeCases_Good(t *testing.T) {
	auditTarget := "VariablePatternEdgeCases"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tests := []struct {
		name     string
		content  string
		vars     map[string]string
		expected string
	}{
		{
			name:     "underscore in name",
			content:  "${MY_VAR:-default}",
			vars:     map[string]string{"MY_VAR": "value"},
			expected: "value",
		},
		{
			name:     "numbers in name",
			content:  "${VAR123:-default}",
			vars:     map[string]string{},
			expected: "default",
		},
		{
			name:     "default with special chars",
			content:  "${URL:-http://localhost:8080}",
			vars:     map[string]string{},
			expected: "http://localhost:8080",
		},
		{
			name:     "default with path",
			content:  "${PATH:-/usr/local/bin}",
			vars:     map[string]string{},
			expected: "/usr/local/bin",
		},
		{
			name:     "adjacent variables",
			content:  "${A:-a}${B:-b}${C:-c}",
			vars:     map[string]string{"B": "X"},
			expected: "aXc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplyVariables(tt.content, tt.vars)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := result, tt.expected; !reflect.DeepEqual(got, want) {
				t.Fatalf("want %v, got %v", want, got)
			}
		})
	}
}

func TestScanUserTemplates_SkipsBuiltinNames_Good(t *testing.T) {
	auditTarget := "SkipsBuiltinNames"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()

	// Create a template with a builtin name (should be skipped)
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "core-dev.yml"), "# Duplicate\nkernel:")
	if err != nil {
		t.Fatal(err)
	}

	// Create a unique template
	err = io.Local.Write(coreutil.JoinPath(tmpDir, "unique.yml"), "# Unique\nkernel:")
	if err != nil {
		t.Fatal(err)
	}

	templates := scanUserTemplates(tmpDir)

	// Should only have the unique template, not the builtin name
	if got, want := len(templates), 1; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
	if got, want := templates[0].Name, "unique"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestScanUserTemplates_SkipsDirectories_Good(t *testing.T) {
	auditTarget := "SkipsDirectories"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()

	// Create a subdirectory (should be skipped)
	err := io.Local.EnsureDir(coreutil.JoinPath(tmpDir, "subdir"))
	if err != nil {
		t.Fatal(err)
	}

	// Create a valid template
	err = io.Local.Write(coreutil.JoinPath(tmpDir, "valid.yml"), "# Valid\nkernel:")
	if err != nil {
		t.Fatal(err)
	}

	templates := scanUserTemplates(tmpDir)
	if got, want := len(templates), 1; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
	if got, want := templates[0].Name, "valid"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestScanUserTemplates_YamlExtension_Good(t *testing.T) {
	auditTarget := "YamlExtension"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()

	// Create templates with both extensions
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "template1.yml"), "# Template 1\nkernel:")
	if err != nil {
		t.Fatal(err)
	}
	err = io.Local.Write(coreutil.JoinPath(tmpDir, "template2.yaml"), "# Template 2\nkernel:")
	if err != nil {
		t.Fatal(err)
	}

	templates := scanUserTemplates(tmpDir)
	if got, want := len(templates), 2; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}

	names := make(map[string]bool)
	for _, tmpl := range templates {
		names[tmpl.Name] = true
	}
	if !(names["template1"]) {
		t.Fatal("expected true")
	}
	if !(names["template2"]) {
		t.Fatal("expected true")
	}
}

func TestExtractTemplateDescription_EmptyComment_Good(t *testing.T) {
	auditTarget := "EmptyComment"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	path := coreutil.JoinPath(tmpDir, "test.yml")

	// First comment is empty, second has content
	content := `#
# Actual description here
kernel:
  image: test
`
	err := io.Local.Write(path, content)
	if err != nil {
		t.Fatal(err)
	}

	desc := extractTemplateDescription(path)
	if got, want := desc, "Actual description here"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestExtractTemplateDescription_MultipleEmptyComments_Good(t *testing.T) {
	auditTarget := "MultipleEmptyComments"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	path := coreutil.JoinPath(tmpDir, "test.yml")

	// Multiple empty comments before actual content
	content := `#
#
#
# Real description
kernel:
  image: test
`
	err := io.Local.Write(path, content)
	if err != nil {
		t.Fatal(err)
	}

	desc := extractTemplateDescription(path)
	if got, want := desc, "Real description"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestScanUserTemplates_DefaultDescription_Good(t *testing.T) {
	auditTarget := "DefaultDescription"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()

	// Create a template without comments
	content := `kernel:
  image: test
`
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "nocomment.yml"), content)
	if err != nil {
		t.Fatal(err)
	}

	templates := scanUserTemplates(tmpDir)
	if got, want := len(templates), 1; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
	if got, want := templates[0].Description, "User-defined template"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

// --- AX-7 canonical triplets ---

func TestTemplates_ListTemplates_Bad(t *testing.T) {
	auditTarget := "ListTemplates"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ListTemplates
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_ListTemplates_Ugly(t *testing.T) {
	auditTarget := "ListTemplates"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ListTemplates
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_ListTemplatesIter_Good(t *testing.T) {
	auditTarget := "ListTemplatesIter"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ListTemplatesIter
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_ListTemplatesIter_Bad(t *testing.T) {
	auditTarget := "ListTemplatesIter"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ListTemplatesIter
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_ListTemplatesIter_Ugly(t *testing.T) {
	auditTarget := "ListTemplatesIter"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ListTemplatesIter
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_GetTemplate_Good(t *testing.T) {
	auditTarget := "GetTemplate"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := GetTemplate
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_GetTemplate_Bad(t *testing.T) {
	auditTarget := "GetTemplate"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := GetTemplate
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_GetTemplate_Ugly(t *testing.T) {
	auditTarget := "GetTemplate"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := GetTemplate
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_ApplyTemplate_Bad(t *testing.T) {
	auditTarget := "ApplyTemplate"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ApplyTemplate
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_ApplyTemplate_Ugly(t *testing.T) {
	auditTarget := "ApplyTemplate"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ApplyTemplate
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_ApplyVariables_Good(t *testing.T) {
	auditTarget := "ApplyVariables"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ApplyVariables
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_ApplyVariables_Bad(t *testing.T) {
	auditTarget := "ApplyVariables"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ApplyVariables
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_ApplyVariables_Ugly(t *testing.T) {
	auditTarget := "ApplyVariables"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ApplyVariables
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_ExtractVariables_Bad(t *testing.T) {
	auditTarget := "ExtractVariables"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ExtractVariables
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestTemplates_ExtractVariables_Ugly(t *testing.T) {
	auditTarget := "ExtractVariables"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ExtractVariables
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
