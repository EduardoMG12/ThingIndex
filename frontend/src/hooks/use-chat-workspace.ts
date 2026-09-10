"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

import type { Source } from "@/components/chat/SourceReference";
import type { ChatWorkspaceMessage } from "@/components/chat/types";
import { useCollections } from "@/hooks/use-collections";
import { useProviderRuntimeOverview } from "@/hooks/use-provider-runtime-overview";
import {
    createChatConversation,
    deleteChatConversation,
    fetchChatConversation,
    fetchChatConversations,
    streamCollectionChat,
    type ChatConversationSummary,
    type ChatHistoryMessage,
    type ChatSource,
    type ChatStreamEvent,
    type PersistedChatMessage,
} from "@/lib/api/chat";
import { useI18n } from "@/lib/i18n/provider";

type SubQuery = {
    id: string;
    question: string;
    status: "pending" | "running" | "done";
};

const CHAT_STEPS: Array<{ id: string; questionKey: string }> = [
    { id: "collection-context", questionKey: "dashboard.chat.stepContext" },
    { id: "source-selection", questionKey: "dashboard.chat.stepSources" },
    { id: "answer-generation", questionKey: "dashboard.chat.stepAnswer" },
];

function buildInitialSteps(t: (key: string) => string): SubQuery[] {
    return CHAT_STEPS.map((step) => ({
        id: step.id,
        question: t(step.questionKey),
        status: "pending",
    }));
}

function mapStageToSteps(stage: string | undefined, t: (key: string) => string): SubQuery[] {
    const steps = buildInitialSteps(t);

    switch (stage) {
        case "retrieving_context":
            steps[0].status = "running";
            break;
        case "selecting_sources":
            steps[0].status = "done";
            steps[1].status = "running";
            break;
        case "generating_answer":
            steps[0].status = "done";
            steps[1].status = "done";
            steps[2].status = "running";
            break;
        case "completed":
            steps.forEach((step) => {
                step.status = "done";
            });
            break;
        default:
            break;
    }

    return steps;
}

function normalizeSources(sources: ChatSource[]): Source[] {
    return sources.map((source) => ({
        id: source.id,
        docName: source.docName,
        page: source.page,
        snippet: source.snippet,
        documentId: source.documentId,
        nodeId: source.nodeId,
        paragraphId: source.paragraphId,
        lines: source.lines,
    }));
}

function normalizePersistedMessages(messages: PersistedChatMessage[]): ChatWorkspaceMessage[] {
    return messages.map((message) => ({
        id: String(message.id),
        role: message.role,
        content: message.content,
        sources: normalizeSources(message.sources),
        usage: message.usage,
        createdAt: message.createdAt,
    }));
}

function buildConversationTitle(prompt: string, t: (key: string) => string) {
    const normalized = prompt.replace(/\s+/g, " ").trim();
    if (!normalized) {
        return t("dashboard.chat.newConversation");
    }

    return normalized.length > 80 ? `${normalized.slice(0, 80).trim()}...` : normalized;
}

function createDraftMessage(role: "user" | "assistant", content = ""): ChatWorkspaceMessage {
    return {
        id: crypto.randomUUID(),
        role,
        content,
        createdAt: new Date().toISOString(),
        sources: [],
    };
}

/**
 * `collectionId` controlado: quem embute o chat (a Area de trabalho) escolhe a
 * colecao; o seletor proprio e a hidratacao por `?collection` ficam para a
 * pagina /chat, que nao passa nada.
 */
