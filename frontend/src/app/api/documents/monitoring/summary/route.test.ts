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

import { GET } from "@/app/api/documents/monitoring/summary/route";

describe("documents monitoring summary route", () => {
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
        parseProxyJsonWithFallbackMock.mockResolvedValue({ status: 200, data: { total_documents: 1 } });
        localizedRouteErrorMock.mockResolvedValue(
            NextResponse.json({ message: "Could not load the monitoring summary." }, { status: 502 })
        );
    });

    it("forwards the backend payload with auth cleanup semantics", async () => {
        vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({ status: 200 }), { status: 200 }));

        const response = await GET(new Request("http://localhost/api/documents/monitoring/summary"));

        expect(createForwardHeadersMock).toHaveBeenCalled();
        expect(parseProxyJsonWithFallbackMock).toHaveBeenCalled();
        expect(proxyJsonWithAuthCleanupMock).toHaveBeenCalledWith({ status: 200, data: { total_documents: 1 } }, 200);
        expect(response.status).toBe(200);
    });

    it("returns a localized route error when the upstream fetch fails", async () => {
        vi.mocked(fetch).mockRejectedValue(new Error("upstream offline"));

        const response = await GET(new Request("http://localhost/api/documents/monitoring/summary"));

        expect(localizedRouteErrorMock).toHaveBeenCalledWith(
            expect.any(Request),
            "apiRouteErrors.documents.monitoringSummary",
            502
        );
        expect(response.status).toBe(502);
    });
});
