import { useState } from "react";

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { DangerConfirmModal } from "@/components/ui/danger-confirm-modal";

function ControlledModal(props: {
    onCancel?: () => void;
    onConfirm?: () => void;
    isPending?: boolean;
}) {
    const [value, setValue] = useState("");

    return (
        <DangerConfirmModal
            isOpen
            eyebrow="Danger"
            title="Delete collection"
            description="This action cannot be undone."
            confirmLabel="Delete now"
            expectedValue="DELETE"
            confirmationValue={value}
            onConfirmationChange={setValue}
            onCancel={props.onCancel ?? vi.fn()}
            onConfirm={props.onConfirm ?? vi.fn()}
            isPending={props.isPending}
        />
    );
}

describe("DangerConfirmModal", () => {
    it("does not render when it is closed", () => {
        const { container } = render(
            <DangerConfirmModal
                isOpen={false}
                eyebrow="Danger"
                title="Delete"
                description="No return path."
                confirmLabel="Confirm"
                onCancel={vi.fn()}
                onConfirm={vi.fn()}
            />
        );

        expect(container).toBeEmptyDOMElement();
    });

    it("requires the expected confirmation value before enabling the destructive action", async () => {
        const user = userEvent.setup();
        const onConfirm = vi.fn();

        render(<ControlledModal onConfirm={onConfirm} />);

        const confirmButton = screen.getByRole("button", { name: "Delete now" });
        expect(confirmButton).toBeDisabled();

        await user.type(screen.getByPlaceholderText("DELETE"), "  DELETE  ");

        expect(confirmButton).toBeEnabled();

        await user.click(confirmButton);
        expect(onConfirm).toHaveBeenCalledTimes(1);
    });

    it("lets the user dismiss the modal from the backdrop control", async () => {
        const user = userEvent.setup();
        const onCancel = vi.fn();

        render(<ControlledModal onCancel={onCancel} />);

        await user.click(screen.getByRole("button", { name: "Fechar confirmacao" }));
        expect(onCancel).toHaveBeenCalledTimes(1);
    });

    it("disables both actions while the destructive request is pending", () => {
        render(<ControlledModal isPending />);

        expect(screen.getByRole("button", { name: "Cancelar" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Delete now" })).toBeDisabled();
    });
});
