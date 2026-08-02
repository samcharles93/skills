package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestWorkspaceSummaryToolEndToEnd(t *testing.T) {
	ctx := context.Background()
	root := createWorkspaceFixture(t)

	clientSession := newClientSession(t, ctx)
	defer clientSession.Close()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "workspace_summary",
		Arguments: map[string]any{"root": root},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned isError=true: %s", firstText(t, res))
	}

	var output WorkspaceSummaryOutput
	decodeStructured(t, res, &output)
	if output.WorkspaceKind != "go.work" {
		t.Fatalf("unexpected workspace kind: %#v", output)
	}
	if output.ModuleCount != 2 {
		t.Fatalf("unexpected module count: %#v", output)
	}
	if output.PackageCount != 2 {
		t.Fatalf("unexpected package count: %#v", output)
	}
	if !strings.Contains(firstText(t, res), "Detected 2 module(s)") {
		t.Fatalf("unexpected summary text: %q", firstText(t, res))
	}
}

func TestPackageToolsEndToEnd(t *testing.T) {
	ctx := context.Background()
	root := createSingleModuleFixture(t)

	clientSession := newClientSession(t, ctx)
	defer clientSession.Close()

	listRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_packages",
		Arguments: map[string]any{
			"root":  root,
			"limit": 1,
			"query": "example.com/sample",
		},
	})
	if err != nil {
		t.Fatalf("call list_packages: %v", err)
	}
	if listRes.IsError {
		t.Fatalf("list_packages returned isError=true: %s", firstText(t, listRes))
	}

	var listOutput ListPackagesOutput
	decodeStructured(t, listRes, &listOutput)
	if !listOutput.Truncated {
		t.Fatalf("expected truncated list output: %#v", listOutput)
	}
	if listOutput.MatchedCount < 2 {
		t.Fatalf("expected at least two matching packages: %#v", listOutput)
	}
	if listOutput.ReturnedCount != 1 {
		t.Fatalf("expected one returned package: %#v", listOutput)
	}

	describeRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "describe_package",
		Arguments: map[string]any{
			"root":    root,
			"package": "./internal/thing",
		},
	})
	if err != nil {
		t.Fatalf("call describe_package: %v", err)
	}
	if describeRes.IsError {
		t.Fatalf("describe_package returned isError=true: %s", firstText(t, describeRes))
	}

	var describeOutput DescribePackageOutput
	decodeStructured(t, describeRes, &describeOutput)
	if describeOutput.Package.ImportPath != "example.com/sample/internal/thing" {
		t.Fatalf("unexpected package details: %#v", describeOutput.Package)
	}
	if !describeOutput.Package.HasTests || describeOutput.Package.TestFileCount != 1 {
		t.Fatalf("unexpected test details: %#v", describeOutput.Package)
	}
	if !strings.Contains(firstText(t, describeRes), "example.com/sample/internal/thing") {
		t.Fatalf("unexpected describe summary: %q", firstText(t, describeRes))
	}
}

func TestChangedPackagesToolEndToEnd(t *testing.T) {
	ctx := context.Background()
	root := createSingleModuleFixture(t)
	requireGit(t)
	initGitRepo(t, root)

	writeFile(t, filepath.Join(root, "internal", "thing", "thing.go"), "package thing\n\nfunc Value() string { return \"updated\" }\n")

	clientSession := newClientSession(t, ctx)
	defer clientSession.Close()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "changed_packages",
		Arguments: map[string]any{"root": root},
	})
	if err != nil {
		t.Fatalf("call changed_packages: %v", err)
	}
	if res.IsError {
		t.Fatalf("changed_packages returned isError=true: %s", firstText(t, res))
	}

	var output ChangedPackagesOutput
	decodeStructured(t, res, &output)
	if output.ChangedFileCount != 1 {
		t.Fatalf("unexpected changed file count: %#v", output)
	}
	if output.PackageCount != 1 {
		t.Fatalf("unexpected package count: %#v", output)
	}
	if len(output.Packages) != 1 || output.Packages[0].ImportPath != "example.com/sample/internal/thing" {
		t.Fatalf("unexpected changed package mapping: %#v", output)
	}
	if output.ChangedFiles[0] != "internal/thing/thing.go" {
		t.Fatalf("unexpected changed files: %#v", output.ChangedFiles)
	}
}

func TestRunTestsToolAcceptsCapitalizedPackageKey(t *testing.T) {
	ctx := context.Background()
	root := createSingleModuleFixture(t)

	clientSession := newClientSession(t, ctx)
	defer clientSession.Close()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_tests_for_package",
		Arguments: map[string]any{
			"root":    root,
			"Package": "./internal/thing",
		},
	})
	if err != nil {
		t.Fatalf("call run_tests_for_package: %v", err)
	}
	if res.IsError {
		t.Fatalf("run_tests_for_package returned isError=true: %s", firstText(t, res))
	}

	var output RunTestsOutput
	decodeStructured(t, res, &output)
	if !output.Success {
		t.Fatalf("expected package tests to pass: %#v", output)
	}
	if output.Package != "example.com/sample/internal/thing" {
		t.Fatalf("unexpected tested package: %#v", output)
	}
}

func TestWorkspaceSummaryToolFailure(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	clientSession := newClientSession(t, ctx)
	defer clientSession.Close()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "workspace_summary",
		Arguments: map[string]any{"root": root},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected tool error result, got success: %#v", res)
	}
	if !strings.Contains(firstText(t, res), "not inside a Go module or workspace") {
		t.Fatalf("unexpected tool error text: %q", firstText(t, res))
	}
}

func newClientSession(t *testing.T, ctx context.Context) *mcp.ClientSession {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := newServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connect server: %v", err)
	}
	t.Cleanup(func() {
		serverSession.Close()
	})

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "v1.0.0",
	}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	return clientSession
}

func decodeStructured(t *testing.T, result *mcp.CallToolResult, target any) {
	t.Helper()

	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
}

func firstText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if len(result.Content) == 0 {
		t.Fatal("tool returned no content")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type %T", result.Content[0])
	}
	return text.Text
}

func createSingleModuleFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/sample\n\ngo 1.24.0\n")
	writeFile(t, filepath.Join(root, "sample.go"), "package sample\n\nfunc Answer() int { return 42 }\n")
	writeFile(t, filepath.Join(root, "internal", "thing", "thing.go"), "package thing\n\nfunc Value() string { return \"ok\" }\n")
	writeFile(t, filepath.Join(root, "internal", "thing", "thing_test.go"), "package thing\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) {\n\tif Value() != \"ok\" {\n\t\tt.Fatal(\"want ok\")\n\t}\n}\n")
	return root
}

func createWorkspaceFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.work"), "go 1.24.0\n\nuse (\n\t./app\n\t./lib\n)\n")
	writeFile(t, filepath.Join(root, "app", "go.mod"), "module example.com/app\n\ngo 1.24.0\n")
	writeFile(t, filepath.Join(root, "app", "app.go"), "package app\n\nfunc Name() string { return \"app\" }\n")
	writeFile(t, filepath.Join(root, "lib", "go.mod"), "module example.com/lib\n\ngo 1.24.0\n")
	writeFile(t, filepath.Join(root, "lib", "lib.go"), "package lib\n\nfunc Name() string { return \"lib\" }\n")
	return root
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}
