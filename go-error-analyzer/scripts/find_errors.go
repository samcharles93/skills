// +build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ErrorIssue represents an error handling issue
type ErrorIssue struct {
	Package   string `json:"package"`
	Function  string `json:"function"`
	IssueType string `json:"issue_type"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Message   string `json:"message"`
}

func main() {
	var (
		format  = flag.String("format", "text", "Output format: text or json")
		rootDir = flag.String("root", ".", "Root directory to analyze")
	)
	flag.Parse()

	var issues []ErrorIssue

	err := filepath.Walk(*rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.Contains(path, "_test.go") || strings.Contains(path, "vendor") {
			return nil
		}

		fileIssues := analyzeFile(path)
		issues = append(issues, fileIssues...)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(issues)
	} else {
		printIssues(issues)
	}
}

func analyzeFile(filename string) []ErrorIssue {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", filename, err)
		return nil
	}

	var issues []ErrorIssue
	pkgName := f.Name.Name

	// Analyze all functions
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// Check function body for error issues
		if fn.Body != nil {
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch stmt := n.(type) {
				case *ast.AssignStmt:
					checkAssignment(stmt, &issues, pkgName, fn.Name.Name, filename, fset)
				case *ast.ExprStmt:
					checkExpression(stmt, &issues, pkgName, fn.Name.Name, filename, fset)
				}
				return true
			})
		}
	}

	return issues
}

func checkAssignment(stmt *ast.AssignStmt, issues *[]ErrorIssue, pkgName, funcName, filename string, fset *token.FileSet) {
	// Check for discarded errors: _, _ = ... or _, err := ... where err is unused
	if len(stmt.Lhs) == 2 && len(stmt.Rhs) == 1 {
		// Check if left side has blank identifier
		if ident1, ok := stmt.Lhs[0].(*ast.Ident); ok && ident1.Name == "_" {
			if ident2, ok := stmt.Lhs[1].(*ast.Ident); ok {
				// Check if it's likely an error
				if ident2.Name == "_" || strings.Contains(ident2.Name, "err") {
					*issues = append(*issues, ErrorIssue{
						Package:   pkgName,
						Function:  funcName,
						IssueType: "discarded_error",
						File:      filename,
						Line:      fset.Position(stmt.Pos()).Line,
						Message:   "Error value discarded with blank identifier",
					})
				}
			}
		}
	}

	// Check if error is assigned but not checked
	if len(stmt.Lhs) >= 1 {
		for _, lhs := range stmt.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok && ident.Name == "err" {
				// This is an error assignment - we'll track if it's checked
				// In a real implementation, we'd track this across statements
				_ = ident
			}
		}
	}
}

func checkExpression(stmt *ast.ExprStmt, issues *[]ErrorIssue, pkgName, funcName, filename string, fset *token.FileSet) {
	// Check for function calls that return errors but aren't checked
	if call, ok := stmt.X.(*ast.CallExpr); ok {
		if isLikelyErrorReturning(call) {
			*issues = append(*issues, ErrorIssue{
				Package:   pkgName,
				Function:  funcName,
				IssueType: "unchecked_error",
				File:      filename,
				Line:      fset.Position(stmt.Pos()).Line,
				Message:   fmt.Sprintf("Function call likely returns error but is not checked"),
			})
		}
	}
}

func isLikelyErrorReturning(call *ast.CallExpr) bool {
	// Check if it's a method call on a type that commonly returns errors
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		methodName := sel.Sel.Name
		// Common methods that return errors
		errorReturningMethods := []string{
			"Close", "Write", "Read", "WriteString", "WriteByte",
			"Flush", "Sync", "Seek", "Marshal", "Unmarshal",
			"Encode", "Decode", "Parse", "Scan",
		}
		for _, m := range errorReturningMethods {
			if methodName == m {
				return true
			}
		}
	}
	return false
}

func printIssues(issues []ErrorIssue) {
	if len(issues) == 0 {
		fmt.Println("No unhandled errors found!")
		return
	}

	byFile := make(map[string][]ErrorIssue)
	for _, issue := range issues {
		byFile[issue.File] = append(byFile[issue.File], issue)
	}

	for file, fileIssues := range byFile {
		fmt.Printf("\n%s:\n", file)
		for _, issue := range fileIssues {
			fmt.Printf("  [%s] %s:%d: %s\n", issue.IssueType, issue.Function, issue.Line, issue.Message)
		}
	}

	fmt.Printf("\nTotal issues: %d\n", len(issues))
}
