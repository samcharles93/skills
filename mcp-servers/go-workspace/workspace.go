package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type ModuleSummary struct {
	Path         string `json:"path" jsonschema:"Go module path"`
	Dir          string `json:"dir" jsonschema:"absolute directory of the module"`
	GoMod        string `json:"go_mod" jsonschema:"absolute path to the module go.mod file"`
	PackageCount int    `json:"package_count" jsonschema:"number of Go packages discovered in this module"`
}

type PackageSummary struct {
	ImportPath    string `json:"import_path" jsonschema:"full import path for the package"`
	Name          string `json:"name" jsonschema:"Go package name"`
	Dir           string `json:"dir" jsonschema:"absolute directory containing the package"`
	ModulePath    string `json:"module_path,omitempty" jsonschema:"owning Go module path"`
	GoFileCount   int    `json:"go_file_count" jsonschema:"number of non-test Go source files"`
	TestFileCount int    `json:"test_file_count" jsonschema:"number of test Go source files"`
	HasTests      bool   `json:"has_tests" jsonschema:"whether the package contains test files"`
}

type PackageDetails struct {
	ImportPath    string   `json:"import_path" jsonschema:"full import path for the package"`
	Name          string   `json:"name" jsonschema:"Go package name"`
	Dir           string   `json:"dir" jsonschema:"absolute directory containing the package"`
	ModulePath    string   `json:"module_path,omitempty" jsonschema:"owning Go module path"`
	GoFiles       []string `json:"go_files,omitempty" jsonschema:"non-test Go source files in the package directory"`
	TestGoFiles   []string `json:"test_go_files,omitempty" jsonschema:"internal test files in the package directory"`
	XTestGoFiles  []string `json:"x_test_go_files,omitempty" jsonschema:"external test files in the package directory"`
	Imports       []string `json:"imports,omitempty" jsonschema:"imports used by the package"`
	TestImports   []string `json:"test_imports,omitempty" jsonschema:"imports used by internal tests"`
	XTestImports  []string `json:"x_test_imports,omitempty" jsonschema:"imports used by external tests"`
	GoFileCount   int      `json:"go_file_count" jsonschema:"number of non-test Go source files"`
	TestFileCount int      `json:"test_file_count" jsonschema:"number of test Go source files"`
	HasTests      bool     `json:"has_tests" jsonschema:"whether the package contains test files"`
}

type WorkspaceInspection struct {
	ResolvedRoot  string
	WorkspaceKind string
	WorkspaceRoot string
	WorkspaceFile string
	GoVersion     string
	Modules       []ModuleSummary
	Packages      []PackageInfo
}

type PackageInfo struct {
	ImportPath   string
	Name         string
	Dir          string
	ModulePath   string
	GoFiles      []string
	TestGoFiles  []string
	XTestGoFiles []string
	Imports      []string
	TestImports  []string
	XTestImports []string
}

type goEnvInfo struct {
	GOMOD     string `json:"GOMOD"`
	GOWORK    string `json:"GOWORK"`
	GOVERSION string `json:"GOVERSION"`
}

type goWorkInfo struct {
	Use []struct {
		DiskPath string `json:"DiskPath"`
	} `json:"Use"`
}

type goModuleInfo struct {
	Path  string `json:"Path"`
	Dir   string `json:"Dir"`
	GoMod string `json:"GoMod"`
}

type goPackageInfo struct {
	ImportPath   string        `json:"ImportPath"`
	Name         string        `json:"Name"`
	Dir          string        `json:"Dir"`
	GoFiles      []string      `json:"GoFiles"`
	TestGoFiles  []string      `json:"TestGoFiles"`
	XTestGoFiles []string      `json:"XTestGoFiles"`
	Imports      []string      `json:"Imports"`
	TestImports  []string      `json:"TestImports"`
	XTestImports []string      `json:"XTestImports"`
	Module       *goModuleInfo `json:"Module"`
}

