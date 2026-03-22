package api

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveAPIReferenceOriginPrefersForwardedHeaders(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://internal/docs", nil)
	req.Host = "internal:8080"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "api.thingindex.dev")

	if got := resolveAPIReferenceOrigin(req); got != "https://api.thingindex.dev" {
		t.Fatalf("expected forwarded origin https://api.thingindex.dev, got %q", got)
	}
}

func TestRenderAPIReferencePageIncludesHostedAndSelfHostedPaths(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://example.com/docs", nil)
	body, err := renderAPIReferencePage(buildAPIReferencePageData(req))
	if err != nil {
		t.Fatalf("renderAPIReferencePage returned error: %v", err)
	}

	html := string(body)
	expectedFragments := []string{
		"ThingIndex API Reference",
		"http://example.com/docs",
		"http://example.com/swagger/index.html",
		"http://example.com/swagger/doc.json",
		"http://example.com/api/v1",
		"http://example.com/api/v1/providers/support?family=LLM",
		"Consume this deployment",
		"Run locally from GitHub",
		"./scripts/bootstrap.sh",
		"./scripts/stack.sh up",
		"api/docs/swagger.yaml",
		"docs/open-source-onboarding.md",
		"examples/http/thingindex-api.http",
		"./scripts/doctor.sh",
		"http://localhost:3000",
		"http://localhost:8000/api/v1",
		"@scalar/api-reference",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(html, fragment) {
			t.Fatalf("expected rendered html to contain %q", fragment)
		}
	}
}
