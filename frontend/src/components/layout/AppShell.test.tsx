import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { pathnameState } = vi.hoisted(() => ({ pathnameState: { value: "/es/workspace" } }));

vi.mock("next/navigation", () => ({
    usePathname: () => pathnameState.value,
    useRouter: () => ({ push: vi.fn(), replace: vi.fn(), refresh: vi.fn(), prefetch: vi.fn(), back: vi.fn() }),
}));

vi.mock("@/components/auth/actions", () => ({ logout: vi.fn() }));

vi.mock("@/lib/api/current-user", () => ({
    fetchCurrentUser: vi.fn(async () => null),
    updateCurrentUserProfile: vi.fn(),
}));

vi.mock("@/lib/api/chat", () => ({
    deleteChatConversation: vi.fn(),
    updateChatConversationTitle: vi.fn(),
}));

import AppShell from "@/components/layout/AppShell";
import { createTestWrapper } from "@/test/test-utils";

describe("AppShell edicao do titulo da conversa (F-14)", () => {
    beforeEach(() => {
        pathnameState.value = "/es/workspace";
    });

    it("botoes de cancelar e salvar a edicao tem nome acessivel no locale", async () => {
        render(<AppShell>conteudo</AppShell>, { wrapper: createTestWrapper("es") });

        await act(async () => {
            window.dispatchEvent(
                new CustomEvent("thingindex:chat-sidebar-sync", {
                    detail: {
                        selectedCollectionId: 1,
                        activeConversationId: 9,
                        conversations: [{ id: 9, title: "Conversa", collection_id: 1 }],
                    },
                })
            );
        });

        fireEvent.click(screen.getByRole("button", { name: "Renombrar conversación" }));

        expect(screen.getByRole("button", { name: "Cancelar edición del título" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Guardar título" })).toBeInTheDocument();
    });
});
