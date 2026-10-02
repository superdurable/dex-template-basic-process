package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type manifest struct {
	SchemaVersion                       int               `json:"schemaVersion"`
	BuildProfile                        string            `json:"buildProfile"`
	TemplateVersion                     string            `json:"templateVersion"`
	MinimumSandboxImageContractRevision int               `json:"minimumSandboxImageContractRevision"`
	OpenAPISpec                         string            `json:"openapiSpec"`
	ApplicationManifest                 string            `json:"applicationManifest"`
	AgentInstructions                   string            `json:"agentInstructions"`
	Commands                            map[string]string `json:"commands"`
}

func checkTemplateContract() {
	root := "."
	manifestBytes, err := os.ReadFile(filepath.Join(root, ".superverse", "template.json"))
	if err != nil {
		failf("read manifest: %v", err)
	}
	var contract manifest
	if err := json.Unmarshal(manifestBytes, &contract); err != nil {
		failf("decode manifest: %v", err)
	}
	if contract.SchemaVersion != 1 || contract.BuildProfile != "go-react-v1" || contract.TemplateVersion != "1.9.2" || contract.MinimumSandboxImageContractRevision != 3 {
		failf("unexpected template identity: %+v", contract)
	}
	if baseline := strings.TrimSpace(readFile(filepath.Join(root, "DEX_SERVER_BASELINE"))); baseline != "server/v0.14.1" {
		failf("unexpected Dex Server baseline: %q", baseline)
	}
	if baseline := strings.TrimSpace(readFile(filepath.Join(root, "DEX_CLI_BASELINE"))); baseline != "cli-v0.14.1" {
		failf("unexpected Dex CLI baseline: %q", baseline)
	}
	goModule := readFile(filepath.Join(root, "go.mod"))
	if !strings.Contains(goModule, "github.com/superdurable/dex/sdk-go v0.13.1") {
		failf("template must pin Dex Go SDK v0.13.1")
	}
	if contract.ApplicationManifest != "dex-app.yaml" {
		failf("unexpected application manifest: %q", contract.ApplicationManifest)
	}
	for _, path := range []string{
		contract.OpenAPISpec,
		contract.ApplicationManifest,
		contract.AgentInstructions,
	} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			failf("manifest path %q: %v", path, err)
		}
	}
	makefile := readFile(filepath.Join(root, "Makefile"))
	agents := readFile(filepath.Join(root, contract.AgentInstructions))
	readme := readFile(filepath.Join(root, "README.md"))
	for name, contents := range map[string]string{"AGENTS.md": agents, "README.md": readme} {
		if strings.Contains(contents, "/opt/superverse/dex-skills") {
			failf("%s must discover Dex Skills through the coding-agent host", name)
		}
	}
	if !strings.Contains(agents, "`dex-app-builder` skill") {
		failf("AGENTS.md must require the installed dex-app-builder skill")
	}
	for name, contents := range map[string]string{"AGENTS.md": agents, "README.md": readme} {
		for _, required := range []string{"`runtime`", "`normaliz`", "internal/worker", "make check-contracts", "make check"} {
			if !strings.Contains(contents, required) {
				failf("%s does not document the concrete naming policy: missing %q", name, required)
			}
		}
	}
	for name, contents := range map[string]string{"AGENTS.md": agents, "README.md": readme} {
		contents = strings.Join(strings.Fields(contents), " ")
		for _, required := range []string{
			"reproducibility baselines",
			"newer stable patch release",
			"without separate user authorization",
		} {
			if !strings.Contains(contents, required) {
				failf("%s does not document the Dex patch compatibility policy: missing %q", name, required)
			}
		}
	}
	expectedCommands := map[string]string{
		"bootstrap":        "make bootstrap",
		"generate":         "make generate",
		"checkFdgV2":       "make check-fdg-v2",
		"releaseArtifacts": "make superverse-release-artifacts",
		"checkContracts":   "make check-contracts",
		"checkStatic":      "make check-static",
		"build":            "make build",
		"dev":              "make dev",
		"check":            "make check",
	}
	if len(contract.Commands) != len(expectedCommands) {
		failf("unexpected template commands: %+v", contract.Commands)
	}
	for name, command := range expectedCommands {
		if contract.Commands[name] != command {
			failf("template command %q = %q, want %q", name, contract.Commands[name], command)
		}
		target := strings.TrimPrefix(command, "make ")
		if !strings.Contains(makefile, "\n"+target+":") && !strings.HasPrefix(makefile, target+":") {
			failf("Makefile does not define %q", target)
		}
		if !strings.Contains(agents, command) {
			failf("AGENTS.md does not mention %q", command)
		}
	}
	for _, removedPath := range []string{
		".gitmodules",
		".agents",
		"cmd/mock-server",
		"internal/mockserver",
		"docs/local-mock.md",
		"scripts/run-mock-e2e.sh",
		"scripts/with-mock.sh",
		"web/src/MockControls.tsx",
	} {
		if _, err := os.Stat(filepath.Join(root, removedPath)); !os.IsNotExist(err) {
			failf("removed template path %q still exists", removedPath)
		}
	}
	gitignore := readFile(filepath.Join(root, ".gitignore"))
	for _, generatedPath := range []string{"/internal/api/generated/", "/web/src/api/generated/"} {
		if !strings.Contains(gitignore, generatedPath) {
			failf(".gitignore must exclude %q", generatedPath)
		}
	}
	gitCheck := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	gitCheck.Dir = root
	if err := gitCheck.Run(); err == nil {
		command := exec.Command("git", "ls-files", "internal/api/generated", "web/src/api/generated")
		command.Dir = root
		trackedGenerated, err := command.Output()
		if err != nil {
			failf("list tracked generated files: %v", err)
		}
		if strings.TrimSpace(string(trackedGenerated)) != "" {
			failf("generated OpenAPI files must not be tracked:\n%s", trackedGenerated)
		}
	}

	dependencyUpdater := readFile(filepath.Join(root, "scripts", "update-dex-dependencies.py"))
	for _, required := range []string{
		"github.com/superdurable/dex/sdk-go",
		"DEX_SERVER_BASELINE",
		"DEX_CLI_BASELINE",
		"templateVersion",
		"GITHUB_OUTPUT",
	} {
		if !strings.Contains(dependencyUpdater, required) {
			failf("Dex dependency updater does not mention %q", required)
		}
	}
}

