import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { fetchApiUpstream, getApiBaseCandidates } from "@/app/api/_utils/upstream";

describe("api upstream helper", () => {
    beforeEach(() => {
        vi.restoreAllMocks();
        vi.stubGlobal("fetch", vi.fn());
    });

    afterEach(() => {
        delete process.env.API_INTERNAL_URL;
        delete process.env.NEXT_PUBLIC_API_URL;
    });

    it("deduplicates configured and default API base candidates", () => {
        process.env.API_INTERNAL_URL = "http://api:8080/api/v1/";
        process.env.NEXT_PUBLIC_API_URL = "http://localhost:8000/api/v1";

        expect(getApiBaseCandidates()).toEqual([
            "http://api:8080/api/v1",
            "http://localhost:8000/api/v1",
        ]);
    });

    it("falls back to the host API when the container hostname is unreachable", async () => {
        process.env.API_INTERNAL_URL = "http://api:8080/api/v1";
        process.env.NEXT_PUBLIC_API_URL = "http://localhost:8000/api/v1";

        vi.mocked(fetch)
            .mockRejectedValueOnce(new Error("getaddrinfo ENOTFOUND api"))
            .mockResolvedValueOnce(new Response(JSON.stringify({ ok: true }), { status: 200 }));

        const response = await fetchApiUpstream("/documents/collections", {
            method: "GET",
        });

        expect(vi.mocked(fetch).mock.calls[0]?.[0]).toBe("http://api:8080/api/v1/documents/collections");
        expect(vi.mocked(fetch).mock.calls[1]?.[0]).toBe("http://localhost:8000/api/v1/documents/collections");
        expect(response.status).toBe(200);
    });

    it("falls back to the container API when localhost is unreachable inside docker", async () => {
        process.env.API_INTERNAL_URL = "http://localhost:8000/api/v1";
        process.env.NEXT_PUBLIC_API_URL = "http://localhost:8000/api/v1";

        vi.mocked(fetch)
            .mockRejectedValueOnce(new Error("connect ECONNREFUSED 127.0.0.1:8000"))
            .mockResolvedValueOnce(new Response(JSON.stringify({ ok: true }), { status: 200 }));

        const response = await fetchApiUpstream("/user/providers", {
            method: "GET",
        });

        expect(vi.mocked(fetch).mock.calls[0]?.[0]).toBe("http://localhost:8000/api/v1/user/providers");
        expect(vi.mocked(fetch).mock.calls[1]?.[0]).toBe("http://api:8080/api/v1/user/providers");
        expect(response.status).toBe(200);
    });
});
