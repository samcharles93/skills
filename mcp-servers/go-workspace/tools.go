package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type WorkspaceSummaryInput struct {
	Root string `json:"root,omitempty" jsonschema:"absolute or relative directory to inspect; defaults to GO_WORKSPACE_ROOT or the current working directory"`
}

type WorkspaceSummaryOutput struct {
	ResolvedRoot  string          `json:"resolved_root" jsonschema:"absolute directory that was inspected"`
	WorkspaceKind string          `json:"workspace_kind" jsonschema:"whether the root resolved to a go.work workspace or a single module"`
	WorkspaceRoot string          `json:"workspace_root" jsonschema:"absolute directory containing the active go.work or go.mod file"`
	WorkspaceFile string          `json:"workspace_file" jsonschema:"absolute path to the active go.work or go.mod file"`
	GoVersion     string          `json:"go_version" jsonschema:"Go toolchain version reported by go env"`
	ModuleCount   int             `json:"module_count" jsonschema:"number of active modules in the workspace"`
	PackageCount  int             `json:"package_count" jsonschema:"number of discovered Go packages"`
	Modules       []ModuleSummary `json:"modules" jsonschema:"active modules in the workspace"`
}

type ListPackagesInput struct {
	Root  string `json:"root,omitempty" jsonschema:"absolute or relative directory to inspect; defaults to GO_WORKSPACE_ROOT or the current working directory"`
	Query string `json:"query,omitempty" jsonschema:"optional case-insensitive substring filter for import path, package name, or directory"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum number of packages to return; defaults to 100 and is capped at 200"`
}

type ListPackagesOutput struct {
	ResolvedRoot  string           `json:"resolved_root" jsonschema:"absolute directory that was inspected"`
	Query         string           `json:"query,omitempty" jsonschema:"case-insensitive package filter that was applied"`
	MatchedCount  int              `json:"matched_count" jsonschema:"number of packages that matched before limit was applied"`
	ReturnedCount int              `json:"returned_count" jsonschema:"number of packages returned in this response"`
	Truncated     bool             `json:"truncated" jsonschema:"whether results were limited by the requested or default limit"`
	Packages      []PackageSummary `json:"packages" jsonschema:"package summaries matching the query"`
}

type DescribePackageInput struct {
	Root    string `json:"root,omitempty" jsonschema:"absolute or relative directory to inspect; defaults to GO_WORKSPACE_ROOT or the current working directory"`
	Package string `json:"package" jsonschema:"full import path or directory for the package to describe"`
}

type DescribePackageOutput struct {
	ResolvedRoot string         `json:"resolved_root" jsonschema:"absolute directory that was inspected"`
	Package      PackageDetails `json:"package" jsonschema:"package details for the resolved package"`
}

type ChangedPackagesInput struct {
	Root             string `json:"root,omitempty" jsonschema:"absolute or relative directory to inspect; defaults to GO_WORKSPACE_ROOT or the current working directory"`
	BaseRef          string `json:"base_ref,omitempty" jsonschema:"optional git revision to diff against using base_ref...HEAD; when omitted, the tool inspects staged and unstaged changes"`
	IncludeUntracked *bool  `json:"include_untracked,omitempty" jsonschema:"whether to include untracked files when base_ref is omitted; defaults to true"`
}

type ChangedPackagesOutput struct {
	ResolvedRoot     string           `json:"resolved_root" jsonschema:"absolute directory that was inspected"`
	BaseRef          string           `json:"base_ref,omitempty" jsonschema:"git revision used for the diff when provided"`
	ChangedFileCount int              `json:"changed_file_count" jsonschema:"number of changed files under the inspected root"`
	ChangedFiles     []string         `json:"changed_files" jsonschema:"changed files relative to the inspected root"`
	PackageCount     int              `json:"package_count" jsonschema:"number of Go packages affected by the changed files"`
	Packages         []PackageSummary `json:"packages" jsonschema:"Go packages affected by the changed files"`
	UnmatchedFiles   []string         `json:"unmatched_files,omitempty" jsonschema:"changed files that did not map to a discovered Go package"`
}

