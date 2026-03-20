import { renderHook, waitFor, act } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/categories", () => ({
    createCategory: vi.fn(),
    fetchCategories: vi.fn(),
}));

import { useCategories } from "@/hooks/use-categories";
import { createCategory, fetchCategories } from "@/lib/api/categories";

function createCategoryRecord(overrides: Record<string, unknown> = {}) {
    return {
        id: 1,
        user_id: 3,
        name: "Finance",
        description: "",
        created_at: "2026-03-19T00:00:00Z",
        updated_at: "2026-03-19T00:00:00Z",
        ...overrides,
    };
}

describe("useCategories", () => {
    beforeEach(() => {
        vi.mocked(fetchCategories).mockReset();
        vi.mocked(createCategory).mockReset();
    });

    it("loads categories on mount", async () => {
        vi.mocked(fetchCategories).mockResolvedValue([
            createCategoryRecord(),
            createCategoryRecord({ id: 2, name: "Legal" }),
        ]);

        const { result } = renderHook(() => useCategories());

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        expect(fetchCategories).toHaveBeenCalledWith(expect.any(AbortSignal));
        expect(result.current.error).toBeNull();
        expect(result.current.categories).toHaveLength(2);
    });

    it("creates a new category with trimmed input and stores it at the top of the list", async () => {
        vi.mocked(fetchCategories).mockResolvedValue([createCategoryRecord()]);
        vi.mocked(createCategory).mockResolvedValue(createCategoryRecord({ id: 4, name: "Legal" }));

        const { result } = renderHook(() => useCategories());

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        await act(async () => {
            await result.current.createAndStoreCategory("  Legal  ");
        });

        expect(createCategory).toHaveBeenCalledWith("Legal");
        expect(result.current.categories).toEqual([
            createCategoryRecord({ id: 4, name: "Legal" }),
            createCategoryRecord(),
        ]);
    });

    it("replaces existing categories when the normalized name already exists", async () => {
        vi.mocked(fetchCategories).mockResolvedValue([
            createCategoryRecord({
                id: 1,
                name: "Finance",
                description: "Old description",
            }),
        ]);
        vi.mocked(createCategory).mockResolvedValue(
            createCategoryRecord({
                id: 7,
                name: " finance ",
                description: "Updated description",
            })
        );

        const { result } = renderHook(() => useCategories());

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        await act(async () => {
            await result.current.createAndStoreCategory("Finance");
        });

        expect(result.current.categories).toEqual([
            createCategoryRecord({
                id: 7,
                name: " finance ",
                description: "Updated description",
            }),
        ]);
    });
});