func inspectWorkspace(ctx context.Context, requestedRoot string) (WorkspaceInspection, error) {
	resolvedRoot, err := resolveRoot(requestedRoot)
	if err != nil {
		return WorkspaceInspection{}, err
	}

	envInfo, err := readGoEnv(ctx, resolvedRoot)
	if err != nil {
		return WorkspaceInspection{}, err
	}

	modules, workspaceKind, workspaceRoot, workspaceFile, err := discoverModules(ctx, resolvedRoot, envInfo)
	if err != nil {
		return WorkspaceInspection{}, err
	}

	packages := make([]PackageInfo, 0)
	for _, module := range modules {
		modulePackages, err := listModulePackages(ctx, module.Dir)
		if err != nil {
			return WorkspaceInspection{}, fmt.Errorf("list packages for %s: %w", module.Path, err)
		}
		packages = append(packages, modulePackages...)
	}

	packages = dedupePackages(packages)
	moduleCounts := make(map[string]int, len(modules))
	for _, pkg := range packages {
		moduleCounts[pkg.ModulePath]++
	}
	for i := range modules {
		modules[i].PackageCount = moduleCounts[modules[i].Path]
	}

	sort.Slice(modules, func(i, j int) bool {
		if modules[i].Path == modules[j].Path {
			return modules[i].Dir < modules[j].Dir
		}
		return modules[i].Path < modules[j].Path
	})

	return WorkspaceInspection{
		ResolvedRoot:  resolvedRoot,
		WorkspaceKind: workspaceKind,
		WorkspaceRoot: workspaceRoot,
		WorkspaceFile: workspaceFile,
		GoVersion:     envInfo.GOVERSION,
		Modules:       modules,
		Packages:      packages,
	}, nil
}

func resolveRoot(requestedRoot string) (string, error) {
	root := strings.TrimSpace(requestedRoot)
	if root == "" {
		root = strings.TrimSpace(os.Getenv("GO_WORKSPACE_ROOT"))
	}
	if root == "" {
		root = "."
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}

	info, err := os.Stat(absoluteRoot)
	if err != nil {
		return "", fmt.Errorf("inspect root %q: %w", absoluteRoot, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("root %q is not a directory", absoluteRoot)
	}

	return absoluteRoot, nil
}

func readGoEnv(ctx context.Context, dir string) (goEnvInfo, error) {
	output, err := runCommand(ctx, dir, "go", "env", "-json", "GOMOD", "GOWORK", "GOVERSION")
	if err != nil {
		return goEnvInfo{}, fmt.Errorf("read go env: %w", err)
	}

	var info goEnvInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return goEnvInfo{}, fmt.Errorf("decode go env output: %w", err)
	}
	return info, nil
}

func discoverModules(ctx context.Context, resolvedRoot string, envInfo goEnvInfo) ([]ModuleSummary, string, string, string, error) {
	if isActiveGoPath(envInfo.GOWORK) {
		workFile := filepath.Clean(envInfo.GOWORK)
		workDir := filepath.Dir(workFile)
		modules, err := readWorkspaceModules(ctx, workDir)
		if err != nil {
			return nil, "", "", "", err
		}
		return modules, "go.work", workDir, workFile, nil
	}

	if isActiveGoPath(envInfo.GOMOD) {
		module, err := readMainModule(ctx, filepath.Dir(envInfo.GOMOD))
		if err != nil {
			return nil, "", "", "", err
		}
		return []ModuleSummary{module}, "module", module.Dir, module.GoMod, nil
	}

	return nil, "", "", "", fmt.Errorf("%s is not inside a Go module or workspace", resolvedRoot)
}

func isActiveGoPath(path string) bool {
	path = strings.TrimSpace(path)
	return path != "" && path != os.DevNull && path != "off"
}

func readWorkspaceModules(ctx context.Context, workDir string) ([]ModuleSummary, error) {
	output, err := runCommand(ctx, workDir, "go", "work", "edit", "-json")
	if err != nil {
		return nil, fmt.Errorf("read go.work metadata: %w", err)
	}

	var workInfo goWorkInfo
	if err := json.Unmarshal(output, &workInfo); err != nil {
		return nil, fmt.Errorf("decode go.work metadata: %w", err)
	}

	modules := make([]ModuleSummary, 0, len(workInfo.Use))
	seen := make(map[string]struct{}, len(workInfo.Use))
	for _, use := range workInfo.Use {
		moduleDir := filepath.Clean(filepath.Join(workDir, filepath.FromSlash(use.DiskPath)))
		if _, ok := seen[moduleDir]; ok {
			continue
		}
		seen[moduleDir] = struct{}{}

		module, err := readMainModule(ctx, moduleDir)
		if err != nil {
			return nil, fmt.Errorf("read module %s from go.work: %w", moduleDir, err)
		}
		modules = append(modules, module)
	}

	return modules, nil
}

