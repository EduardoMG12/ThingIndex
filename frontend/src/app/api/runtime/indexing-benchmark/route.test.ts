import { beforeEach, describe, expect, it, vi } from "vitest";

const { createForwardHeadersMock } = vi.hoisted(() => ({
    createForwardHeadersMock: vi.fn(),
}));

vi.mock("@/app/api/_utils/locale", () => ({
    createForwardHeaders: createForwardHeadersMock,
}));

import { GET } from "@/app/api/runtime/indexing-benchmark/route";

describe("indexing benchmark route", () => {
    beforeEach(() => {
        vi.restoreAllMocks();
        vi.stubGlobal("fetch", vi.fn());
        createForwardHeadersMock.mockResolvedValue(new Headers({ "Accept-Language": "en" }));
    });

    it("forwards the public indexing benchmark payload", async () => {
        vi.mocked(fetch).mockResolvedValue(
            new Response(JSON.stringify({ status: 200, data: { dataset_version: "2026-03-relational-baseline-v1" } }), { status: 200 })
        );

        const response = await GET(new Request("http://localhost/api/runtime/indexing-benchmark"));

        expect(createForwardHeadersMock).toHaveBeenCalled();
        expect(vi.mocked(fetch).mock.calls[0]?.[0]).toBe("http://api:8080/api/v1/meta/indexing-benchmark");
        expect(response.status).toBe(200);
    });

    it("returns a 502 payload when the upstream fetch fails", async () => {
        vi.mocked(fetch).mockRejectedValue(new Error("upstream offline"));

        const response = await GET(new Request("http://localhost/api/runtime/indexing-benchmark"));
        const body = await response.json();

        expect(response.status).toBe(502);
        expect(body.message).toContain("benchmark publico");
    });
});
