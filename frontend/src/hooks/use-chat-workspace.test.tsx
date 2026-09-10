import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const navigation = vi.hoisted(() => ({
    replace: vi.fn(),
    pathname: "/en/chat",
    searchParams: new URLSearchParams(),
}));

vi.mock("next/navigation", () => ({
    usePathname: () => navigation.pathname,
    useRouter: () => ({
        replace: navigation.replace,
    }),
    useSearchParams: () => navigation.searchParams,
}));

vi.mock("@/hooks/use-collections", () => ({
    useCollections: vi.fn(),
}));

vi.mock("@/hooks/use-provider-runtime-overview", () => ({
    useProviderRuntimeOverview: vi.fn(),
}));

vi.mock("@/lib/api/chat", () => ({
    createChatConversation: vi.fn(),
    deleteChatConversation: vi.fn(),
    fetchChatConversation: vi.fn(),
    fetchChatConversations: vi.fn(),
    streamCollectionChat: vi.fn(),
}));

import { useChatWorkspace } from "@/hooks/use-chat-workspace";
import { useCollections } from "@/hooks/use-collections";
import { useProviderRuntimeOverview } from "@/hooks/use-provider-runtime-overview";
import {
    deleteChatConversation,
    fetchChatConversation,
    fetchChatConversations,
    streamCollectionChat,
} from "@/lib/api/chat";
import { createTestWrapper } from "@/test/test-utils";

function createConversation(id: number, title: string) {
    return {
        id,
        collectionId: 7,
        title,
        lastMessageAt: `2026-03-20T12:0${id - 10}:00Z`,
        createdAt: `2026-03-20T12:0${id - 10}:00Z`,
        updatedAt: `2026-03-20T12:0${id - 10}:00Z`,
    };
}

