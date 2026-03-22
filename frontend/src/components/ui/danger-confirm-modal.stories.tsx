import { useState, type ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { expect, userEvent, within } from "storybook/test";

import { DangerConfirmModal } from "@/components/ui/danger-confirm-modal";

type DangerConfirmModalStoryProps = ComponentProps<typeof DangerConfirmModal>;

const baseArgs: DangerConfirmModalStoryProps = {
    isOpen: true,
    eyebrow: "Destructive action",
    title: "Delete collection",
    description: "This removes the documents and their derived indexing data.",
    confirmLabel: "Confirm deletion",
    cancelLabel: "Cancel",
    closeLabel: "Close confirmation",
    expectedValue: "Operations",
    inputLabel: "Confirmation value",
    confirmationPrompt: "Type Operations to confirm the removal.",
    isPending: false,
    onCancel: () => undefined,
    onConfirm: () => undefined,
};

const meta = {
    title: "UI/DangerConfirmModal",
    component: DangerConfirmModal,
    tags: ["autodocs"],
    args: baseArgs,
    parameters: {
        layout: "centered",
    },
} satisfies Meta<typeof DangerConfirmModal>;

export default meta;

type Story = StoryObj<typeof meta>;

function ControlledDangerConfirmModal(args: DangerConfirmModalStoryProps) {
    const [value, setValue] = useState("");

    return (
        <DangerConfirmModal
            {...args}
            confirmationValue={value}
            onConfirmationChange={setValue}
            onCancel={() => undefined}
            onConfirm={() => undefined}
        />
    );
}

export const RequiresTypedConfirmation: Story = {
    args: baseArgs,
    render: (args) => <ControlledDangerConfirmModal {...args} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement);
        const input = canvas.getByLabelText("Confirmation value");
        const confirmButton = canvas.getByRole("button", { name: "Confirm deletion" });

        await expect(confirmButton).toBeDisabled();
        await userEvent.type(input, "Operations");
        await expect(confirmButton).toBeEnabled();
    },
};

export const PendingState: Story = {
    args: {
        ...baseArgs,
        isPending: true,
    },
    render: (args) => <ControlledDangerConfirmModal {...args} />,
};
