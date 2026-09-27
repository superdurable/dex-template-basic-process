package templatecontract_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type manifest struct {
	SchemaVersion                       int               `json:"schemaVersion"`
	BuildProfile                        string            `json:"buildProfile"`
	TemplateVersion                     string            `json:"templateVersion"`
	MinimumSandboxImageContractRevision int               `json:"minimumSandboxImageContractRevision"`
	OpenAPISpec                         string            `json:"openapiSpec"`
	AgentInstructions                   string            `json:"agentInstructions"`
	DexSkill                            string            `json:"dexSkill"`
	Commands                            map[string]string `json:"commands"`
}

func TestTemplateContract(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	manifestBytes, err := os.ReadFile(filepath.Join(root, ".superverse", "template.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var contract manifest
	if err := json.Unmarshal(manifestBytes, &contract); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if contract.SchemaVersion != 1 || contract.BuildProfile != "go-react-v1" || contract.TemplateVersion != "1.5.1" || contract.MinimumSandboxImageContractRevision != 2 {
		t.Fatalf("unexpected template identity: %+v", contract)
	}
	if baseline := strings.TrimSpace(readFile(t, filepath.Join(root, "DEX_SERVER_BASELINE"))); baseline != "server/v0.13.2" {
		t.Fatalf("unexpected Dex Server baseline: %q", baseline)
	}
	if baseline := strings.TrimSpace(readFile(t, filepath.Join(root, "DEX_CLI_BASELINE"))); baseline != "cli-v0.13.8" {
		t.Fatalf("unexpected Dex CLI baseline: %q", baseline)
	}
	goModule := readFile(t, filepath.Join(root, "go.mod"))
	if !strings.Contains(goModule, "github.com/superdurable/dex/sdk-go v0.13.1") {
		t.Fatal("template must pin Dex Go SDK v0.13.1")
	}
	for _, path := range []string{contract.OpenAPISpec, contract.AgentInstructions, contract.DexSkill} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Errorf("manifest path %q: %v", path, err)
		}
	}
	makefile := readFile(t, filepath.Join(root, "Makefile"))
	agents := readFile(t, filepath.Join(root, contract.AgentInstructions))
	for _, command := range contract.Commands {
		target := strings.TrimPrefix(command, "make ")
		if !strings.Contains(makefile, "\n"+target+":") && !strings.HasPrefix(makefile, target+":") {
			t.Errorf("Makefile does not define %q", target)
		}
		if !strings.Contains(agents, command) {
			t.Errorf("AGENTS.md does not mention %q", command)
		}
	}
	gitmodules := readFile(t, filepath.Join(root, ".gitmodules"))
	if !strings.Contains(gitmodules, "path = .agents/skills/dex-app-builder/upstream") ||
		!strings.Contains(gitmodules, "url = https://github.com/superdurable/dex-skills.git") {
		t.Fatal("Dex skill submodule path or public HTTPS URL is not allowlisted")
	}
	if _, err := os.Stat(filepath.Join(root, ".agents/skills/dex-sdk/SKILL.md")); err != nil {
		t.Fatalf("Dex SDK wrapper: %v", err)
	}
	command := exec.Command("git", "ls-files", "--stage", ".agents/skills/dex-app-builder/upstream")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("read Dex skill submodule pin: %v", err)
	}
	fields := strings.Fields(string(output))
	if len(fields) < 2 || fields[0] != "160000" || fields[1] != "664a8d1c376fbcdd9d124636699b3fa84770bb0d" {
		t.Fatalf("Dex skill is not pinned as a gitlink: %q", output)
	}

	dependencyUpdater := readFile(t, filepath.Join(root, "scripts", "update-dex-dependencies.py"))
	for _, required := range []string{
		"github.com/superdurable/dex/sdk-go",
		"DEX_SERVER_BASELINE",
		"DEX_CLI_BASELINE",
		".agents/skills/dex-app-builder/upstream",
		"templateVersion",
		"skills_pin_outdated",
		"GITHUB_OUTPUT",
	} {
		if !strings.Contains(dependencyUpdater, required) {
			t.Errorf("Dex dependency updater does not mention %q", required)
		}
	}
	updateWorkflow := readFile(t, filepath.Join(root, ".github", "workflows", "update-dex-dependencies.yml"))
	for _, required := range []string{
		"schedule:",
		"workflow_dispatch:",
		"pull-requests: write",
		"automation/update-dex-dependencies",
		"gh release view --repo superdurable/dex-skills",
		"gh pr create",
		"gh workflow run ci.yml",
	} {
		if !strings.Contains(updateWorkflow, required) {
			t.Errorf("Dex dependency update workflow does not mention %q", required)
		}
	}
	ciWorkflow := readFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"))
	if !strings.Contains(ciWorkflow, "workflow_dispatch:") {
		t.Fatal("Template CI must support explicit dispatch from dependency update PRs")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}
