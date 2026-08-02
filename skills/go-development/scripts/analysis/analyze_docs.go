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

// DocIssue represents a documentation issue
type DocIssue struct {
	Package   string `json:"package"`
	Name      string `json:"name"`
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

	var issues []DocIssue

	// Walk Go files
	err := filepath.Walk(*rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.Contains(path, "vendor") {
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

func analyzeFile(filename string) []DocIssue {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", filename, err)
		return nil
	}

	var issues []DocIssue
	pkgName := f.Name.Name

	// Check package documentation
	if !hasPackageDoc(f) {
		issues = append(issues, DocIssue{
			Package:   pkgName,
			Name:      pkgName,
			IssueType: "missing_package_doc",
			File:      filename,
			Line:      fset.Position(f.Package).Line,
			Message:   "Package lacks documentation comment",
		})
	}

	// Check declarations
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() && !hasDoc(d.Doc) {
				issues = append(issues, DocIssue{
					Package:   pkgName,
					Name:      d.Name.Name,
					IssueType: "missing_func_doc",
					File:      filename,
					Line:      fset.Position(d.Pos()).Line,
					Message:   fmt.Sprintf("Exported function %s lacks documentation", d.Name.Name),
				})
			} else if hasDoc(d.Doc) && !startsWithName(d.Doc.Text(), d.Name.Name) {
				issues = append(issues, DocIssue{
					Package:   pkgName,
					Name:      d.Name.Name,
					IssueType: "invalid_doc_format",
					File:      filename,
					Line:      fset.Position(d.Pos()).Line,
					Message:   fmt.Sprintf("Documentation for %s should start with function name", d.Name.Name),
				})
			}

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() && !hasDoc(d.Doc) {
						issues = append(issues, DocIssue{
							Package:   pkgName,
							Name:      s.Name.Name,
							IssueType: "missing_type_doc",
							File:      filename,
							Line:      fset.Position(s.Pos()).Line,
							Message:   fmt.Sprintf("Exported type %s lacks documentation", s.Name.Name),
						})
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() && !hasDoc(d.Doc) {
							issues = append(issues, DocIssue{
								Package:   pkgName,
								Name:      name.Name,
								IssueType: "missing_value_doc",
								File:      filename,
								Line:      fset.Position(name.Pos()).Line,
								Message:   fmt.Sprintf("Exported value %s lacks documentation", name.Name),
							})
						}
					}
				}
			}
		}
	}

	return issues
}

func hasPackageDoc(f *ast.File) bool {
	// Check if there's a comment before package declaration
	if len(f.Comments) == 0 {
		return false
	}
	// First comment group should be before package
	firstComment := f.Comments[0]
	if f.Package > firstComment.Pos() {
		return false
	}
	// Check if it's a package comment (starts with package name)
	text := firstComment.Text()
	return strings.HasPrefix(text, "Package ") || strings.HasPrefix(text, "package ")
}

func hasDoc(cg *ast.CommentGroup) bool {
	return cg != nil && len(cg.List) > 0
}

func startsWithName(doc, name string) bool {
	// Extract first sentence
	parts := strings.SplitN(doc, ".", 2)
	firstSentence := strings.TrimSpace(parts[0])
	// Check if starts with name
	return strings.HasPrefix(firstSentence, name+" ") || firstSentence == name
}

func printIssues(issues []DocIssue) {
	if len(issues) == 0 {
		fmt.Println("All public APIs are documented!")
		return
	}

	byPackage := make(map[string][]DocIssue)
	for _, issue := range issues {
		byPackage[issue.Package] = append(byPackage[issue.Package], issue)
	}

	for pkg, pkgIssues := range byPackage {
		fmt.Printf("\n%s:\n", pkg)
		for _, issue := range pkgIssues {
			fmt.Printf("  [%s] %s:%d: %s\n", issue.IssueType, issue.File, issue.Line, issue.Message)
		}
	}

	fmt.Printf("\nTotal issues: %d\n", len(issues))
}