describe("useChatWorkspace", () => {
    beforeEach(() => {
        navigation.replace.mockReset();
        navigation.pathname = "/en/chat";
        navigation.searchParams = new URLSearchParams();

        vi.mocked(useCollections).mockReturnValue({
            collections: [
                {
                    id: 7,
                    name: "Operations",
                    documentCount: 3,
                    metadata: {},
                },
            ],
            isLoading: false,
            error: null,
            mergeCollection: vi.fn(),
            reloadCollections: vi.fn(),
        });

        vi.mocked(useProviderRuntimeOverview).mockReturnValue({
            overview: {
                storage: null,
                indexing: null,
                chat: {
                    family: "LLM",
                    activation_target: "CHAT",
                    provider_type: "OPENAI",
                    public_metadata: {
                        model: "gpt-4o-mini",
                    },
                    updated_at: "2026-03-20T12:00:00Z",
                },
            },
            isLoading: false,
            error: null,
        });

        vi.mocked(fetchChatConversations).mockReset();
        vi.mocked(fetchChatConversation).mockReset();
        vi.mocked(deleteChatConversation).mockReset();
    });

    it("starts a fresh draft conversation while keeping the selected collection", async () => {
        vi.mocked(fetchChatConversations).mockResolvedValue([
            createConversation(10, "Quarterly review"),
        ]);
        vi.mocked(fetchChatConversation).mockResolvedValue({
            conversation: createConversation(10, "Quarterly review"),
            messages: [
                {
                    id: 1,
                    role: "assistant",
                    content: "Existing grounded answer",
                    sources: [],
                    createdAt: "2026-03-20T12:00:00Z",
                    updatedAt: "2026-03-20T12:00:00Z",
                },
            ],
        });

        const { result } = renderHook(() => useChatWorkspace(), { wrapper: createTestWrapper("en") });

        await waitFor(() => {
            expect(result.current.isConversationLoading).toBe(false);
        });

        act(() => {
            result.current.setInput("Draft question");
            result.current.startNewConversation();
        });

        expect(result.current.selectedCollectionId).toBe(7);
        expect(result.current.activeConversationId).toBeNull();
        expect(result.current.messages).toEqual([]);
        expect(result.current.input).toBe("");
        expect(navigation.replace).toHaveBeenCalledWith("/en/chat?collection=7", { scroll: false });
    });

    it("deletes the active conversation and activates the next saved one", async () => {
        vi.mocked(fetchChatConversations).mockResolvedValue([
            createConversation(10, "Quarterly review"),
            createConversation(11, "Risk register"),
        ]);
        vi.mocked(fetchChatConversation).mockImplementation(async (conversationId: number) => ({
            conversation: createConversation(conversationId, conversationId === 10 ? "Quarterly review" : "Risk register"),
            messages: [
                {
                    id: conversationId,
                    role: "assistant",
                    content: conversationId === 10 ? "Initial answer" : "Next answer",
                    sources: [],
                    createdAt: "2026-03-20T12:00:00Z",
                    updatedAt: "2026-03-20T12:00:00Z",
                },
            ],
        }));
        vi.mocked(deleteChatConversation).mockResolvedValue(undefined);

        const { result } = renderHook(() => useChatWorkspace(), { wrapper: createTestWrapper("en") });

        await waitFor(() => {
            expect(result.current.activeConversationId).toBe(10);
        });

        await act(async () => {
            await result.current.deleteConversation(10);
        });

        await waitFor(() => {
            expect(result.current.activeConversationId).toBe(11);
        });

        await waitFor(() => {
            expect(result.current.messages).toEqual([
                expect.objectContaining({
                    id: "11",
                    content: "Next answer",
                }),
            ]);
        });

        expect(deleteChatConversation).toHaveBeenCalledWith(10);
        expect(result.current.conversations.map((conversation) => conversation.id)).toEqual([11]);
    });

    it("carrega a conversa escolhida mesmo com outra recebendo stream", async () => {
        // Cenario: o usuario envia uma pergunta na conversa 10 e, enquanto a
        // resposta ainda esta chegando, clica na conversa 11 na barra lateral.
        //
        // O guard do efeito de carga era `if (isLoading) return`, ou seja,
        // bloqueava QUALQUER carga durante o streaming. O cabecalho passava a
        // mostrar o titulo da conversa 11 enquanto a area de mensagens seguia
        // exibindo o conteudo da 10 -- por dezenas de segundos, e com os deltas
        // da 10 sendo escritos sob o rotulo da 11. Agora o guard so protege a
        // conversa que esta de fato streamando.
        vi.mocked(fetchChatConversations).mockResolvedValue([
            createConversation(10, "Em streaming"),
            createConversation(11, "Outra conversa"),
        ]);
        vi.mocked(fetchChatConversation).mockImplementation(async (id: number) => ({
            conversation: createConversation(id, id === 10 ? "Em streaming" : "Outra conversa"),
            messages: [
                {
                    id,
                    role: "assistant" as const,
                    content: id === 10 ? "Resposta da conversa 10" : "Resposta da conversa 11",
                    sources: [],
                    createdAt: "2026-03-20T12:00:00Z",
                    updatedAt: "2026-03-20T12:00:00Z",
                },
            ],
        }));

        // Stream que NAO resolve: mantem isLoading === true durante o teste,
        // que e exatamente a condicao em que o bug se manifestava.
        let liberarStream: (() => void) | undefined;
        vi.mocked(streamCollectionChat).mockImplementation(
            () =>
                new Promise<void>((resolve) => {
                    liberarStream = resolve;
                })
        );

        const { result } = renderHook(() => useChatWorkspace(), { wrapper: createTestWrapper("en") });

        await waitFor(() => {
            expect(result.current.activeConversationId).toBe(10);
        });
        await waitFor(() => {
            expect(result.current.messages[0]?.content).toBe("Resposta da conversa 10");
        });

        // Dispara a pergunta na conversa 10 e deixa o stream pendurado.
        act(() => {
            result.current.setInput("Pergunta na conversa 10");
        });
        act(() => {
            void result.current.sendMessage();
        });

        await waitFor(() => {
            expect(result.current.isLoading).toBe(true);
        });

        // Com o stream da 10 em andamento, troca para a conversa 11.
        act(() => {
            result.current.selectConversation(11);
        });

        await waitFor(() => {
            expect(result.current.activeConversationId).toBe(11);
        });

        // O conteudo precisa acompanhar o titulo. Com o guard antigo as
        // mensagens continuariam sendo as da conversa 10.
        await waitFor(() => {
            expect(result.current.messages.some((m) => m.content === "Resposta da conversa 11")).toBe(true);
        });

        act(() => {
            liberarStream?.();
        });
    });

    it("quando o provedor falha, o card fica com as fontes e a causa", async () => {
        vi.mocked(fetchChatConversations).mockResolvedValue([createConversation(10, "Conversa")]);
        vi.mocked(fetchChatConversation).mockResolvedValue({ conversation: createConversation(10, "Conversa"), messages: [] });
        vi.mocked(streamCollectionChat).mockImplementation(async ({ onEvent }) => {
            onEvent({ type: "sources", sources: [{ id: "265:1:table-9-row-1", docName: "Matriz", page: 1, snippet: "Disciplina: SIS07302", documentId: 265 }] });
            onEvent({ type: "error", message: "Falha ao gerar resposta do chat (monthly usage limit)" });
        });

        const { result } = renderHook(() => useChatWorkspace(), { wrapper: createTestWrapper("pt-BR") });
        await waitFor(() => {
            expect(result.current.activeConversationId).toBe(10);
        });
        act(() => {
            result.current.setInput("Qual o pré-requisito de Estrutura de Dados I?");
        });
        await act(async () => {
            await result.current.sendMessage();
        });

        const resposta = result.current.messages.at(-1);
        expect(resposta?.role).toBe("assistant");
        expect(resposta?.sources).toHaveLength(1);
        expect(resposta?.content).toContain("monthly usage limit");
        expect(result.current.streamError).toContain("monthly usage limit");
    });
});
