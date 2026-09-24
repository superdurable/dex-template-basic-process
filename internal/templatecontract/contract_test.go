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
	SchemaVersion                         int               `json:"schemaVersion"`
	BuildProfile                          string            `json:"buildProfile"`
	TemplateVersion                       string            `json:"templateVersion"`
	MinimumSandboxRuntimeContractRevision int               `json:"minimumSandboxRuntimeContractRevision"`
	OpenAPISpec                           string            `json:"openapiSpec"`
	AgentInstructions                     string            `json:"agentInstructions"`
	DexSkill                              string            `json:"dexSkill"`
	Commands                              map[string]string `json:"commands"`
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
	if contract.SchemaVersion != 1 || contract.BuildProfile != "go-react-v1" || contract.TemplateVersion != "1.3.0" || contract.MinimumSandboxRuntimeContractRevision != 2 {
		t.Fatalf("unexpected template identity: %+v", contract)
	}
	if baseline := strings.TrimSpace(readFile(t, filepath.Join(root, "DEX_SERVER_BASELINE"))); baseline != "server/v0.11.4" {
		t.Fatalf("unexpected Dex Server baseline: %q", baseline)
	}
	if baseline := strings.TrimSpace(readFile(t, filepath.Join(root, "DEX_WEB_V2_BASELINE"))); baseline != "dex-web-v2/v0.3.0" {
		t.Fatalf("unexpected Dex Web baseline: %q", baseline)
	}
	goModule := readFile(t, filepath.Join(root, "go.mod"))
	if !strings.Contains(goModule, "github.com/superdurable/dex/sdk-go v0.11.3") {
		t.Fatal("template must pin Dex Go SDK v0.11.3")
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
	if len(fields) < 2 || fields[0] != "160000" || fields[1] != "af3c182de5dc4765b1e402da39eabe5fe5a13c38" {
		t.Fatalf("Dex skill is not pinned as a gitlink: %q", output)
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
