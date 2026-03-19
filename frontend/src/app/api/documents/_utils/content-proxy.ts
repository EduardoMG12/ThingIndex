import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

export type DocumentContentRouteContext = {
    params: Promise<{
        id: string;
    }>;
};

type ContentProxyMode = "preview" | "download";

export async function proxyDocumentContent(
    request: Request,
    context: DocumentContentRouteContext,
    mode: ContentProxyMode
) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    const { id } = await context.params;

    try {
        const response = await fetch(`${API_URL}/documents/${id}/content`, {
            method: "GET",
            headers: await createForwardHeaders(request, { token }),
            cache: "no-store",
        });

        if (response.status === 401) {
            return unauthorizedJson();
        }

        if (!response.ok || !response.body) {
            const payload = await response.json().catch(() => ({
                message: defaultErrorMessage(mode, response.status),
            }));
            const message = typeof payload?.message === "string" ? payload.message : defaultErrorMessage(mode, response.status);

            if (mode === "preview") {
                return renderPreviewFailure(message, response.status);
            }

            return NextResponse.json({ message }, { status: response.status });
        }

        const headers = new Headers();
        headers.set("Content-Type", response.headers.get("content-type") || "application/octet-stream");
        headers.set("Content-Disposition", buildDispositionHeader(mode, response.headers.get("content-disposition")));
        headers.set("Cache-Control", "no-store");
        headers.set("X-Content-Type-Options", "nosniff");
        headers.set("Referrer-Policy", "no-referrer");

        const contentLength = response.headers.get("content-length");
        if (contentLength) {
            headers.set("Content-Length", contentLength);
        }

        return new NextResponse(response.body, {
            status: response.status,
            headers,
        });
    } catch {
        if (mode === "preview") {
            return renderPreviewFailure("Erro ao carregar o preview do documento.", 502);
        }

        return NextResponse.json(
            { message: "Erro ao baixar o documento original." },
            { status: 502 }
        );
    }
}

function buildDispositionHeader(mode: ContentProxyMode, upstreamDisposition: string | null) {
    const filenameDirective = extractFilenameDirective(upstreamDisposition);
    const dispositionType = mode === "preview" ? "inline" : "attachment";

    return filenameDirective
        ? `${dispositionType}; ${filenameDirective}`
        : `${dispositionType}; filename="documento"`;
}

function extractFilenameDirective(disposition: string | null) {
    if (!disposition) {
        return null;
    }

    const parts = disposition
        .split(";")
        .map((part) => part.trim())
        .filter(Boolean);

    return parts.find((part) => {
        const lowerPart = part.toLowerCase();
        return lowerPart.startsWith("filename=") || lowerPart.startsWith("filename*=");
    }) ?? null;
}

function defaultErrorMessage(mode: ContentProxyMode, status: number) {
    if (mode === "download") {
        return "Nao foi possivel baixar o documento original.";
    }

    if (status === 410) {
        return "O arquivo original nao foi encontrado no storage. O documento esta orfao e precisa de reupload ou exclusao.";
    }

    return "Nao foi possivel carregar o preview do documento.";
}

function renderPreviewFailure(message: string, status: number) {
    const escapedMessage = escapeHtml(message);
    const title = status === 410 ? "Arquivo original indisponivel" : "Preview indisponivel";
    const helperText = status === 410
        ? "Reenvie o arquivo para restaurar o original ou exclua o registro desatualizado."
        : "Tente novamente em instantes ou use o download explicito do original quando disponivel.";

    return new NextResponse(
        `<!doctype html>
<html lang="pt-BR">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>${title}</title>
    <style>
      :root {
        color-scheme: light dark;
        font-family: "IBM Plex Sans", ui-sans-serif, system-ui, sans-serif;
      }
      body {
        margin: 0;
        min-height: 100vh;
        display: grid;
        place-items: center;
        background:
          radial-gradient(circle at top, rgba(201, 93, 33, 0.18), transparent 42%),
          linear-gradient(180deg, #f4efe8 0%, #efe7dd 100%);
        color: #2f241d;
      }
      .panel {
        width: min(560px, calc(100vw - 32px));
        border: 1px solid rgba(120, 72, 37, 0.18);
        border-radius: 24px;
        padding: 28px;
        background: rgba(255, 252, 247, 0.94);
        box-shadow: 0 18px 54px rgba(70, 42, 22, 0.12);
      }
      .eyebrow {
        margin: 0 0 10px;
        font-size: 11px;
        font-weight: 700;
        letter-spacing: 0.24em;
        text-transform: uppercase;
        color: #b45309;
      }
      h1 {
        margin: 0;
        font-size: 1.1rem;
      }
      p {
        margin: 12px 0 0;
        line-height: 1.55;
        font-size: 0.95rem;
      }
      .hint {
        margin-top: 18px;
        padding-top: 18px;
        border-top: 1px solid rgba(120, 72, 37, 0.12);
        color: #5f4b3d;
      }
      @media (prefers-color-scheme: dark) {
        body {
          background:
            radial-gradient(circle at top, rgba(245, 158, 11, 0.18), transparent 42%),
            linear-gradient(180deg, #1b1713 0%, #15120f 100%);
          color: #f5eee7;
        }
        .panel {
          background: rgba(29, 24, 20, 0.94);
          border-color: rgba(251, 191, 36, 0.18);
          box-shadow: 0 18px 54px rgba(0, 0, 0, 0.28);
        }
        .hint {
          border-top-color: rgba(251, 191, 36, 0.14);
          color: #d6c7ba;
        }
      }
    </style>
  </head>
  <body>
    <main class="panel">
      <p class="eyebrow">ThingIndex Preview</p>
      <h1>${title}</h1>
      <p>${escapedMessage}</p>
      <p class="hint">${helperText}</p>
    </main>
  </body>
</html>`,
        {
            status,
            headers: {
                "Content-Type": "text/html; charset=utf-8",
                "Cache-Control": "no-store",
                "X-Content-Type-Options": "nosniff",
                "Content-Security-Policy": "default-src 'none'; style-src 'unsafe-inline'; sandbox;",
            },
        }
    );
}

function escapeHtml(value: string) {
    return value
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll(`"`, "&quot;")
        .replaceAll("'", "&#39;");
}
