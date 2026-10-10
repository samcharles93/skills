package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	mcpSrv *mcp.Server
}

func newServer() *mcp.Server {
	s := &Server{}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "go-workspace",
		Version: serverVersion,
	}, nil)
	s.mcpSrv = server

	mcp.AddTool(server, &mcp.Tool{
		Name:        "workspace_summary",
		Description: "Describe the active Go module or go.work layout for a target root",
	}, WorkspaceSummaryTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_packages",
		Description: "List Go packages under a target root with optional filtering and limits",
	}, ListPackagesTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "describe_package",
		Description: "Describe one Go package by import path or directory",
	}, DescribePackageTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "changed_packages",
		Description: "Map tracked or untracked git changes under a target root to Go packages",
	}, ChangedPackagesTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "run_tests_for_package",
		Description: "Run Go tests for a specific package and identify failures",
	}, s.handleRunTests)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_symbols",
		Description: "Find definitions of symbols (functions, types, variables) in the workspace",
	}, s.handleSearchSymbols)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "analyze_interface_impl",
		Description: "Find types that implement a specific Go interface",
	}, s.handleInterfaceImpl)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_dependents",
		Description: "Find all packages that import a target package to analyze blast radius",
	}, s.handleFindDependents)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_deprecated_usage",
		Description: "Scan the workspace for usage of deprecated symbols",
	}, s.handleFindDeprecated)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "module_dependency_graph",
		Description: "Return the project's module dependency graph",
	}, s.handleModuleGraph)

	registerPkgsiteTools(server, newPkgsiteClient())

	return server
}

func (s *Server) handleRunTests(ctx context.Context, req *mcp.CallToolRequest, input RunTestsInput) (*mcp.CallToolResult, RunTestsOutput, error) {
	_ = req

	inspection, err := inspectWorkspace(ctx, input.Root)
	if err != nil {
		return nil, RunTestsOutput{}, err
	}

	packageRef := input.PackageRef()
	if packageRef == "" {
		return nil, RunTestsOutput{}, fmt.Errorf("package is required")
	}

	pkg, err := findPackage(inspection.Packages, inspection.ResolvedRoot, packageRef)
	if err != nil {
		return nil, RunTestsOutput{}, err
	}

	output, err := runCommand(ctx, pkg.Dir, "go", "test", "-v", "./...")

	var outStr string
	if err != nil {
		outStr = string(output)
		if outStr == "" {
			outStr = fmt.Sprintf("Tests failed: %v", err)
		}
	} else {
		outStr = string(output)
	}

	success := err == nil
	failures := []string{}
	lines := strings.SplitSeq(outStr, "\n")
	for line := range lines {
		if strings.Contains(line, "--- FAIL:") {
			failures = append(failures, strings.TrimSpace(line))
		}
	}

	res := RunTestsOutput{
		ResolvedRoot: inspection.ResolvedRoot,
		Package:      pkg.ImportPath,
		Success:      success,
		Output:       outStr,
		Failures:     failures,
	}

	text := fmt.Sprintf("Ran tests for %s. Success: %v. Found %d failure(s).", res.Package, res.Success, len(res.Failures))
	return textResult(text), res, nil
}

func (s *Server) handleSearchSymbols(ctx context.Context, req *mcp.CallToolRequest, input SearchSymbolsInput) (*mcp.CallToolResult, SearchSymbolsOutput, error) {
	inspection, err := inspectWorkspace(ctx, input.Root)
	if err != nil {
		return nil, SearchSymbolsOutput{}, err
	}

	pattern := fmt.Sprintf(`(^|\s)(func|type|var)\s+%s(\s*\(|\s*struct|\s*=)`, input.Symbol)
	args := []string{"-rnE", pattern, "."}

	output, err := runCommand(ctx, inspection.ResolvedRoot, "grep", args...)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			return textResult("No symbols matching " + input.Symbol + " found."), SearchSymbolsOutput{ResolvedRoot: inspection.ResolvedRoot, Matches: []SymbolMatch{}}, nil
		}
		return nil, SearchSymbolsOutput{}, err
	}

	lines := strings.Split(string(output), "\n")
	matches := make([]SymbolMatch, 0)
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		matches = append(matches, SymbolMatch{
			File:    parts[0],
			Line:    0,
			Context: strings.TrimSpace(parts[2]),
		})
	}

	res := SearchSymbolsOutput{
		ResolvedRoot: inspection.ResolvedRoot,
		Matches:      matches,
	}

	text := fmt.Sprintf("Found %d definitions for symbol %s.", len(matches), input.Symbol)
	return textResult(text), res, nil
}

