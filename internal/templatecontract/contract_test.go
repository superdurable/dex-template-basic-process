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
	if contract.SchemaVersion != 1 || contract.BuildProfile != "go-react-v1" || contract.MinimumSandboxRuntimeContractRevision != 2 {
		t.Fatalf("unexpected template identity: %+v", contract)
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
	if !strings.Contains(gitmodules, "path = .agents/skills/dex-developer/upstream") ||
		!strings.Contains(gitmodules, "url = https://github.com/superdurable/skill-dex-developer.git") {
		t.Fatal("Dex skill submodule path or public HTTPS URL is not allowlisted")
	}
	command := exec.Command("git", "ls-files", "--stage", ".agents/skills/dex-developer/upstream")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("read Dex skill submodule pin: %v", err)
	}
	fields := strings.Fields(string(output))
	if len(fields) < 2 || fields[0] != "160000" || len(fields[1]) != 40 {
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
