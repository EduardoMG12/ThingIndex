import { beforeEach, describe, expect, it, vi } from "vitest";

import { fetchWithAuthRedirect, UnauthorizedRequestError } from "@/lib/api/fetch-with-auth-redirect";
import { LOCALE_COOKIE_NAME, LOCALE_HEADER_NAME } from "@/lib/i18n/config";

describe("fetch-with-auth-redirect", () => {
    beforeEach(() => {
        vi.unstubAllGlobals();
        document.cookie = `${LOCALE_COOKIE_NAME}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
        document.documentElement.lang = "";
        vi.stubGlobal("fetch", vi.fn());
    });

    it("forwards the normalized locale in request headers", async () => {
        const fetchMock = vi.mocked(fetch);
        fetchMock.mockResolvedValue(new Response(null, { status: 200 }));

        document.cookie = `${LOCALE_COOKIE_NAME}=pt`;

        await fetchWithAuthRedirect("/api/documents", {
            method: "GET",
            headers: {
                "X-Test": "enabled",
            },
        });

        expect(fetchMock).toHaveBeenCalledTimes(1);

        const headers = fetchMock.mock.calls[0][1]?.headers as Headers;
        expect(headers.get("Accept-Language")).toBe("pt-BR");
        expect(headers.get(LOCALE_HEADER_NAME)).toBe("pt-BR");
        expect(headers.get("X-Test")).toBe("enabled");
    });

    it("redirects unauthorized requests to the localized login page", async () => {
        const fetchMock = vi.mocked(fetch);
        const assign = vi.fn();

        fetchMock.mockResolvedValue(
            new Response(JSON.stringify({ message: "unauthorized" }), {
                status: 401,
                headers: {
                    "Content-Type": "application/json",
                },
            })
        );

        document.documentElement.lang = "es";
        vi.stubGlobal("window", {
            location: {
                pathname: "/es/dashboard",
                assign,
            },
        });

        await expect(fetchWithAuthRedirect("/api/documents")).rejects.toBeInstanceOf(UnauthorizedRequestError);
        expect(assign).toHaveBeenCalledWith("/es/login");
    });

    it("avoids redirect loops when already on the localized login page", async () => {
        const fetchMock = vi.mocked(fetch);
        const assign = vi.fn();

        fetchMock.mockResolvedValue(new Response(null, { status: 401 }));

        document.documentElement.lang = "en";
        vi.stubGlobal("window", {
            location: {
                pathname: "/en/login",
                assign,
            },
        });

        await expect(fetchWithAuthRedirect("/api/documents")).rejects.toBeInstanceOf(UnauthorizedRequestError);
        expect(assign).not.toHaveBeenCalled();
    });
});
