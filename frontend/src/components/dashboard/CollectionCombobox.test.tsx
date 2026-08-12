import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { createTestWrapper } from "@/test/test-utils";

const {
    useCollectionsMock,
} = vi.hoisted(() => ({
    useCollectionsMock: vi.fn(),
}));

vi.mock("@/hooks/use-collections", () => ({
    useCollections: useCollectionsMock,
}));

import CollectionCombobox from "@/components/dashboard/CollectionCombobox";

describe("CollectionCombobox", () => {
    it("shows an inline load warning and retries when the input gains focus after an error", async () => {
        const user = userEvent.setup();
        const reloadCollections = vi.fn();

        useCollectionsMock.mockReturnValue({
            collections: [],
            isLoading: false,
            error: "Erro ao consultar collections.",
            reloadCollections,
        });

        render(<CollectionCombobox value="" onChange={vi.fn()} />, {
            wrapper: createTestWrapper("en"),
        });

        await user.click(screen.getByRole("textbox"));

        expect(reloadCollections).toHaveBeenCalledTimes(1);
        expect(
            screen.getByText("Could not load existing collections. You can still type a new one or refresh the list.")
        ).toBeInTheDocument();
        expect(screen.getByText("Erro ao consultar collections.")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Refresh status" })).toBeInTheDocument();
    });
});
