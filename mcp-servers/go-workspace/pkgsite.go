package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultPkgsiteURL = "https://pkg.go.dev/v1"
	pkgsiteURLEnv     = "PKGSITE_API_URL"
	defaultPageLimit  = 20
	maxPageLimit      = 100
	maxDocBytes       = 64 << 10
	maxResponseBytes  = 8 << 20
)

type pkgsiteClient struct {
	baseURL string
	http    *http.Client
}

func newPkgsiteClient() *pkgsiteClient {
	base := strings.TrimRight(os.Getenv(pkgsiteURLEnv), "/")
	if base == "" {
		base = defaultPkgsiteURL
	}
	return &pkgsiteClient{
		baseURL: base,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// PkgsiteError is the error body returned by the pkg.go.dev API.
type PkgsiteError struct {
	Code       int                `json:"code"`
	Message    string             `json:"message"`
	Fixes      []string           `json:"fixes,omitempty"`
	Candidates []PkgsiteCandidate `json:"candidates,omitempty"`
}

type PkgsiteCandidate struct {
	ModulePath  string `json:"modulePath" jsonschema:"module that could contain the path"`
	PackagePath string `json:"packagePath" jsonschema:"package path within that module"`
}

func (e *PkgsiteError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "pkg.go.dev: %d %s", e.Code, e.Message)
	for _, f := range e.Fixes {
		fmt.Fprintf(&b, "; fix: %s", f)
	}
	if len(e.Candidates) > 0 {
		b.WriteString("; ambiguous, retry with module set to one of:")
		for _, c := range e.Candidates {
			fmt.Fprintf(&b, " %s", c.ModulePath)
		}
	}
	return b.String()
}

func pkgsiteGet[T any](ctx context.Context, c *pkgsiteClient, endpoint, path string, query url.Values) (T, error) {
	var zero T

	u := c.baseURL + "/" + endpoint
	if path != "" {
		path = strings.Trim(strings.TrimSpace(path), "/")
		segments := strings.Split(path, "/")
		for i, s := range segments {
			segments[i] = url.PathEscape(s)
		}
		u += "/" + strings.Join(segments, "/")
	}
	if enc := query.Encode(); enc != "" {
		u += "?" + enc
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-workspace-mcp/"+serverVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return zero, fmt.Errorf("pkg.go.dev %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return zero, fmt.Errorf("read pkg.go.dev %s response: %w", endpoint, err)
	}

	if resp.StatusCode != http.StatusOK {
		apiErr := &PkgsiteError{}
		if json.Unmarshal(body, apiErr) != nil || apiErr.Message == "" {
			apiErr = &PkgsiteError{Code: resp.StatusCode, Message: strings.TrimSpace(string(body))}
		}
		if apiErr.Code == 0 {
			apiErr.Code = resp.StatusCode
		}
		return zero, apiErr
	}

	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		return zero, fmt.Errorf("decode pkg.go.dev %s response: %w", endpoint, err)
	}
	return out, nil
}

type queryBuilder url.Values

func (q queryBuilder) str(key, val string) {
	if val = strings.TrimSpace(val); val != "" {
		url.Values(q).Set(key, val)
	}
}

func (q queryBuilder) flag(key string, val bool) {
	if val {
		url.Values(q).Set(key, "true")
	}
}

func (q queryBuilder) page(p PageInput) {
	limit := p.Limit
	switch {
	case limit <= 0:
		limit = defaultPageLimit
	case limit > maxPageLimit:
		limit = maxPageLimit
	}
	url.Values(q).Set("limit", strconv.Itoa(limit))
	q.str("token", p.Token)
	q.str("filter", p.Filter)
}

// PageInput is embedded by every paginated tool input.
type PageInput struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum items to return; default 20, capped at 100"`
	Token  string `json:"token,omitempty" jsonschema:"nextPageToken from a previous call; repeat that call's other arguments unchanged"`
	Filter string `json:"filter,omitempty" jsonschema:"boolean Go expression over the item's JSON fields, e.g. kind == \"Type\", hasPrefix(version, \"v2.\"), matches(path, \"^cmd/\"); supports ==, !=, <, >, +, contains, hasPrefix, hasSuffix, matches"`
}

