import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import IndexingScopeCard from "@/components/settings/IndexingScopeCard";
import { createTestWrapper } from "@/test/test-utils";

vi.mock("next/navigation", () => ({ usePathname: () => "/pt-BR/settings" }));

const fetchMock = vi.fn();
vi.mock("@/lib/api/fetch-with-auth-redirect", () => ({ fetchWithAuthRedirect: (...args: unknown[]) => fetchMock(...args) }));

function resposta(data: Record<string, boolean>) {
    return { ok: true, json: async () => ({ status: "success", message: "ok", data }) };
}

describe("IndexingScopeCard", () => {
    beforeEach(() => fetchMock.mockReset());

    it("le as preferencias e salva ao desligar um toggle", async () => {
        fetchMock
            .mockResolvedValueOnce(resposta({ document_synopsis: true, page_summaries: true, section_summaries: false }))
            .mockResolvedValueOnce(resposta({ document_synopsis: true, page_summaries: false, section_summaries: false }));
        render(<IndexingScopeCard />, { wrapper: createTestWrapper("pt-BR") });
        await waitFor(() => expect(screen.getByTestId("escopo-sectionSummaries")).toHaveAttribute("aria-checked", "false"));
        expect(screen.getByTestId("escopo-pageSummaries")).toHaveAttribute("aria-checked", "true");

        fireEvent.click(screen.getByTestId("escopo-pageSummaries"));
        await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
        const [url, init] = fetchMock.mock.calls[1] as [string, RequestInit];
        expect(url).toBe("/api/user/indexing-preferences");
        expect(init.method).toBe("PUT");
        expect(JSON.parse(String(init.body))).toEqual({ document_synopsis: true, page_summaries: false, section_summaries: false });
        await waitFor(() => expect(screen.getByTestId("escopo-pageSummaries")).toHaveAttribute("aria-checked", "false"));
    });
});