func WorkspaceSummaryTool(ctx context.Context, req *mcp.CallToolRequest, input WorkspaceSummaryInput) (*mcp.CallToolResult, WorkspaceSummaryOutput, error) {
	_ = req

	inspection, err := inspectWorkspace(ctx, input.Root)
	if err != nil {
		return nil, WorkspaceSummaryOutput{}, err
	}

	output := WorkspaceSummaryOutput{
		ResolvedRoot:  inspection.ResolvedRoot,
		WorkspaceKind: inspection.WorkspaceKind,
		WorkspaceRoot: inspection.WorkspaceRoot,
		WorkspaceFile: inspection.WorkspaceFile,
		GoVersion:     inspection.GoVersion,
		ModuleCount:   len(inspection.Modules),
		PackageCount:  len(inspection.Packages),
		Modules:       inspection.Modules,
	}

	text := fmt.Sprintf(
		"Detected %d module(s) and %d package(s) from %s.",
		output.ModuleCount,
		output.PackageCount,
		output.ResolvedRoot,
	)
	return textResult(text), output, nil
}

func ListPackagesTool(ctx context.Context, req *mcp.CallToolRequest, input ListPackagesInput) (*mcp.CallToolResult, ListPackagesOutput, error) {
	_ = req

	inspection, err := inspectWorkspace(ctx, input.Root)
	if err != nil {
		return nil, ListPackagesOutput{}, err
	}

	selected, matchedCount, truncated := selectPackages(inspection.Packages, input.Query, input.Limit)
	summaries := make([]PackageSummary, 0, len(selected))
	for _, pkg := range selected {
		summaries = append(summaries, pkg.Summary())
	}

	output := ListPackagesOutput{
		ResolvedRoot:  inspection.ResolvedRoot,
		Query:         strings.TrimSpace(input.Query),
		MatchedCount:  matchedCount,
		ReturnedCount: len(summaries),
		Truncated:     truncated,
		Packages:      summaries,
	}

	text := fmt.Sprintf(
		"Listed %d of %d matching package(s) under %s.",
		output.ReturnedCount,
		output.MatchedCount,
		output.ResolvedRoot,
	)
	return textResult(text), output, nil
}

func DescribePackageTool(ctx context.Context, req *mcp.CallToolRequest, input DescribePackageInput) (*mcp.CallToolResult, DescribePackageOutput, error) {
	_ = req

	inspection, err := inspectWorkspace(ctx, input.Root)
	if err != nil {
		return nil, DescribePackageOutput{}, err
	}

	pkg, err := findPackage(inspection.Packages, inspection.ResolvedRoot, input.Package)
	if err != nil {
		return nil, DescribePackageOutput{}, err
	}

	output := DescribePackageOutput{
		ResolvedRoot: inspection.ResolvedRoot,
		Package:      pkg.Details(),
	}

	text := fmt.Sprintf(
		"Described package %s with %d Go file(s) and %d test file(s).",
		output.Package.ImportPath,
		output.Package.GoFileCount,
		output.Package.TestFileCount,
	)
	return textResult(text), output, nil
}