export function useChatWorkspace(opcoes: { collectionId?: number | null } = {}) {
    const { t } = useI18n();
    const router = useRouter();
    const pathname = usePathname();
    const searchParams = useSearchParams();
    const { collections, isLoading: collectionsLoading } = useCollections();
    const {
        overview: providerRuntimeOverview,
        isLoading: isProviderRuntimeLoading,
        error: providerRuntimeError,
    } = useProviderRuntimeOverview();

    const [messages, setMessages] = useState<ChatWorkspaceMessage[]>([]);
    const [input, setInput] = useState("");
    const [isLoading, setIsLoading] = useState(false);
    const [activeSubQueries, setActiveSubQueries] = useState<SubQuery[]>([]);
    const [selectedCollectionId, setSelectedCollectionId] = useState<number | null>(opcoes.collectionId ?? null);
    const collectionIdControlado = opcoes.collectionId ?? null;
    useEffect(() => {
        if (collectionIdControlado !== null) {
            setSelectedCollectionId(collectionIdControlado);
        }
    }, [collectionIdControlado]);
    const [conversations, setConversations] = useState<ChatConversationSummary[]>([]);
    const [activeConversationId, setActiveConversationId] = useState<number | null>(null);
    const [isConversationsLoading, setIsConversationsLoading] = useState(false);
    const [isConversationLoading, setIsConversationLoading] = useState(false);
    const [deletingConversationId, setDeletingConversationId] = useState<number | null>(null);
    const [streamError, setStreamError] = useState<string | null>(null);
    const [showStreamingCard, setShowStreamingCard] = useState(false);

    const scrollAreaRef = useRef<HTMLDivElement>(null);
    const scrollRef = useRef<HTMLDivElement>(null);
    const textareaRef = useRef<HTMLTextAreaElement>(null);
    const abortRef = useRef<AbortController | null>(null);
    const pendingDeltaRef = useRef("");
    const streamMessageIdRef = useRef<string | null>(null);
    const flushTimerRef = useRef<number | null>(null);
    const streamingCardTimerRef = useRef<number | null>(null);
    const hasShownDeltaRef = useRef(false);
    const shouldAutoScrollRef = useRef(true);
    // Qual conversa esta recebendo o stream neste momento.
    //
    // Existe para o guard do efeito de carga ser PRECISO. Antes ele era
    // `if (isLoading) return`, o que bloqueava a carga para qualquer conversa
    // enquanto houvesse streaming: trocar de conversa no meio de uma resposta
    // deixava o cabecalho com o titulo da conversa nova e a area de mensagens
    // com o conteudo da antiga -- por dezenas de segundos, e com os deltas da
    // conversa antiga sendo escritos sob o rotulo da nova.
    //
    // O guard amplo tinha razao de existir: ao enviar a primeira mensagem de
    // uma conversa nova, `setActiveConversationId(created.id)` muda o id no
    // meio do stream, e sem guard o efeito buscaria do servidor e apagaria o
    // texto que esta chegando. Guardar o id resolve os dois casos: bloqueia so
    // a conversa que esta streamando, e libera a carga de qualquer outra.
    const streamingConversationIdRef = useRef<number | null>(null);
    const previousMessageCountRef = useRef(0);
    const hasHydratedRouteRef = useRef(false);
    const previousNewTriggerRef = useRef<string | null>(null);

    const hasCollections = collections.some((collection) => typeof collection.id === "number");
    const hasChatRuntime = Boolean(providerRuntimeOverview.chat);
    const activeCollection = collections.find((collection) => collection.id === selectedCollectionId) ?? null;
    const activeConversation = conversations.find((conversation) => conversation.id === activeConversationId) ?? null;
    const canType = hasCollections && !isLoading;
    const canSubmit = Boolean(input.trim()) && Boolean(selectedCollectionId) && hasChatRuntime && !isLoading;
    const isStreaming = isLoading && Boolean(streamMessageIdRef.current);

    const stopDeltaFlush = useCallback(() => {
        if (flushTimerRef.current !== null) {
            window.clearTimeout(flushTimerRef.current);
            flushTimerRef.current = null;
        }
    }, []);

    const flushPendingDelta = () => {
        const messageId = streamMessageIdRef.current;
        if (!messageId) {
            pendingDeltaRef.current = "";
            stopDeltaFlush();
            return;
        }

        if (!pendingDeltaRef.current) {
            stopDeltaFlush();
            return;
        }

        const chunkSize = pendingDeltaRef.current.length > 80 ? 20 : 8;
        const chunk = pendingDeltaRef.current.slice(0, chunkSize);
        pendingDeltaRef.current = pendingDeltaRef.current.slice(chunk.length);

        setMessages((current) =>
            current.map((message) =>
                message.id === messageId
                    ? { ...message, content: `${message.content}${chunk}` }
                    : message
            )
        );

        flushTimerRef.current = window.setTimeout(flushPendingDelta, 24);
    };

    const queueDelta = (messageId: string, delta: string) => {
        if (!delta) {
            return;
        }

        streamMessageIdRef.current = messageId;
        pendingDeltaRef.current += delta;

        if (flushTimerRef.current === null) {
            flushPendingDelta();
        }
    };

    const flushAllPendingDelta = () => {
        const messageId = streamMessageIdRef.current;
        if (!messageId || !pendingDeltaRef.current) {
            stopDeltaFlush();
            return;
        }

        const remainder = pendingDeltaRef.current;
        pendingDeltaRef.current = "";
        stopDeltaFlush();

        setMessages((current) =>
            current.map((message) =>
                message.id === messageId
                    ? { ...message, content: `${message.content}${remainder}` }
                    : message
            )
        );
    };

    const displayStreamingCard = () => {
        setShowStreamingCard(true);
        hasShownDeltaRef.current = false;
        if (streamingCardTimerRef.current !== null) {
            window.clearTimeout(streamingCardTimerRef.current);
        }
        streamingCardTimerRef.current = window.setTimeout(() => {
            setShowStreamingCard(false);
            streamingCardTimerRef.current = null;
        }, 900);
    };

    const hideStreamingCard = useCallback(() => {
        if (streamingCardTimerRef.current !== null) {
            window.clearTimeout(streamingCardTimerRef.current);
            streamingCardTimerRef.current = null;
        }
        setShowStreamingCard(false);
    }, []);

    const resetDraftConversation = useCallback(() => {
        if (abortRef.current) {
            abortRef.current.abort();
            abortRef.current = null;
        }

        pendingDeltaRef.current = "";
        streamMessageIdRef.current = null;
        stopDeltaFlush();
        streamingConversationIdRef.current = null;
        setActiveConversationId(null);
        setMessages([]);
        setInput("");
        setIsLoading(false);
        setActiveSubQueries([]);
        setStreamError(null);
        previousMessageCountRef.current = 0;
        hideStreamingCard();
    }, [hideStreamingCard, stopDeltaFlush]);

    useEffect(() => {
        const viewport = scrollAreaRef.current?.querySelector("[data-radix-scroll-area-viewport]");
        if (!(viewport instanceof HTMLDivElement)) {
            return;
        }

        const updateAutoScroll = () => {
            const distanceFromBottom = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
            shouldAutoScrollRef.current = distanceFromBottom < 120;
        };

        updateAutoScroll();
        viewport.addEventListener("scroll", updateAutoScroll, { passive: true });

        return () => {
            viewport.removeEventListener("scroll", updateAutoScroll);
        };
    }, []);

    useEffect(() => {
        const previousCount = previousMessageCountRef.current;
        const hasNewMessage = messages.length > previousCount;

        if (hasNewMessage || shouldAutoScrollRef.current) {
            scrollRef.current?.scrollIntoView({
                behavior: hasNewMessage ? "smooth" : "auto",
                block: "end",
            });
        }

        previousMessageCountRef.current = messages.length;
    }, [messages]);

    useEffect(() => {
        if (selectedCollectionId !== null) {
            return;
        }

        const queryCollectionId = Number(searchParams.get("collection"));
        if (Number.isFinite(queryCollectionId) && queryCollectionId > 0) {
            setSelectedCollectionId(queryCollectionId);
            hasHydratedRouteRef.current = true;
            return;
        }

        const firstCollection = collections.find((collection) => typeof collection.id === "number");
        if (firstCollection && typeof firstCollection.id === "number") {
            setSelectedCollectionId(firstCollection.id);
        }

        hasHydratedRouteRef.current = true;
    }, [collections, searchParams, selectedCollectionId]);

    useEffect(() => {
        return () => {
            abortRef.current?.abort();
            stopDeltaFlush();
            hideStreamingCard();
        };
    }, [hideStreamingCard, stopDeltaFlush]);

    useEffect(() => {
        if (!selectedCollectionId) {
            setConversations([]);
            setActiveConversationId(null);
            setMessages([]);
            setStreamError(null);
            setActiveSubQueries([]);
            previousMessageCountRef.current = 0;
            return;
        }

        const controller = new AbortController();
        setIsConversationsLoading(true);
        setConversations([]);
        setActiveConversationId(null);
        setMessages([]);
        setStreamError(null);
        setActiveSubQueries([]);
        previousMessageCountRef.current = 0;

        void fetchChatConversations(selectedCollectionId, controller.signal)
            .then((items) => {
                setConversations(items);
                setActiveConversationId((current) => {
                    if (current && items.some((conversation) => conversation.id === current)) {
                        return current;
                    }
                    return items[0]?.id ?? null;
                });
            })
            .catch((error) => {
                if (controller.signal.aborted) {
                    return;
                }
                setConversations([]);
                setActiveConversationId(null);
                setMessages([]);
                setStreamError(error instanceof Error ? error.message : t("dashboard.chat.error"));
            })
            .finally(() => {
                if (!controller.signal.aborted) {
                    setIsConversationsLoading(false);
                }
            });

        return () => {
            controller.abort();
        };
    }, [selectedCollectionId, t]);

    useEffect(() => {
        if (!hasHydratedRouteRef.current) {
            return;
        }

        const queryConversationId = Number(searchParams.get("conversation"));
        if (!Number.isFinite(queryConversationId) || queryConversationId <= 0) {
            return;
        }

        if (conversations.some((conversation) => conversation.id === queryConversationId) && activeConversationId !== queryConversationId) {
            setActiveConversationId(queryConversationId);
        }
    }, [searchParams, conversations, activeConversationId]);

    useEffect(() => {
        const newTrigger = searchParams.get("new");
        if (!newTrigger || previousNewTriggerRef.current === newTrigger) {
            return;
        }

        previousNewTriggerRef.current = newTrigger;
        resetDraftConversation();
    }, [searchParams, resetDraftConversation]);

    useEffect(() => {
        if (!activeConversationId) {
            setMessages([]);
            setStreamError(null);
            setActiveSubQueries([]);
            previousMessageCountRef.current = 0;
            return;
        }

        // Bloqueia apenas a conversa que esta recebendo o stream: recarregar
        // do servidor no meio dele apagaria os deltas que ja chegaram. Trocar
        // para OUTRA conversa carrega normalmente.
        if (isLoading && streamingConversationIdRef.current === activeConversationId) {
            return;
        }

        const controller = new AbortController();
        setIsConversationLoading(true);
        setStreamError(null);
        setActiveSubQueries([]);

        void fetchChatConversation(activeConversationId, controller.signal)
            .then((detail) => {
                setMessages(normalizePersistedMessages(detail.messages));
            })
            .catch((error) => {
                if (controller.signal.aborted) {
                    return;
                }
                setMessages([]);
                setStreamError(error instanceof Error ? error.message : t("dashboard.chat.error"));
            })
            .finally(() => {
                if (!controller.signal.aborted) {
                    setIsConversationLoading(false);
                }
            });

        return () => {
            controller.abort();
        };
    }, [activeConversationId, isLoading, t]);

    const handleStreamEvent = (assistantMessageId: string) => (event: ChatStreamEvent) => {
        switch (event.type) {
            case "stage":
                setActiveSubQueries(mapStageToSteps(event.stage, t));
                break;
            case "sources":
                setMessages((current) =>
                    current.map((message) =>
                        message.id === assistantMessageId
                            ? { ...message, sources: normalizeSources(event.sources) }
                            : message
                    )
                );
                break;
            case "delta":
                if (!hasShownDeltaRef.current) {
                    hasShownDeltaRef.current = true;
                    hideStreamingCard();
                }
                queueDelta(assistantMessageId, event.delta);
                break;
            case "done":
                flushAllPendingDelta();
                hideStreamingCard();
                setActiveSubQueries(mapStageToSteps(event.stage, t));
                if (event.usage) {
                    const usage = event.usage;
                    setMessages((current) =>
                        current.map((message) => (message.id === assistantMessageId ? { ...message, usage } : message))
                    );
                }
                break;
            case "error":
                flushAllPendingDelta();
                hideStreamingCard();
                setStreamError(event.message);
                // As fontes ja chegaram antes do modelo falhar: o card fica
                // com elas e com a causa, em vez de um balao vazio. E a prova
                // de que a recuperacao funcionou mesmo sem resposta.
                setMessages((current) =>
                    current.map((message) =>
                        message.id === assistantMessageId && !message.content.trim()
                            ? { ...message, content: event.message }
                            : message
                    )
                );
                break;
            default:
                break;
        }
    };

    const ensureConversation = async (prompt: string): Promise<ChatConversationSummary> => {
        if (activeConversationId) {
            const existing = conversations.find((conversation) => conversation.id === activeConversationId);
            if (existing) {
                return existing;
            }
        }

        if (!selectedCollectionId) {
            throw new Error(t("dashboard.chat.error"));
        }

        const created = await createChatConversation(
            selectedCollectionId,
            buildConversationTitle(prompt, t)
        );
        setConversations((current) => [created, ...current]);
        setActiveConversationId(created.id);
        return created;
    };

    const sendMessage = async () => {
        const prompt = input.trim();
        if (!prompt || isLoading || !selectedCollectionId) {
            return;
        }

        if (!hasChatRuntime) {
            setStreamError(t("providerRuntime.chat.missing"));
            return;
        }

        setIsLoading(true);
        streamingConversationIdRef.current = activeConversationId;
        setStreamError(null);
        setActiveSubQueries(buildInitialSteps(t));
        displayStreamingCard();

        let conversation: ChatConversationSummary;
        try {
            conversation = await ensureConversation(prompt);
        } catch (error) {
            setIsLoading(false);
            streamingConversationIdRef.current = null;
            setActiveSubQueries([]);
            setStreamError(error instanceof Error ? error.message : t("dashboard.chat.error"));
            return;
        }

        const userMessage = createDraftMessage("user", prompt);
        const assistantMessage = createDraftMessage("assistant");

        // ensureConversation pode ter CRIADO a conversa agora (primeira
        // mensagem), trocando o activeConversationId. Reaponta a ref para o id
        // definitivo, senao o guard protegeria a conversa errada.
        streamingConversationIdRef.current = conversation.id;

        pendingDeltaRef.current = "";
        streamMessageIdRef.current = assistantMessage.id;
        stopDeltaFlush();

        const history: ChatHistoryMessage[] = messages
            .filter((message) => message.content.trim())
            .slice(-6)
            .map((message) => ({
                role: message.role,
                content: message.content,
            }));

        setMessages((current) => [...current, userMessage, assistantMessage]);
        setInput("");

        const controller = new AbortController();
        abortRef.current = controller;

        try {
            await streamCollectionChat({
                conversationId: conversation.id,
                collectionId: selectedCollectionId,
                prompt,
                history,
                signal: controller.signal,
                onEvent: handleStreamEvent(assistantMessage.id),
            });

            const timestamp = new Date().toISOString();
            setConversations((current) => {
                const next = current.some((item) => item.id === conversation.id)
                    ? current.map((item) =>
                        item.id === conversation.id
                            ? { ...item, title: item.title || buildConversationTitle(prompt, t), lastMessageAt: timestamp, updatedAt: timestamp }
                            : item
                    )
                    : [{ ...conversation, lastMessageAt: timestamp, updatedAt: timestamp }, ...current];

                return next.sort((left, right) => {
                    const leftValue = left.lastMessageAt || left.updatedAt;
                    const rightValue = right.lastMessageAt || right.updatedAt;
                    return rightValue.localeCompare(leftValue);
                });
            });
        } catch (error) {
            flushAllPendingDelta();
            if (controller.signal.aborted) {
                setMessages((current) =>
                    current.filter((message) => !(message.id === assistantMessage.id && !message.content.trim()))
                );
                return;
            }

            const message = error instanceof Error ? error.message : t("dashboard.chat.error");
            setStreamError(message);
            setMessages((current) =>
                current.map((item) =>
                    item.id === assistantMessage.id && !item.content.trim()
                        ? { ...item, content: t("dashboard.chat.fallbackError") }
                        : item
                )
            );
        } finally {
            flushAllPendingDelta();
            streamingConversationIdRef.current = null;
            if (!controller.signal.aborted) {
                setIsLoading(false);
                setActiveSubQueries([]);
            }
            if (abortRef.current === controller) {
                abortRef.current = null;
            }
        }
    };

    const cancelStreaming = () => {
        abortRef.current?.abort();
        abortRef.current = null;
        flushAllPendingDelta();
        streamingConversationIdRef.current = null;
        setIsLoading(false);
        setActiveSubQueries([]);
        setStreamError(t("dashboard.chat.cancelled"));
        hideStreamingCard();
    };

    const startNewConversation = () => {
        resetDraftConversation();
    };

    const selectConversation = (conversationId: number) => {
        if (conversationId === activeConversationId) {
            return;
        }

        setStreamError(null);
        setActiveSubQueries([]);
        setActiveConversationId(conversationId);
    };

    const deleteConversation = async (conversationId: number) => {
        if (deletingConversationId !== null) {
            return;
        }

        setDeletingConversationId(conversationId);

        try {
            await deleteChatConversation(conversationId);

            const nextConversations = conversations.filter((conversation) => conversation.id !== conversationId);
            setConversations(nextConversations);

            if (activeConversationId === conversationId) {
                if (abortRef.current) {
                    abortRef.current.abort();
                    abortRef.current = null;
                }

                pendingDeltaRef.current = "";
                streamMessageIdRef.current = null;
                stopDeltaFlush();
                streamingConversationIdRef.current = null;
                setIsLoading(false);
                setMessages([]);
                setInput("");
                setStreamError(null);
                setActiveSubQueries([]);
                previousMessageCountRef.current = 0;
                hideStreamingCard();
                setActiveConversationId(nextConversations[0]?.id ?? null);
            }

            window.dispatchEvent(new CustomEvent("thingindex:chat-conversation-deleted", {
                detail: {
                    id: conversationId,
                },
            }));
        } catch (error) {
            const message = error instanceof Error ? error.message : t("apiErrors.chat.delete");
            setStreamError(message);
            throw error;
        } finally {
            setDeletingConversationId(null);
        }
    };

    useEffect(() => {
        window.dispatchEvent(new CustomEvent("thingindex:chat-sidebar-sync", {
            detail: {
                selectedCollectionId,
                activeConversationId,
                conversations,
            },
        }));
    }, [selectedCollectionId, activeConversationId, conversations]);

    useEffect(() => {
        const handleConversationUpdated = (event: Event) => {
            const detail = (event as CustomEvent<ChatConversationSummary>).detail;
            if (!detail) {
                return;
            }

            setConversations((current) =>
                current.map((conversation) => (
                    conversation.id === detail.id ? detail : conversation
                ))
            );
        };

        const handleConversationDeleted = (event: Event) => {
            const detail = (event as CustomEvent<{ id: number }>).detail;
            if (!detail) {
                return;
            }

            setConversations((current) => current.filter((conversation) => conversation.id !== detail.id));
            setActiveConversationId((current) => (current === detail.id ? null : current));
            setMessages((current) => (activeConversationId === detail.id ? [] : current));
            setInput((current) => (activeConversationId === detail.id ? "" : current));
            setStreamError(null);
            setActiveSubQueries([]);
        };

        window.addEventListener("thingindex:chat-conversation-updated", handleConversationUpdated as EventListener);
        window.addEventListener("thingindex:chat-conversation-deleted", handleConversationDeleted as EventListener);

        return () => {
            window.removeEventListener("thingindex:chat-conversation-updated", handleConversationUpdated as EventListener);
            window.removeEventListener("thingindex:chat-conversation-deleted", handleConversationDeleted as EventListener);
        };
    }, [activeConversationId]);

    useEffect(() => {
        if (!hasHydratedRouteRef.current) {
            return;
        }

        // Preserva o que nao e nosso (a Area de trabalho usa ?doc=).
        const params = new URLSearchParams(searchParams.toString());
        params.delete("collection");
        params.delete("conversation");
        params.delete("new");
        if (selectedCollectionId) {
            params.set("collection", String(selectedCollectionId));
        }
        if (activeConversationId) {
            params.set("conversation", String(activeConversationId));
        }

        const next = params.toString();
        const current = searchParams.toString();
        if (next !== current) {
            router.replace(next ? `${pathname}?${next}` : pathname, { scroll: false });
        }
        // `searchParams` estava faltando aqui e era lido de um closure velho, o
        // que podia disparar um router.replace redundante para uma URL que ja
        // era a atual.
        //
        // Incluir a dependencia NAO gera loop: o replace muda searchParams e
        // reexecuta o efeito, mas na segunda passada `next === current` e o
        // guard impede a segunda escrita. O efeito converge.
    }, [selectedCollectionId, activeConversationId, pathname, router, searchParams]);

    return {
        activeCollection,
        activeConversation,
        activeConversationId,
        activeSubQueries,
        canSubmit,
        canType,
        cancelStreaming,
        collections,
        collectionsLoading,
        conversations,
        deleteConversation,
        deletingConversationId,
        hasChatRuntime,
        hasCollections,
        input,
        isConversationLoading,
        isConversationsLoading,
        isLoading,
        isProviderRuntimeLoading,
        isStreaming,
        messages,
        providerRuntimeError,
        providerRuntimeOverview,
        scrollAreaRef,
        scrollRef,
        selectConversation,
        selectedCollectionId,
        sendMessage,
        setInput,
        setSelectedCollectionId,
        showStreamingCard,
        startNewConversation,
        streamError,
        streamMessageId: streamMessageIdRef.current,
        textareaRef,
    };
}
