package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type recordedRequest struct {
	path  string
	query string
}

func fakePkgsite(t *testing.T, routes map[string]string) *[]recordedRequest {
	t.Helper()
	var got []recordedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, recordedRequest{path: r.URL.EscapedPath(), query: r.URL.RawQuery})
		w.Header().Set("Content-Type", "application/json")
		body, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":404,"message":"not found"}`))
			return
		}
		if strings.HasPrefix(body, "!") {
			w.WriteHeader(http.StatusBadRequest)
			body = body[1:]
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv(pkgsiteURLEnv, srv.URL+"/v1")
	return &got
}

func callTool(t *testing.T, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx := context.Background()
	session := newClientSession(t, ctx)
	t.Cleanup(func() { _ = session.Close() })
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	return res
}

func TestPkgsitePackage(t *testing.T) {
	got := fakePkgsite(t, map[string]string{
		"/v1/package/github.com/google/go-cmp/cmp": `{"modulePath":"github.com/google/go-cmp","version":"v0.7.0","isLatest":true,"path":"github.com/google/go-cmp/cmp","name":"cmp","synopsis":"Package cmp determines equality of values.","docs":"` + strings.Repeat("x", maxDocBytes+10) + `"}`,
	})

	res := callTool(t, "pkgsite_package", map[string]any{
		"path": "github.com/google/go-cmp/cmp", "version": "master", "doc": "md", "imports": true,
	})
	if res.IsError {
		t.Fatalf("isError: %s", firstText(t, res))
	}
	var out PkgsitePackage
	decodeStructured(t, res, &out)
	if out.ModulePath != "github.com/google/go-cmp" || !out.IsLatest {
		t.Fatalf("unexpected output: %+v", out)
	}
	if !out.DocsTruncated || len(out.Docs) != maxDocBytes {
		t.Fatalf("docs not truncated: len=%d truncated=%v", len(out.Docs), out.DocsTruncated)
	}
	if q := (*got)[0].query; q != "doc=md&imports=true&version=master" {
		t.Fatalf("unexpected query %q", q)
	}
}

func TestPkgsiteVersionsPagination(t *testing.T) {
	got := fakePkgsite(t, map[string]string{
		"/v1/versions/example.com/m": `{"items":[{"modulePath":"example.com/m","version":"v1.2.0","retracted":true,"retractionReason":"bad"}],"total":3,"nextPageToken":"abc"}`,
	})

	res := callTool(t, "pkgsite_versions", map[string]any{"path": "example.com/m", "limit": 500, "token": "prev"})
	if res.IsError {
		t.Fatalf("isError: %s", firstText(t, res))
	}
	var out Page[PkgsiteModuleVersion]
	decodeStructured(t, res, &out)
	if out.Total != 3 || out.NextPageToken != "abc" || !out.Items[0].Retracted {
		t.Fatalf("unexpected output: %+v", out)
	}
	if !strings.Contains(firstText(t, res), `token="abc"`) {
		t.Fatalf("summary does not offer next token: %q", firstText(t, res))
	}
	if q := (*got)[0].query; q != "limit=100&token=prev" {
		t.Fatalf("limit not capped or token not passed: %q", q)
	}
}

func TestPkgsiteUnknownTotal(t *testing.T) {
	fakePkgsite(t, map[string]string{
		"/v1/versions/example.com/m": `{"items":[{"modulePath":"example.com/m","version":"v1.0.0"}],"total":-1}`,
	})

	res := callTool(t, "pkgsite_versions", map[string]any{"path": "example.com/m"})
	if text := firstText(t, res); !strings.Contains(text, "unknown number") {
		t.Fatalf("total of -1 not reported as unknown: %q", text)
	}
}

func TestPkgsiteEmptyPage(t *testing.T) {
	fakePkgsite(t, map[string]string{
		"/v1/vulns/example.com/m": `{"items":null,"total":0}`,
	})

	res := callTool(t, "pkgsite_vulns", map[string]any{"path": "example.com/m"})
	if res.IsError {
		t.Fatalf("isError on empty page: %s", firstText(t, res))
	}
}

func TestPkgsiteAmbiguousPathReportsCandidates(t *testing.T) {
	fakePkgsite(t, map[string]string{
		"/v1/package/example.com/a/b/c": `!{"code":400,"message":"ambiguous path","candidates":[{"modulePath":"example.com/a","packagePath":"example.com/a/b/c"},{"modulePath":"example.com/a/b","packagePath":"example.com/a/b/c"}]}`,
	})

	res := callTool(t, "pkgsite_package", map[string]any{"path": "example.com/a/b/c"})
	if !res.IsError {
		t.Fatal("expected isError for ambiguous path")
	}
	text := firstText(t, res)
	for _, want := range []string{"ambiguous", "example.com/a ", "example.com/a/b"} {
		if !strings.Contains(text, want) {
			t.Fatalf("error %q missing %q", text, want)
		}
	}
}

func TestPkgsiteEscapesPathSegments(t *testing.T) {
	got := fakePkgsite(t, map[string]string{})

	callTool(t, "pkgsite_symbols", map[string]any{"path": "/example.com/a b/"})
	if p := (*got)[0].path; p != "/v1/symbols/example.com/a%20b" {
		t.Fatalf("unexpected path %q", p)
	}
}

func TestPkgsiteRequiresInput(t *testing.T) {
	fakePkgsite(t, map[string]string{})

	for name, args := range map[string]map[string]any{
		"pkgsite_search":  {"query": " "},
		"pkgsite_package": {"path": ""},
	} {
		if res := callTool(t, name, args); !res.IsError {
			t.Errorf("%s: expected isError for empty input", name)
		}
	}
}