func ChangedPackagesTool(ctx context.Context, req *mcp.CallToolRequest, input ChangedPackagesInput) (*mcp.CallToolResult, ChangedPackagesOutput, error) {
	_ = req

	inspection, err := inspectWorkspace(ctx, input.Root)
	if err != nil {
		return nil, ChangedPackagesOutput{}, err
	}

	includeUntracked := true
	if input.IncludeUntracked != nil {
		includeUntracked = *input.IncludeUntracked
	}
	if strings.TrimSpace(input.BaseRef) != "" {
		includeUntracked = false
	}

	files, err := discoverChangedFiles(ctx, inspection.ResolvedRoot, input.BaseRef, includeUntracked)
	if err != nil {
		return nil, ChangedPackagesOutput{}, err
	}

	changedFiles := make([]string, 0, len(files))
	for _, file := range files {
		changedFiles = append(changedFiles, file.Relative)
	}

	packages, unmatched := changedPackages(files, inspection.Packages, inspection.ResolvedRoot)
	output := ChangedPackagesOutput{
		ResolvedRoot:     inspection.ResolvedRoot,
		BaseRef:          strings.TrimSpace(input.BaseRef),
		ChangedFileCount: len(changedFiles),
		ChangedFiles:     changedFiles,
		PackageCount:     len(packages),
		Packages:         packages,
		UnmatchedFiles:   unmatched,
	}

	text := fmt.Sprintf(
		"Mapped %d changed file(s) to %d package(s) under %s.",
		output.ChangedFileCount,
		output.PackageCount,
		output.ResolvedRoot,
	)
	return textResult(text), output, nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func selectPackages(packages []PackageInfo, query string, limit int) ([]PackageInfo, int, bool) {
	query = strings.ToLower(strings.TrimSpace(query))
	matches := make([]PackageInfo, 0, len(packages))
	for _, pkg := range packages {
		if query == "" || matchesPackageQuery(pkg, query) {
			matches = append(matches, pkg)
		}
	}

	matchedCount := len(matches)
	limit = normalizedLimit(limit)
	truncated := matchedCount > limit
	if truncated {
		matches = matches[:limit]
	}

	return matches, matchedCount, truncated
}

func matchesPackageQuery(pkg PackageInfo, query string) bool {
	return strings.Contains(strings.ToLower(pkg.ImportPath), query) ||
		strings.Contains(strings.ToLower(pkg.Name), query) ||
		strings.Contains(strings.ToLower(pkg.Dir), query)
}

func normalizedLimit(limit int) int {
	switch {
	case limit <= 0:
		return 100
	case limit > 200:
		return 200
	default:
		return limit
	}
}

func looksLikeDirectoryReference(reference string) bool {
	return filepath.IsAbs(reference) ||
		strings.HasPrefix(reference, ".") ||
		strings.Contains(reference, string(filepath.Separator)) ||
		strings.Contains(reference, "/")
}

type RunTestsInput struct {
	Root              string `json:"root,omitempty" jsonschema:"absolute or relative directory to inspect; defaults to GO_WORKSPACE_ROOT or the current working directory"`
	PackagePath       string `json:"package,omitempty" jsonschema:"full import path or directory for the package to test"`
	LegacyPackagePath string `json:"Package,omitempty" jsonschema:"deprecated alias for package; use lowercase package when possible"`
}

type RunTestsOutput struct {
	ResolvedRoot string   `json:"resolved_root" jsonschema:"absolute directory that was inspected"`
	Package      string   `json:"package" jsonschema:"resolved package import path"`
	Success      bool     `json:"success" jsonschema:"whether all tests in the package passed"`
	Output       string   `json:"output" jsonschema:"full stdout/stderr from the test command"`
	Failures     []string `json:"failures" jsonschema:"list of failing test names"`
}

func (input RunTestsInput) PackageRef() string {
	packageRef := strings.TrimSpace(input.PackagePath)
	if packageRef != "" {
		return packageRef
	}
	return strings.TrimSpace(input.LegacyPackagePath)
}

func findPackage(packages []PackageInfo, resolvedRoot string, reference string) (PackageInfo, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return PackageInfo{}, fmt.Errorf("package reference is required")
	}

	if pkg, ok := findPackageByDirectory(packages, resolvedRoot, reference); ok {
		return pkg, nil
	}

	for _, pkg := range packages {
		if pkg.ImportPath == reference {
			return pkg, nil
		}
	}

	suffixMatches := make([]PackageInfo, 0)
	for _, pkg := range packages {
		if strings.HasSuffix(pkg.ImportPath, reference) {
			suffixMatches = append(suffixMatches, pkg)
		}
	}
	if len(suffixMatches) == 1 {
		return suffixMatches[0], nil
	}
	if len(suffixMatches) > 1 {
		return PackageInfo{}, fmt.Errorf("package reference %q is ambiguous; use a full import path or directory", reference)
	}

	return PackageInfo{}, fmt.Errorf("package %q was not found under %s", reference, resolvedRoot)
}