// Template publishing automation belongs to the template repository, not exported apps.
func checkTemplateRepositoryAutomation() {
	root := "."
	updateWorkflow := readFile(filepath.Join(root, ".github", "workflows", "update-dex-dependencies.yml"))
	for _, required := range []string{
		"schedule:",
		"workflow_dispatch:",
		"pull-requests: write",
		"automation/update-dex-dependencies",
		"gh pr create",
		"gh workflow run ci.yml",
	} {
		if !strings.Contains(updateWorkflow, required) {
			failf("Dex dependency update workflow does not mention %q", required)
		}
	}
	ciWorkflow := readFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	for _, required := range []string{
		"workflow_dispatch:",
		"scripts/check-template-version.py",
		"gh release create",
		"contents: write",
	} {
		if !strings.Contains(ciWorkflow, required) {
			failf("Template CI does not mention %q", required)
		}
	}
}

func checkGoNamesDescribeConcreteResponsibilities() {
	root := "."
	for _, directory := range []string{"cmd", "internal", "tools"} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() && (entry.Name() == "generated" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			if entry.IsDir() {
				checkConcreteName(path, entry.Name())
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}
			positions := token.NewFileSet()
			source, err := parser.ParseFile(positions, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			if ast.IsGenerated(source) {
				return nil
			}
			checkConcreteName(path, entry.Name())
			ast.Inspect(source, func(node ast.Node) bool {
				check := func(identifier *ast.Ident) {
					checkConcreteName(positions.Position(identifier.Pos()).String(), identifier.Name)
				}
				// Inspect declarations, not references to third-party identifiers.
				switch declaration := node.(type) {
				case *ast.File:
					check(declaration.Name)
				case *ast.ImportSpec:
					if declaration.Name != nil {
						check(declaration.Name)
					}
				case *ast.TypeSpec:
					check(declaration.Name)
				case *ast.FuncDecl:
					check(declaration.Name)
				case *ast.Field:
					for _, name := range declaration.Names {
						check(name)
					}
				case *ast.ValueSpec:
					for _, name := range declaration.Names {
						check(name)
					}
				case *ast.AssignStmt:
					if declaration.Tok == token.DEFINE {
						for _, expression := range declaration.Lhs {
							if name, ok := expression.(*ast.Ident); ok {
								check(name)
							}
						}
					}
				case *ast.RangeStmt:
					if declaration.Tok == token.DEFINE {
						for _, expression := range []ast.Expr{declaration.Key, declaration.Value} {
							if name, ok := expression.(*ast.Ident); ok {
								check(name)
							}
						}
					}
				case *ast.LabeledStmt:
					check(declaration.Label)
				}
				return true
			})
			return nil
		})
		if err != nil {
			failf("inspect Go names in %s: %v", directory, err)
		}
	}
}