func readMainModule(ctx context.Context, dir string) (ModuleSummary, error) {
	output, err := runCommand(ctx, dir, "go", "list", "-m", "-json")
	if err != nil {
		return ModuleSummary{}, fmt.Errorf("read main module in %s: %w", dir, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	expectedDir := filepath.Clean(dir)
	var fallback *goModuleInfo
	for {
		var module goModuleInfo
		if err := decoder.Decode(&module); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return ModuleSummary{}, fmt.Errorf("decode main module metadata for %s: %w", dir, err)
		}
		if fallback == nil {
			moduleCopy := module
			fallback = &moduleCopy
		}
		if filepath.Clean(module.Dir) != expectedDir {
			continue
		}
		return ModuleSummary{
			Path:  module.Path,
			Dir:   module.Dir,
			GoMod: module.GoMod,
		}, nil
	}

	if fallback == nil {
		return ModuleSummary{}, fmt.Errorf("no module metadata found for %s", dir)
	}

	return ModuleSummary{
		Path:  fallback.Path,
		Dir:   fallback.Dir,
		GoMod: fallback.GoMod,
	}, nil
}

func listModulePackages(ctx context.Context, dir string) ([]PackageInfo, error) {
	output, err := runCommand(ctx, dir, "go", "list", "-json", "./...")
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	packages := make([]PackageInfo, 0)
	for {
		var pkg goPackageInfo
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode go list output for %s: %w", dir, err)
		}
		packages = append(packages, PackageInfo{
			ImportPath:   pkg.ImportPath,
			Name:         pkg.Name,
			Dir:          pkg.Dir,
			ModulePath:   modulePath(pkg.Module),
			GoFiles:      sortedStrings(pkg.GoFiles),
			TestGoFiles:  sortedStrings(pkg.TestGoFiles),
			XTestGoFiles: sortedStrings(pkg.XTestGoFiles),
			Imports:      sortedStrings(pkg.Imports),
			TestImports:  sortedStrings(pkg.TestImports),
			XTestImports: sortedStrings(pkg.XTestImports),
		})
	}

	sort.Slice(packages, func(i, j int) bool {
		if packages[i].ImportPath == packages[j].ImportPath {
			return packages[i].Dir < packages[j].Dir
		}
		return packages[i].ImportPath < packages[j].ImportPath
	})

	return packages, nil
}

func modulePath(module *goModuleInfo) string {
	if module == nil {
		return ""
	}
	return module.Path
}

func dedupePackages(packages []PackageInfo) []PackageInfo {
	seen := make(map[string]struct{}, len(packages))
	deduped := make([]PackageInfo, 0, len(packages))
	for _, pkg := range packages {
		key := pkg.ImportPath + "::" + pkg.Dir
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, pkg)
	}
	return deduped
}

func sortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	return sorted
}

func (pkg PackageInfo) Summary() PackageSummary {
	return PackageSummary{
		ImportPath:    pkg.ImportPath,
		Name:          pkg.Name,
		Dir:           pkg.Dir,
		ModulePath:    pkg.ModulePath,
		GoFileCount:   len(pkg.GoFiles),
		TestFileCount: pkg.TestFileCount(),
		HasTests:      pkg.HasTests(),
	}
}

func (pkg PackageInfo) Details() PackageDetails {
	return PackageDetails{
		ImportPath:    pkg.ImportPath,
		Name:          pkg.Name,
		Dir:           pkg.Dir,
		ModulePath:    pkg.ModulePath,
		GoFiles:       append([]string(nil), pkg.GoFiles...),
		TestGoFiles:   append([]string(nil), pkg.TestGoFiles...),
		XTestGoFiles:  append([]string(nil), pkg.XTestGoFiles...),
		Imports:       append([]string(nil), pkg.Imports...),
		TestImports:   append([]string(nil), pkg.TestImports...),
		XTestImports:  append([]string(nil), pkg.XTestImports...),
		GoFileCount:   len(pkg.GoFiles),
		TestFileCount: pkg.TestFileCount(),
		HasTests:      pkg.HasTests(),
	}
}

func (pkg PackageInfo) TestFileCount() int {
	return len(pkg.TestGoFiles) + len(pkg.XTestGoFiles)
}

func (pkg PackageInfo) HasTests() bool {
	return pkg.TestFileCount() > 0
}

func runCommand(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()

	// Return stdout even if there was an error, as it often contains the test failure details.
	combinedOutput := append(stdout.Bytes(), stderr.Bytes()...)

	if err != nil {
		return combinedOutput, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}

	return combinedOutput, nil
}

func isWithinRoot(root string, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || (!strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != "..")
}
