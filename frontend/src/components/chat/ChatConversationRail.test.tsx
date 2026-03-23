import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ChatConversationRail } from "@/components/chat/ChatConversationRail";
import { createTestWrapper } from "@/test/test-utils";

const conversations = [
    {
        id: 10,
        collectionId: 7,
        title: "Quarterly review",
        lastMessageAt: "2026-03-20T12:15:00Z",
        createdAt: "2026-03-20T12:00:00Z",
        updatedAt: "2026-03-20T12:15:00Z",
    },
    {
        id: 11,
        collectionId: 7,
        title: "Risk register",
        lastMessageAt: "2026-03-20T13:00:00Z",
        createdAt: "2026-03-20T12:30:00Z",
        updatedAt: "2026-03-20T13:00:00Z",
    },
];

describe("ChatConversationRail", () => {
    it("selects, creates, and deletes conversations from the workspace rail", async () => {
        const user = userEvent.setup();
        const onSelectConversation = vi.fn();
        const onStartNewConversation = vi.fn();
        const onDeleteConversation = vi.fn().mockResolvedValue(undefined);

        render(
            <ChatConversationRail
                conversations={conversations}
                activeConversationId={10}
                isLoading={false}
                deletingConversationId={null}
                isWorkspaceBusy={false}
                onSelectConversation={onSelectConversation}
                onStartNewConversation={onStartNewConversation}
                onDeleteConversation={onDeleteConversation}
            />,
            { wrapper: createTestWrapper("en") }
        );

        await user.click(screen.getByRole("button", { name: /Risk register/i }));
        expect(onSelectConversation).toHaveBeenCalledWith(11);

        await user.click(screen.getByRole("button", { name: "New chat" }));
        expect(onStartNewConversation).toHaveBeenCalledTimes(1);

        await user.click(screen.getAllByRole("button", { name: "Delete conversation" })[0]);
        await user.type(screen.getByLabelText("Confirmation value"), "Quarterly review");
        await user.click(screen.getByRole("button", { name: "Confirm deletion" }));

        expect(onDeleteConversation).toHaveBeenCalledWith(10);
    });
});
