package api

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"
)

type apiReferencePageData struct {
	CurrentOrigin   string
	CurrentAPIBase  string
	ScalarURL       string
	SwaggerUIURL    string
	SwaggerJSONURL  string
	SwaggerYAMLRef  string
	OnboardingRef   string
	ExamplesRef     string
	LocalFrontendUI string
	LocalAPIBase    string
}

var apiReferencePageTemplate = template.Must(template.New("api-reference").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>ThingIndex API Reference</title>
    <style>
      :root {
        color-scheme: light;
        font-family: Inter, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      }

      body {
        margin: 0;
        background: #f4efe6;
        color: #1f2937;
      }

      main {
        max-width: 1100px;
        margin: 0 auto;
        padding: 40px 20px 72px;
      }

      .hero,
      .fallback {
        border: 1px solid #ddd4c5;
        border-radius: 24px;
        background: rgba(255, 255, 255, 0.94);
        box-shadow: 0 24px 60px rgba(15, 23, 42, 0.08);
      }

      .hero {
        padding: 28px;
      }

      .eyebrow {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        border: 1px solid #d8b778;
        border-radius: 999px;
        background: #fff7e6;
        padding: 8px 12px;
        font-size: 11px;
        font-weight: 700;
        letter-spacing: 0.2em;
        text-transform: uppercase;
        color: #a16207;
      }

      h1 {
        margin: 18px 0 12px;
        font-size: clamp(2rem, 4vw, 3.25rem);
        line-height: 1.05;
      }

      p {
        margin: 0;
        line-height: 1.7;
      }

      .resource-grid {
        display: grid;
        gap: 16px;
        margin-top: 28px;
        grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
      }

      .resource-card {
        border: 1px solid #e6dfd0;
        border-radius: 18px;
        background: #fffdf9;
        padding: 18px;
      }

      .resource-card h2 {
        margin: 0 0 8px;
        font-size: 0.98rem;
      }

      .resource-card a {
        color: #0f766e;
        font-weight: 700;
        text-decoration: none;
      }

      .resource-card a:hover {
        text-decoration: underline;
      }

      .fallback {
        margin-top: 24px;
        padding: 24px 28px;
      }

      code {
        border-radius: 8px;
        background: #f8f4ec;
        padding: 2px 8px;
        font-size: 0.92em;
      }

      ul {
        margin: 12px 0 0;
        padding-left: 20px;
      }

      #scalar-shell {
        margin-top: 28px;
        overflow: hidden;
        border-radius: 24px;
        border: 1px solid #ddd4c5;
        min-height: 540px;
        background: #ffffff;
      }
    </style>
  </head>
  <body>
    <main>
      <section class="hero">
        <span class="eyebrow">ThingIndex API</span>
        <h1>Reference and fallback docs for open-source consumers</h1>
        <p>
          This page works for both entry paths: consuming the current deployment directly or cloning
          the repository and running the full stack locally. Scalar stays the primary reference,
          with Swagger UI and raw OpenAPI JSON available as fallbacks.
        </p>

        <div class="resource-grid">
          <article class="resource-card">
            <h2>Scalar API reference</h2>
            <p>Primary interactive documentation view served from this API.</p>
            <p><a href="{{.ScalarURL}}">{{.ScalarURL}}</a></p>
          </article>
          <article class="resource-card">
            <h2>Swagger UI</h2>
            <p>Fallback local viewer when the Scalar embed cannot render.</p>
            <p><a href="{{.SwaggerUIURL}}">{{.SwaggerUIURL}}</a></p>
          </article>
          <article class="resource-card">
            <h2>OpenAPI JSON</h2>
            <p>Raw contract for generators and tooling.</p>
            <p><a href="{{.SwaggerJSONURL}}">{{.SwaggerJSONURL}}</a></p>
          </article>
          <article class="resource-card">
            <h2>Repository references</h2>
            <p>See <code>{{.ExamplesRef}}</code> for ready-made requests and <code>{{.SwaggerYAMLRef}}</code> for the versioned contract.</p>
          </article>
        </div>
      </section>

      <section class="fallback">
        <h2>Two supported entry paths</h2>
        <div class="resource-grid">
          <article class="resource-card">
            <h2>Consume this deployment</h2>
            <p>Use the current origin directly when the API is already online.</p>
            <ul>
              <li>Current origin: <code>{{.CurrentOrigin}}</code></li>
              <li>API base: <code>{{.CurrentAPIBase}}</code></li>
              <li>Scalar docs: <code>{{.ScalarURL}}</code></li>
              <li>Swagger UI fallback: <code>{{.SwaggerUIURL}}</code></li>
            </ul>
          </article>
          <article class="resource-card">
            <h2>Run locally from GitHub</h2>
            <p>Clone the repository when you need the frontend, workers, local troubleshooting and versioned docs history.</p>
            <ul>
              <li>Local UI: <code>{{.LocalFrontendUI}}</code></li>
              <li>Local API: <code>{{.LocalAPIBase}}</code></li>
              <li><code>./scripts/bootstrap.sh</code></li>
              <li><code>./scripts/stack.sh up</code></li>
              <li><code>./scripts/doctor.sh</code></li>
            </ul>
          </article>
        </div>
        <p style="margin-top:16px">
          If the embedded Scalar UI does not load, use Swagger UI or the raw OpenAPI JSON directly.
          For repository setup and troubleshooting, the guide lives at <code>{{.OnboardingRef}}</code>.
        </p>
      </section>

      <div id="scalar-shell">
        <script id="api-reference" data-url="{{.SwaggerJSONURL}}"></script>
      </div>
    </main>

    <noscript>
      <main>
        <section class="fallback">
          <h2>JavaScript disabled</h2>
          <p>Use Swagger UI or the raw OpenAPI JSON with the links above.</p>
        </section>
      </main>
    </noscript>

    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>
`))

func resolveAPIReferenceOrigin(r *http.Request) string {
	if r == nil {
		return "http://localhost:8000"
	}

	host := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if host == "" {
		host = "localhost:8000"
	}

	scheme := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	return scheme + "://" + host
}

func buildAPIReferencePageData(r *http.Request) apiReferencePageData {
	currentOrigin := strings.TrimRight(resolveAPIReferenceOrigin(r), "/")

	return apiReferencePageData{
		CurrentOrigin:   currentOrigin,
		CurrentAPIBase:  currentOrigin + "/api/v1",
		ScalarURL:       currentOrigin + "/docs",
		SwaggerUIURL:    currentOrigin + "/swagger/index.html",
		SwaggerJSONURL:  currentOrigin + "/swagger/doc.json",
		SwaggerYAMLRef:  "api/docs/swagger.yaml",
		OnboardingRef:   "docs/open-source-onboarding.md",
		ExamplesRef:     "examples/http/thingindex-api.http",
		LocalFrontendUI: "http://localhost:3000",
		LocalAPIBase:    "http://localhost:8000/api/v1",
	}
}

func renderAPIReferencePage(data apiReferencePageData) ([]byte, error) {
	var body bytes.Buffer

	if err := apiReferencePageTemplate.Execute(&body, data); err != nil {
		return nil, err
	}

	return body.Bytes(), nil
}