func checkConcreteName(location, name string) {
	for _, stem := range []string{"runtime", "normaliz"} {
		if strings.Contains(strings.ToLower(name), stem) {
			failf("%s: name %q contains prohibited stem %q; name the concrete responsibility", location, name, stem)
		}
	}
}

func readFile(path string) string {
	contents, err := os.ReadFile(path)
	if err != nil {
		failf("read %s: %v", path, err)
	}
	return string(contents)
}

// Static repository validation is separate from API integration tests.
func main() {
	repositoryAutomation := flag.Bool("repository-automation", false, "also validate template-owned release/update workflows")
	flag.Parse()
	if flag.NArg() != 0 {
		failf("unexpected positional arguments")
	}
	checkTemplateContract()
	checkGoNamesDescribeConcreteResponsibilities()
	checkTestPolicy()
	if *repositoryAutomation {
		checkTemplateRepositoryAutomation()
	}
	fmt.Println("Template contract, Go names, and integration-only test policy are valid.")
}

func failf(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", arguments...)
	os.Exit(1)
}

func checkTestPolicy() {
	var frontend struct {
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(readFile("web/package.json")), &frontend); err != nil {
		failf("decode frontend dependencies: %v", err)
	}
	for _, dependency := range []string{"vitest", "jsdom", "@testing-library/react", "@testing-library/jest-dom"} {
		if _, exists := frontend.DevDependencies[dependency]; exists {
			failf("unit-test dependency %q is prohibited", dependency)
		}
	}
	if strings.Contains(readFile("Makefile"), "test-unit") {
		failf("unit-test Make targets are prohibited")
	}
	frontendTestName := regexp.MustCompile(`\.(test|spec)\.[cm]?[jt]sx?$`)
	for _, directory := range []string{"cmd", "internal", "tools", "scripts", "web"} {
		err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				switch entry.Name() {
				case "generated", "vendor", "node_modules", "dist", "test-results", "playwright-report", "__pycache__":
					return filepath.SkipDir
				}
				return nil
			}
			name := entry.Name()
			isGoTest := strings.HasSuffix(name, "_test.go")
			isFrontendTest := frontendTestName.MatchString(name)
			if strings.HasSuffix(name, ".py") && (strings.HasPrefix(name, "test_") || strings.HasSuffix(name, "_test.py")) {
				failf("%s: Python unit-test suites are prohibited; use real API integration coverage", path)
			}
			if !isGoTest && !isFrontendTest {
				return nil
			}
			contents := readFile(path)
			if isGoTest && !strings.HasPrefix(contents, "//go:build integration\n") {
				failf("%s: Go tests must use the integration build tag and call real dependency APIs", path)
			}
			if isFrontendTest && (!strings.HasPrefix(filepath.ToSlash(path), "web/e2e/") || !strings.HasSuffix(name, ".spec.ts")) {
				failf("%s: frontend tests must be real Playwright E2E tests in web/e2e", path)
			}
			for _, prohibited := range []string{
				"vitest", "@testing-library/", "gomock", "testify/mock", "go-sqlmock", "gock", "httpmock", "msw", "vi.mock", "jest.mock", ".route(", ".routeFromHAR(",
			} {
				if strings.Contains(contents, prohibited) {
					failf("%s: mocked integration tests are prohibited (%s)", path, prohibited)
				}
			}
			return nil
		})
		if err != nil {
			failf("inspect test policy in %s: %v", directory, err)
		}
	}
}
