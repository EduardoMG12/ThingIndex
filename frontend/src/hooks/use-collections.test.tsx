import { renderHook, waitFor, act } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/collections", () => ({
    fetchCollections: vi.fn(),
}));

import { useCollections } from "@/hooks/use-collections";
import { fetchCollections } from "@/lib/api/collections";

function createCollection(overrides: Record<string, unknown> = {}) {
    return {
        id: 1,
        name: "Beta",
        documentCount: 2,
        metadata: {},
        ...overrides,
    };
}

describe("useCollections", () => {
    beforeEach(() => {
        vi.mocked(fetchCollections).mockReset();
    });

    it("loads collections on mount", async () => {
        vi.mocked(fetchCollections).mockResolvedValue([
            createCollection(),
            createCollection({ id: 2, name: "Alpha" }),
        ]);

        const { result } = renderHook(() => useCollections());

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        expect(fetchCollections).toHaveBeenCalledWith(expect.any(AbortSignal));
        expect(result.current.error).toBeNull();
        expect(result.current.collections).toHaveLength(2);
    });

    it("merges new draft collections and keeps the list sorted", async () => {
        vi.mocked(fetchCollections).mockResolvedValue([createCollection()]);

        const { result } = renderHook(() => useCollections());

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        act(() => {
            result.current.mergeCollection("  alpha  ");
        });

        expect(result.current.collections).toEqual([
            expect.objectContaining({
                id: "draft:alpha",
                name: "alpha",
                documentCount: 0,
            }),
            expect.objectContaining({
                id: 1,
                name: "Beta",
            }),
        ]);
    });

    it("replaces existing collections when the name only changes by casing", async () => {
        vi.mocked(fetchCollections).mockResolvedValue([
            createCollection({
                id: 1,
                name: "Case A",
                documentCount: 1,
            }),
        ]);

        const { result } = renderHook(() => useCollections());

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        act(() => {
            result.current.mergeCollection({
                id: 7,
                name: "case a",
                documentCount: 5,
                metadata: {
                    synced: true,
                },
            });
        });

        expect(result.current.collections).toEqual([
            {
                id: 7,
                name: "case a",
                documentCount: 5,
                metadata: {
                    synced: true,
                },
            },
        ]);
    });

    it("reloads collections explicitly without reusing the initial abort signal", async () => {
        vi.mocked(fetchCollections)
            .mockResolvedValueOnce([createCollection()])
            .mockResolvedValueOnce([createCollection({ id: 3, name: "Gamma" })]);

        const { result } = renderHook(() => useCollections());

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        await act(async () => {
            await result.current.reloadCollections();
        });

        expect(fetchCollections).toHaveBeenNthCalledWith(2);
        expect(result.current.collections).toEqual([
            expect.objectContaining({
                id: 3,
                name: "Gamma",
            }),
        ]);
    });
});
