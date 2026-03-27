# ThingIndex Security Checklist

Status: current runtime review for the mainline repository.  
Scope: application-level controls, abuse cases and operator-facing hardening.  
This file documents what is already enforced, what is covered by tests, and what is still explicitly out of scope.

## 1. Upload abuse controls

- [x] Request body size is capped before multipart parsing.
- [x] Upload validation checks empty files, declared MIME, sniffed MIME and extension compatibility.
- [x] Spoofed PDF declarations are rejected.
- [x] Spreadsheet declarations without spreadsheet extension are rejected.
- [x] Unsupported MIME types are rejected before storage.
- [x] Request ID is surfaced back to the frontend on upload failures for traceability.

## 2. Preview and download hardening

- [x] Inline preview is limited to safe content types (`application/pdf` and `text/plain`).
- [x] HTML files may still be stored, but they are not rendered inline by the preview route.
- [x] `Content-Disposition` is rebuilt from sanitized filenames instead of replaying unsafe upstream values.
- [x] Preview/download responses use `Cache-Control: no-store`.
- [x] Preview/download responses use `X-Content-Type-Options: nosniff`.
- [x] Preview/download responses use `Referrer-Policy: no-referrer`.
- [x] Preview error documents are HTML-escaped and rendered under a restrictive CSP sandbox.

## 3. Provider endpoint and SSRF controls

- [x] Storage endpoints are validated against loopback, link-local, private and other restricted ranges by default.
- [x] Storage hostname resolution is revalidated before dialing to reduce DNS rebinding bypasses.
- [x] LLM `base_url` must be valid and is restricted to HTTPS outside development.
- [x] Private LLM base URLs are rejected outside the allowed development path.
- [x] Public metadata derived from provider URLs is sanitized before being stored in audit payloads.

## 4. Error exposure policy

- [x] Auth, document and provider handlers return stable user-facing messages instead of raw internal errors.
- [x] Provider connection errors are classified into auth, endpoint, timeout, model or generic failure buckets.
- [x] Orphaned document content returns a controlled `410 Gone` response instead of leaking storage internals.
- [x] Preview fallback pages avoid echoing raw HTML from upstream content.

## 5. Objective coverage

The current repository includes automated coverage for:

- upload MIME spoofing and size enforcement;
- storage endpoint validation and restricted host blocking;
- provider credential and endpoint validation;
- content disposition sanitization;
- preview blocking for unsafe inline content;
- frontend filename sanitization during download.

## 6. Residual risks and explicit non-claims

The repository does **not** currently claim:

- antivirus or malware scanning of uploaded files;
- production-grade sandboxing of browser PDF viewers;
- formal pentest or external security audit;
- infrastructure hardening beyond the current local/development oriented runtime;
- full DLP or policy enforcement for uploaded content.

These items remain future hardening work and should not be implied as delivered.
