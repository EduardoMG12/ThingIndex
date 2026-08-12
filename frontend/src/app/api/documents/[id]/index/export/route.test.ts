import { beforeEach, describe, expect, it, vi } from "vitest";
import { NextResponse } from "next/server";

const {
    cookiesMock,
    createForwardHeadersMock,
    localizedRouteErrorMock,
    parseProxyJsonWithFallbackMock,
    proxyJsonWithAuthCleanupMock,
    unauthorizedJsonMock,
} = vi.hoisted(() => ({
    cookiesMock: vi.fn(),
    createForwardHeadersMock: vi.fn(),
    localizedRouteErrorMock: vi.fn(),
    parseProxyJsonWithFallbackMock: vi.fn(),
    proxyJsonWithAuthCleanupMock: vi.fn(),
    unauthorizedJsonMock: vi.fn(),
}));

vi.mock("next/headers", () => ({
    cookies: cookiesMock,
}));

vi.mock("@/app/api/_utils/locale", () => ({
    createForwardHeaders: createForwardHeadersMock,
}));

vi.mock("@/app/api/_utils/auth", () => ({
    proxyJsonWithAuthCleanup: proxyJsonWithAuthCleanupMock,
    unauthorizedJson: unauthorizedJsonMock,
}));

vi.mock("@/app/api/_utils/messages", () => ({
    localizedRouteError: localizedRouteErrorMock,
    parseProxyJsonWithFallback: parseProxyJsonWithFallbackMock,
}));

import { GET } from "@/app/api/documents/[id]/index/export/route";

describe("document index export route", () => {
    beforeEach(() => {
        vi.restoreAllMocks();
        vi.stubGlobal("fetch", vi.fn());
        cookiesMock.mockResolvedValue({
            get: () => ({ value: "token-123" }),
        });
        createForwardHeadersMock.mockResolvedValue(new Headers({ Authorization: "Bearer token-123" }));
        unauthorizedJsonMock.mockResolvedValue(NextResponse.json({ message: "unauthorized" }, { status: 401 }));
        proxyJsonWithAuthCleanupMock.mockImplementation((payload: unknown, status: number) =>
            NextResponse.json(payload, { status })
        );
        parseProxyJsonWithFallbackMock.mockResolvedValue({ message: "upstream error" });
        localizedRouteErrorMock.mockResolvedValue(
            NextResponse.json({ message: "Could not export the document index." }, { status: 502 })
        );
    });

    it("streams the export attachment while sanitizing headers", async () => {
        vi.mocked(fetch).mockResolvedValue(
            new Response('{"kind":"document_index_export"}', {
                status: 200,
                headers: {
                    "Content-Type": "application/json; charset=utf-8",
                    "Content-Disposition": "attachment; filename*=UTF-8''Contrato%20final.thingindex-index.json",
                },
            })
        );

        const response = await GET(
            new Request("http://localhost/api/documents/9/index/export"),
            { params: Promise.resolve({ id: "9" }) }
        );

        expect(createForwardHeadersMock).toHaveBeenCalledWith(
            expect.any(Request),
            expect.objectContaining({
                token: "token-123",
                headers: expect.objectContaining({
                    "X-ThingIndex-Export-Origin": "http://localhost",
                }),
            })
        );
        expect(response.status).toBe(200);
        expect(response.headers.get("Content-Type")).toBe("application/json");
        expect(response.headers.get("Content-Disposition")).toBe(
            "attachment; filename*=UTF-8''Contrato%20final.thingindex-index.json"
        );
        expect(await response.text()).toContain("document_index_export");
    });

    it("returns a localized route error when the upstream fetch fails", async () => {
        vi.mocked(fetch).mockRejectedValue(new Error("upstream offline"));

        const response = await GET(
            new Request("http://localhost/api/documents/9/index/export"),
            { params: Promise.resolve({ id: "9" }) }
        );

        expect(localizedRouteErrorMock).toHaveBeenCalledWith(
            expect.any(Request),
            "apiRouteErrors.documents.indexExportLoad",
            502
        );
        expect(response.status).toBe(502);
    });
});
