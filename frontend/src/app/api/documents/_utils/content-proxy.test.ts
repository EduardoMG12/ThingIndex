import { beforeEach, describe, expect, it, vi } from "vitest";
import { NextResponse } from "next/server";

const { cookiesMock, createForwardHeadersMock, getRequestLocaleMock, unauthorizedJsonMock } = vi.hoisted(() => ({
    cookiesMock: vi.fn(),
    createForwardHeadersMock: vi.fn(),
    getRequestLocaleMock: vi.fn(),
    unauthorizedJsonMock: vi.fn(),
}));

vi.mock("next/headers", () => ({
    cookies: cookiesMock,
}));

vi.mock("@/app/api/_utils/locale", () => ({
    createForwardHeaders: createForwardHeadersMock,
    getRequestLocale: getRequestLocaleMock,
}));

vi.mock("@/app/api/_utils/auth", () => ({
    unauthorizedJson: unauthorizedJsonMock,
}));

import { proxyDocumentContent } from "@/app/api/documents/_utils/content-proxy";

describe("content proxy", () => {
    beforeEach(() => {
        vi.restoreAllMocks();
        vi.stubGlobal("fetch", vi.fn());
        cookiesMock.mockResolvedValue({
            get: () => ({ value: "token-123" }),
        });
        createForwardHeadersMock.mockResolvedValue(new Headers({ Authorization: "Bearer token-123" }));
        getRequestLocaleMock.mockResolvedValue("pt-BR");
        unauthorizedJsonMock.mockImplementation(() => NextResponse.json({ message: "unauthorized" }, { status: 401 }));
    });

    it("blocks inline preview for unsafe html content", async () => {
        vi.mocked(fetch).mockResolvedValue(
            new Response("<html><body>active content</body></html>", {
                status: 200,
                headers: {
                    "Content-Type": "text/html; charset=utf-8",
                    "Content-Disposition": `attachment; filename="dangerous.html"`,
                },
            })
        );

        const response = await proxyDocumentContent(
            new Request("http://localhost/api/documents/9/preview"),
            { params: Promise.resolve({ id: "9" }) },
            "preview"
        );

        expect(response.status).toBe(415);
        expect(response.headers.get("Content-Type")).toBe("text/html; charset=utf-8");
        expect(response.headers.get("Content-Security-Policy")).toContain("sandbox");

        const body = await response.text();
        expect(body).toContain("Preview bloqueado por segurança");
        expect(body).toContain("download explícito do original");
        expect(body).not.toContain("active content");
    });

    it("sanitizes download headers while preserving streamed content", async () => {
        vi.mocked(fetch).mockResolvedValue(
            new Response("pdf", {
                status: 200,
                headers: {
                    "Content-Type": "application/pdf; charset=binary",
                    "Content-Disposition": "attachment; filename*=UTF-8''..%2Fevil%2F..%2Ffile.pdf",
                    "Content-Length": "3",
                },
            })
        );

        const response = await proxyDocumentContent(
            new Request("http://localhost/api/documents/9/download"),
            { params: Promise.resolve({ id: "9" }) },
            "download"
        );

        expect(response.status).toBe(200);
        expect(response.headers.get("Content-Type")).toBe("application/pdf");
        expect(response.headers.get("Content-Disposition")).toBe("attachment; filename*=UTF-8''file.pdf");
        expect(response.headers.get("Cross-Origin-Resource-Policy")).toBe("same-origin");
        expect(await response.text()).toBe("pdf");
    });

    it("keeps safe preview headers for pdf content", async () => {
        vi.mocked(fetch).mockResolvedValue(
            new Response("pdf", {
                status: 200,
                headers: {
                    "Content-Type": "application/pdf",
                    "Content-Disposition": `inline; filename="relatorio final.pdf"`,
                },
            })
        );

        const response = await proxyDocumentContent(
            new Request("http://localhost/api/documents/9/preview"),
            { params: Promise.resolve({ id: "9" }) },
            "preview"
        );

        expect(response.status).toBe(200);
        expect(response.headers.get("Content-Disposition")).toBe("inline; filename*=UTF-8''relatorio%20final.pdf");
        expect(response.headers.get("X-Frame-Options")).toBe("SAMEORIGIN");
        expect(response.headers.get("Referrer-Policy")).toBe("no-referrer");
    });
});