func findPackageByDirectory(packages []PackageInfo, resolvedRoot string, reference string) (PackageInfo, bool) {
	if !looksLikeDirectoryReference(reference) {
		return PackageInfo{}, false
	}

	directory := reference
	if !filepath.IsAbs(directory) {
		directory = filepath.Join(resolvedRoot, directory)
	}
	directory = filepath.Clean(directory)

	for _, pkg := range packages {
		if filepath.Clean(pkg.Dir) == directory {
			return pkg, true
		}
	}

	return PackageInfo{}, false
}

type SearchSymbolsInput struct {
	Root   string `json:"root,omitempty" jsonschema:"absolute or relative directory to inspect; defaults to GO_WORKSPACE_ROOT or the current working directory"`
	Symbol string `json:"symbol" jsonschema:"name of the symbol (function, type, or variable) to search for"`
}

type SymbolMatch struct {
	File    string `json:"file" jsonschema:"path to the file containing the definition"`
	Line    int    `json:"line" jsonschema:"line number of the definition"`
	Context string `json:"context" jsonschema:"the line of code containing the definition"`
}

type SearchSymbolsOutput struct {
	ResolvedRoot string        `json:"resolved_root" jsonschema:"absolute directory that was inspected"`
	Matches      []SymbolMatch `json:"matches" jsonschema:"list of symbol definitions found"`
}

type InterfaceImplInput struct {
	Root   string `json:"root,omitempty" jsonschema:"absolute or relative directory to inspect; defaults to GO_WORKSPACE_ROOT or the current working directory"`
	Iface  string `json:"iface" jsonschema:"name of the interface to find implementations for"`
}

type InterfaceImplOutput struct {
	ResolvedRoot    string   `json:"resolved_root" jsonschema:"absolute directory that was inspected"`
	Implementations []string `json:"implementations" jsonschema:"list of types that implement the interface"`
}

type FindDependentsInput struct {
	Root    string `json:"root,omitempty" jsonschema:"absolute or relative directory to inspect; defaults to GO_WORKSPACE_ROOT or the current working directory"`
	Pkg     string `json:"pkg" jsonschema:"full import path or directory for the package to analyze"`
}

type FindDependentsOutput struct {
	ResolvedRoot string   `json:"resolved_root" jsonschema:"absolute directory that was inspected"`
	Dependents   []string `json:"dependents" jsonschema:"list of packages that import the target package"`
}

type FindDeprecatedInput struct {
	Root string `json:"root,omitempty" jsonschema:"absolute or relative directory to inspect; defaults to GO_WORKSPACE_ROOT or the current working directory"`
}

type FindDeprecatedOutput struct {
	ResolvedRoot string   `json:"resolved_root" jsonschema:"absolute directory that was inspected"`
	Usages       []string `json:"usages" jsonschema:"list of files and lines containing // Deprecated: markers"`
}

type ModuleGraphInput struct {
	Root string `json:"root,omitempty" jsonschema:"absolute or relative directory to inspect; defaults to GO_WORKSPACE_ROOT or the current working directory"`
}

type DependencyEdge struct {
	From string `json:"from" jsonschema:"source module"`
	To   string `json:"to" jsonschema:"target module"`
}

type ModuleGraphOutput struct {
	ResolvedRoot string           `json:"resolved_root" jsonschema:"absolute directory that was inspected"`
	Edges        []DependencyEdge `json:"edges" jsonschema:"list of dependency edges"`
}