func (s *Server) handleInterfaceImpl(ctx context.Context, req *mcp.CallToolRequest, input InterfaceImplInput) (*mcp.CallToolResult, InterfaceImplOutput, error) {
	inspection, err := inspectWorkspace(ctx, input.Root)
	if err != nil {
		return nil, InterfaceImplOutput{}, err
	}

	pattern := fmt.Sprintf(`type\s+\w+\s+.*%s`, input.Iface)
	output, err := runCommand(ctx, inspection.ResolvedRoot, "grep", "-rnE", pattern, ".")
	if err != nil {
		return textResult("No implementations found."), InterfaceImplOutput{ResolvedRoot: inspection.ResolvedRoot}, nil
	}

	lines := strings.Split(string(output), "\n")
	impls := make([]string, 0)
	for _, line := range lines {
		if line == "" {
			continue
		}
		impls = append(impls, line)
	}

	res := InterfaceImplOutput{
		ResolvedRoot:    inspection.ResolvedRoot,
		Implementations: impls,
	}

	return textResult(fmt.Sprintf("Found %d potential implementations for %s.", len(impls), input.Iface)), res, nil
}

func (s *Server) handleFindDependents(ctx context.Context, req *mcp.CallToolRequest, input FindDependentsInput) (*mcp.CallToolResult, FindDependentsOutput, error) {
	inspection, err := inspectWorkspace(ctx, input.Root)
	if err != nil {
		return nil, FindDependentsOutput{}, err
	}

	output, err := runCommand(ctx, inspection.ResolvedRoot, "go", "list", "-f", `{{.ImportPath}} {{.Imports}}`, "./...")
	if err != nil {
		return nil, FindDependentsOutput{}, err
	}

	lines := strings.Split(string(output), "\n")
	dependents := make([]string, 0)
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		pkg := parts[0]
		imports := parts[1]
		if strings.Contains(imports, input.Pkg) {
			dependents = append(dependents, pkg)
		}
	}

	res := FindDependentsOutput{
		ResolvedRoot: inspection.ResolvedRoot,
		Dependents:   dependents,
	}

	return textResult(fmt.Sprintf("Found %d packages depending on %s.", len(dependents), input.Pkg)), res, nil
}

func (s *Server) handleFindDeprecated(ctx context.Context, req *mcp.CallToolRequest, input FindDeprecatedInput) (*mcp.CallToolResult, FindDeprecatedOutput, error) {
	inspection, err := inspectWorkspace(ctx, input.Root)
	if err != nil {
		return nil, FindDeprecatedOutput{}, err
	}

	output, err := runCommand(ctx, inspection.ResolvedRoot, "grep", "-rn", "// Deprecated:", ".")
	if err != nil {
		return textResult("No deprecated symbols found."), FindDeprecatedOutput{ResolvedRoot: inspection.ResolvedRoot}, nil
	}

	lines := strings.Split(string(output), "\n")
	usages := make([]string, 0)
	for _, line := range lines {
		if line == "" {
			continue
		}
		usages = append(usages, line)
	}

	res := FindDeprecatedOutput{
		ResolvedRoot: inspection.ResolvedRoot,
		Usages:       usages,
	}

	return textResult(fmt.Sprintf("Found %d deprecated markers.", len(usages))), res, nil
}

func (s *Server) handleModuleGraph(ctx context.Context, req *mcp.CallToolRequest, input ModuleGraphInput) (*mcp.CallToolResult, ModuleGraphOutput, error) {
	inspection, err := inspectWorkspace(ctx, input.Root)
	if err != nil {
		return nil, ModuleGraphOutput{}, err
	}

	output, err := runCommand(ctx, inspection.ResolvedRoot, "go", "mod", "graph")
	if err != nil {
		return nil, ModuleGraphOutput{}, err
	}

	lines := strings.Split(string(output), "\n")
	edges := make([]DependencyEdge, 0)
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 2 {
			edges = append(edges, DependencyEdge{
				From: parts[0],
				To:   parts[1],
			})
		}
	}

	res := ModuleGraphOutput{
		ResolvedRoot: inspection.ResolvedRoot,
		Edges:        edges,
	}

	return textResult(fmt.Sprintf("Parsed %d module dependency edges.", len(edges))), res, nil
}
