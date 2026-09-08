import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import ChatTab from "@/components/dashboard/ChatTab";
import { createTestWrapper } from "@/test/test-utils";

vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/workspace",
    useRouter: () => ({ replace: vi.fn(), push: vi.fn() }),
    useSearchParams: () => new URLSearchParams(),
}));

const usado = vi.fn();
vi.mock("@/hooks/use-chat-workspace", () => ({
    useChatWorkspace: (opcoes: unknown) => {
        usado(opcoes);
        return {
            activeCollection: { id: 3, name: "SI" },
            activeConversation: null,
            activeSubQueries: [],
            canSubmit: false,
            canType: true,
            cancelStreaming: vi.fn(),
            collections: [{ id: 3, name: "SI" }],
            collectionsLoading: false,
            hasChatRuntime: true,
            hasCollections: true,
            input: "",
            isConversationLoading: false,
            isLoading: false,
            isProviderRuntimeLoading: false,
            isStreaming: false,
            messages: [],
            providerRuntimeError: null,
            providerRuntimeOverview: { chat: { provider: "x" } },
            scrollAreaRef: { current: null },
            scrollRef: { current: null },
            selectedCollectionId: 3,
            sendMessage: vi.fn(),
            setInput: vi.fn(),
            setSelectedCollectionId: vi.fn(),
            showStreamingCard: false,
            streamError: null,
            streamMessageId: null,
            textareaRef: { current: null },
        };
    },
}));

describe("ChatTab", () => {
    it("com collectionId controlado nao renderiza o seletor de colecao e repassa o id ao hook", () => {
        render(<ChatTab collectionId={3} compacto />, { wrapper: createTestWrapper("pt-BR") });
        expect(document.querySelector("#chat-collection-select")).toBeNull();
        expect(usado).toHaveBeenLastCalledWith({ collectionId: 3 });
    });

    it("sem collectionId mantem o seletor", () => {
        render(<ChatTab />, { wrapper: createTestWrapper("pt-BR") });
        expect(document.querySelector("#chat-collection-select")).not.toBeNull();
    });
});
