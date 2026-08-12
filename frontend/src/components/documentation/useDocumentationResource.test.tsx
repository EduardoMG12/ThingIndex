import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useDocumentationResource } from "@/components/documentation/useDocumentationResource";

describe("useDocumentationResource", () => {
    it("loads a documentation resource on mount", async () => {
        const loadResource = vi.fn().mockResolvedValue({ label: "Runtime profile" });

        const { result } = renderHook(() =>
            useDocumentationResource(loadResource, "Could not load the resource.")
        );

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        expect(loadResource).toHaveBeenCalledWith(expect.any(AbortSignal));
        expect(result.current.error).toBeNull();
        expect(result.current.resource).toEqual({ label: "Runtime profile" });
    });

    it("exposes the fallback error message when the loader fails with a non-Error value", async () => {
        const loadResource = vi.fn().mockRejectedValue("network down");

        const { result } = renderHook(() =>
            useDocumentationResource(loadResource, "Could not load the resource.")
        );

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        expect(result.current.resource).toBeNull();
        expect(result.current.error).toBe("Could not load the resource.");
    });
});
