import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/fetch-with-auth-redirect", () => ({
    fetchWithAuthRedirect: vi.fn(),
}));

import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";
import { listProviderModels } from "@/lib/api/providers";

describe("providers api", () => {
    beforeEach(() => {
        vi.mocked(fetchWithAuthRedirect).mockReset();
    });

    it("loads and normalizes discovered models", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: {
                        models: ["gpt-4o-mini", " gpt-4.1-mini ", "", null],
                    },
                }),
                { status: 200, headers: { "Content-Type": "application/json" } }
            )
        );

        const models = await listProviderModels({
            family: "LLM",
            providerType: "OPENAI",
            credentials: {
                api_key: "secret",
            },
        });

        expect(vi.mocked(fetchWithAuthRedirect).mock.calls[0]?.[0]).toBe("/api/user/providers/models");
        expect(models).toEqual(["gpt-4o-mini", "gpt-4.1-mini"]);
    });
});