type Page[T any] struct {
	Items         []T    `json:"items"`
	Total         int    `json:"total" jsonschema:"total matching items, or -1 when unknown"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type VersionInput struct {
	Module  string `json:"module,omitempty" jsonschema:"module path; required when the package path could belong to more than one module"`
	Version string `json:"version,omitempty" jsonschema:"semantic version, latest, master or main; defaults to the latest tagged version"`
}

type PkgsiteSearchInput struct {
	Query  string `json:"query" jsonschema:"search terms for packages"`
	Symbol string `json:"symbol,omitempty" jsonschema:"find symbols with this name, restricted to packages matching query"`
	PageInput
}

type PkgsiteSearchResult struct {
	PackagePath string `json:"packagePath"`
	ModulePath  string `json:"modulePath"`
	Version     string `json:"version"`
	Synopsis    string `json:"synopsis"`
}

type PkgsitePackageInput struct {
	Path string `json:"path" jsonschema:"package import path, e.g. github.com/google/go-cmp/cmp or net/http"`
	VersionInput
	GOOS     string `json:"goos,omitempty" jsonschema:"GOOS of the documentation build context"`
	GOARCH   string `json:"goarch,omitempty" jsonschema:"GOARCH of the documentation build context"`
	Doc      string `json:"doc,omitempty" jsonschema:"documentation format: text or md; omit to skip documentation"`
	Examples bool   `json:"examples,omitempty" jsonschema:"include examples in the documentation"`
	Imports  bool   `json:"imports,omitempty" jsonschema:"include the packages this package imports"`
}

type PkgsitePackage struct {
	Path              string   `json:"path"`
	Name              string   `json:"name"`
	ModulePath        string   `json:"modulePath"`
	Version           string   `json:"version"`
	IsLatest          bool     `json:"isLatest"`
	IsStandardLibrary bool     `json:"isStandardLibrary"`
	IsRedistributable bool     `json:"isRedistributable"`
	Synopsis          string   `json:"synopsis"`
	GOOS              string   `json:"goos"`
	GOARCH            string   `json:"goarch"`
	Docs              string   `json:"docs,omitempty"`
	DocsTruncated     bool     `json:"docsTruncated,omitempty"`
	Imports           []string `json:"imports,omitempty"`
}

type PkgsiteModuleInput struct {
	Path    string `json:"path" jsonschema:"module path"`
	Version string `json:"version,omitempty" jsonschema:"semantic version, latest, master or main; defaults to the latest tagged version"`
	Readme  bool   `json:"readme,omitempty" jsonschema:"include the README"`
}

type PkgsiteReadme struct {
	Filepath string `json:"filepath"`
	Contents string `json:"contents"`
}

type PkgsiteModule struct {
	Path              string         `json:"path"`
	Version           string         `json:"version"`
	CommitTime        string         `json:"commitTime"`
	IsLatest          bool           `json:"isLatest"`
	IsStandardLibrary bool           `json:"isStandardLibrary"`
	IsRedistributable bool           `json:"isRedistributable"`
	HasGoMod          bool           `json:"hasGoMod"`
	GoModContents     string         `json:"goModContents,omitempty"`
	RepoURL           string         `json:"repoUrl,omitempty"`
	Readme            *PkgsiteReadme `json:"readme,omitempty"`
}

type PkgsiteVersionsInput struct {
	Path   string `json:"path" jsonschema:"module path"`
	Pseudo bool   `json:"pseudo,omitempty" jsonschema:"include pseudo-versions"`
	PageInput
}

type PkgsiteModuleVersion struct {
	ModulePath        string `json:"modulePath"`
	Version           string `json:"version"`
	LatestVersion     string `json:"latestVersion,omitempty"`
	CommitTime        string `json:"commitTime"`
	HasGoMod          bool   `json:"hasGoMod"`
	IsRedistributable bool   `json:"isRedistributable"`
	Deprecated        bool   `json:"deprecated,omitempty"`
	DeprecationReason string `json:"deprecationReason,omitempty"`
	Retracted         bool   `json:"retracted,omitempty"`
	RetractionReason  string `json:"retractionReason,omitempty"`
}

type PkgsitePackagesInput struct {
	Path    string `json:"path" jsonschema:"module path"`
	Version string `json:"version,omitempty" jsonschema:"semantic version, latest, master or main; defaults to the latest tagged version"`
	PageInput
}

type PkgsitePackageInfo struct {
	Path              string `json:"path"`
	Name              string `json:"name"`
	Synopsis          string `json:"synopsis"`
	IsRedistributable bool   `json:"isRedistributable"`
}

type PkgsitePackages struct {
	ModulePath        string                   `json:"modulePath"`
	Version           string                   `json:"version"`
	IsStandardLibrary bool                     `json:"isStandardLibrary"`
	Packages          Page[PkgsitePackageInfo] `json:"packages"`
}

type PkgsiteSymbolsInput struct {
	Path string `json:"path" jsonschema:"package import path"`
	VersionInput
	GOOS   string `json:"goos,omitempty" jsonschema:"GOOS of the documentation build context"`
	GOARCH string `json:"goarch,omitempty" jsonschema:"GOARCH of the documentation build context"`
	PageInput
}

type PkgsiteSymbol struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Parent   string `json:"parent,omitempty"`
	Synopsis string `json:"synopsis"`
}

type PkgsiteSymbols struct {
	ModulePath string              `json:"modulePath"`
	Version    string              `json:"version"`
	Symbols    Page[PkgsiteSymbol] `json:"symbols"`
}

type PkgsiteImportedByInput struct {
	Path string `json:"path" jsonschema:"package import path"`
	VersionInput
	PageInput
}

type PkgsiteImportedBy struct {
	ModulePath string       `json:"modulePath"`
	Version    string       `json:"version"`
	ImportedBy Page[string] `json:"importedBy"`
}

type PkgsiteVulnsInput struct {
	Path string `json:"path" jsonschema:"module or package path"`
	VersionInput
	PageInput
}

type PkgsiteVulnerability struct {
	ID           string `json:"id"`
	Summary      string `json:"summary"`
	Details      string `json:"details,omitempty"`
	FixedVersion string `json:"fixedVersion,omitempty"`
}

func requirePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("path is required")
	}
	return nil
}

func registerPkgsiteTools(server *mcp.Server, c *pkgsiteClient) {
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(true)}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pkgsite_search",
		Description: "Search pkg.go.dev for packages, or for symbols by name, ranked by relevance",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in PkgsiteSearchInput) (*mcp.CallToolResult, Page[PkgsiteSearchResult], error) {
		if strings.TrimSpace(in.Query) == "" {
			return nil, Page[PkgsiteSearchResult]{}, errors.New("query is required")
		}
		q := queryBuilder{}
		q.str("q", in.Query)
		q.str("symbol", in.Symbol)
		q.page(in.PageInput)
		out, err := pkgsiteGet[Page[PkgsiteSearchResult]](ctx, c, "search", "", url.Values(q))
		if err != nil {
			return nil, out, err
		}
		return textResult(fmt.Sprintf("%d of %s result(s) for %q.%s", len(out.Items), count(out.Total), in.Query, more(out.NextPageToken))), out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pkgsite_package",
		Description: "Get a package from pkg.go.dev at a version: synopsis, module, latest flag, and optionally its documentation and imports",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in PkgsitePackageInput) (*mcp.CallToolResult, PkgsitePackage, error) {
		if err := requirePath(in.Path); err != nil {
			return nil, PkgsitePackage{}, err
		}
		q := queryBuilder{}
		q.str("module", in.Module)
		q.str("version", in.Version)
		q.str("goos", in.GOOS)
		q.str("goarch", in.GOARCH)
		q.str("doc", in.Doc)
		q.flag("examples", in.Examples)
		q.flag("imports", in.Imports)
		out, err := pkgsiteGet[PkgsitePackage](ctx, c, "package", in.Path, url.Values(q))
		if err != nil {
			return nil, out, err
		}
		if len(out.Docs) > maxDocBytes {
			out.Docs = out.Docs[:maxDocBytes]
			out.DocsTruncated = true
		}
		return textResult(fmt.Sprintf("%s %s (module %s): %s", out.Path, out.Version, out.ModulePath, out.Synopsis)), out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pkgsite_module",
		Description: "Get a module from pkg.go.dev at a version: repository, commit time, go.mod contents, and optionally the README",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in PkgsiteModuleInput) (*mcp.CallToolResult, PkgsiteModule, error) {
		if err := requirePath(in.Path); err != nil {
			return nil, PkgsiteModule{}, err
		}
		q := queryBuilder{}
		q.str("version", in.Version)
		q.flag("readme", in.Readme)
		out, err := pkgsiteGet[PkgsiteModule](ctx, c, "module", in.Path, url.Values(q))
		if err != nil {
			return nil, out, err
		}
		return textResult(fmt.Sprintf("%s %s (latest: %v).", out.Path, out.Version, out.IsLatest)), out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pkgsite_versions",
		Description: "List a module's versions newest first, across major versions, with deprecation and retraction status",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in PkgsiteVersionsInput) (*mcp.CallToolResult, Page[PkgsiteModuleVersion], error) {
		if err := requirePath(in.Path); err != nil {
			return nil, Page[PkgsiteModuleVersion]{}, err
		}
		q := queryBuilder{}
		q.flag("pseudo", in.Pseudo)
		q.page(in.PageInput)
		out, err := pkgsiteGet[Page[PkgsiteModuleVersion]](ctx, c, "versions", in.Path, url.Values(q))
		if err != nil {
			return nil, out, err
		}
		latest := ""
		if len(out.Items) > 0 {
			latest = " Newest: " + out.Items[0].Version + "."
		}
		return textResult(fmt.Sprintf("%d of %s version(s) of %s.%s%s", len(out.Items), count(out.Total), in.Path, latest, more(out.NextPageToken))), out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pkgsite_packages",
		Description: "List the packages in a module on pkg.go.dev",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in PkgsitePackagesInput) (*mcp.CallToolResult, PkgsitePackages, error) {
		if err := requirePath(in.Path); err != nil {
			return nil, PkgsitePackages{}, err
		}
		q := queryBuilder{}
		q.str("version", in.Version)
		q.page(in.PageInput)
		out, err := pkgsiteGet[PkgsitePackages](ctx, c, "packages", in.Path, url.Values(q))
		if err != nil {
			return nil, out, err
		}
		return textResult(fmt.Sprintf("%d of %s package(s) in %s %s.%s", len(out.Packages.Items), count(out.Packages.Total), out.ModulePath, out.Version, more(out.Packages.NextPageToken))), out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pkgsite_symbols",
		Description: "List the exported symbols of a package at a version, with kind, parent type and signature",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in PkgsiteSymbolsInput) (*mcp.CallToolResult, PkgsiteSymbols, error) {
		if err := requirePath(in.Path); err != nil {
			return nil, PkgsiteSymbols{}, err
		}
		q := queryBuilder{}
		q.str("module", in.Module)
		q.str("version", in.Version)
		q.str("goos", in.GOOS)
		q.str("goarch", in.GOARCH)
		q.page(in.PageInput)
		out, err := pkgsiteGet[PkgsiteSymbols](ctx, c, "symbols", in.Path, url.Values(q))
		if err != nil {
			return nil, out, err
		}
		return textResult(fmt.Sprintf("%d of %s symbol(s) in %s %s.%s", len(out.Symbols.Items), count(out.Symbols.Total), in.Path, out.Version, more(out.Symbols.NextPageToken))), out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pkgsite_imported_by",
		Description: "List packages outside its own module that import a package, to gauge adoption or the impact of a change",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in PkgsiteImportedByInput) (*mcp.CallToolResult, PkgsiteImportedBy, error) {
		if err := requirePath(in.Path); err != nil {
			return nil, PkgsiteImportedBy{}, err
		}
		q := queryBuilder{}
		q.str("module", in.Module)
		q.str("version", in.Version)
		q.page(in.PageInput)
		out, err := pkgsiteGet[PkgsiteImportedBy](ctx, c, "imported-by", in.Path, url.Values(q))
		if err != nil {
			return nil, out, err
		}
		return textResult(fmt.Sprintf("%s is imported by %s package(s); returned %d.%s", in.Path, count(out.ImportedBy.Total), len(out.ImportedBy.Items), more(out.ImportedBy.NextPageToken))), out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pkgsite_vulns",
		Description: "List known vulnerabilities (Go vulnerability database) affecting a module or package at a version, with fixed versions",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in PkgsiteVulnsInput) (*mcp.CallToolResult, Page[PkgsiteVulnerability], error) {
		if err := requirePath(in.Path); err != nil {
			return nil, Page[PkgsiteVulnerability]{}, err
		}
		q := queryBuilder{}
		q.str("module", in.Module)
		q.str("version", in.Version)
		q.page(in.PageInput)
		out, err := pkgsiteGet[Page[PkgsiteVulnerability]](ctx, c, "vulns", in.Path, url.Values(q))
		if err != nil {
			return nil, out, err
		}
		return textResult(fmt.Sprintf("%s known vulnerabilit(ies) for %s.%s", count(out.Total), in.Path, more(out.NextPageToken))), out, nil
	})
}

func count(n int) string {
	if n < 0 {
		return "an unknown number of"
	}
	return strconv.Itoa(n)
}

func more(token string) string {
	if token == "" {
		return ""
	}
	return " More available: pass token=" + strconv.Quote(token) + "."
}
