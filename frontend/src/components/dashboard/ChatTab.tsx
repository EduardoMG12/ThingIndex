"use client";

import { type ReactNode, useEffect, useRef, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Send, Bot, User, Loader2, Sparkles, AlertCircle, Hash, FolderOpen, Square, Trash2, MessageSquarePlus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import SourceReference, { type Source } from "@/components/chat/SourceReference";
import SubQueryProgress, { type SubQuery } from "@/components/chat/SubQueryProgress";
import { useCollections } from "@/hooks/use-collections";
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
import { cn } from "@/lib/utils";

type Message = {
    id: string;
    role: "user" | "assistant";
    content: string;
    sources?: Source[];
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
    }));
}

function normalizePersistedMessages(messages: PersistedChatMessage[]): Message[] {
    return messages.map((message) => ({
        id: String(message.id),
        role: message.role,
        content: message.content,
        sources: normalizeSources(message.sources),
    }));
}

function buildConversationTitle(prompt: string, t: (key: string) => string) {
    const normalized = prompt.replace(/\s+/g, " ").trim();
    if (!normalized) {
        return t("dashboard.chat.newConversation");
    }

    return normalized.length > 80 ? `${normalized.slice(0, 80).trim()}...` : normalized;
}

function renderFormattedText(content: string): ReactNode {
    const normalizedContent = content.replace(/\r\n/g, "\n").trim();
    if (!normalizedContent) {
        return null;
    }

    const renderInline = (text: string, keyPrefix: string) => text
        .split(/(\*\*[^*]+\*\*)/g)
        .filter(Boolean)
        .map((segment, index) => {
            if (segment.startsWith("**") && segment.endsWith("**") && segment.length > 4) {
                return <strong key={`${keyPrefix}-strong-${index}`} className="font-semibold text-foreground">{segment.slice(2, -2)}</strong>;
            }
            return <span key={`${keyPrefix}-text-${index}`}>{segment}</span>;
        });

    return normalizedContent.split(/\n{2,}/).map((paragraph, paragraphIndex) => {
        const lines = paragraph.split("\n").map((line) => line.trimEnd());
        const bulletLines = lines.filter((line) => /^[-*]\s+/.test(line));

        if (bulletLines.length === lines.length && bulletLines.length > 0) {
            return (
                <ul key={`paragraph-${paragraphIndex}`} className="list-disc space-y-2 pl-5 leading-7">
                    {bulletLines.map((line, lineIndex) => (
                        <li key={`paragraph-${paragraphIndex}-item-${lineIndex}`}>
                            {renderInline(line.replace(/^[-*]\s+/, ""), `paragraph-${paragraphIndex}-item-${lineIndex}`)}
                        </li>
                    ))}
                </ul>
            );
        }

        return (
            <p key={`paragraph-${paragraphIndex}`} className="leading-7">
                {lines.map((line, lineIndex) => (
                    <span key={`paragraph-${paragraphIndex}-line-${lineIndex}`}>
                        {lineIndex > 0 && <br />}
                        {renderInline(line, `paragraph-${paragraphIndex}-line-${lineIndex}`)}
                    </span>
                ))}
            </p>
        );
    });
}

function StreamingPlaceholder({ label }: { label: string }) {
    return (
        <div className="flex items-center gap-3 text-sm text-muted-foreground">
            <div className="flex items-center gap-1.5">
                <span className="h-2 w-2 animate-pulse rounded-full bg-primary/85 [animation-delay:0ms]" />
                <span className="h-2 w-2 animate-pulse rounded-full bg-primary/65 [animation-delay:150ms]" />
                <span className="h-2 w-2 animate-pulse rounded-full bg-primary/45 [animation-delay:300ms]" />
            </div>
            <span className="font-medium text-foreground/80">{label}</span>
        </div>
    );
}

function StreamingStatus({ deltaCount }: { deltaCount: number }) {
    if (deltaCount === 0) {
        return null;
    }

    return (
        <p className="text-[10px] uppercase tracking-[0.3em] text-muted-foreground">
            {`Eventos stream: ${deltaCount}`}
        </p>
    );
}

function StreamingCard() {
    return (
        <motion.div
            key="streaming-card"
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -6 }}
            className="rounded-2xl border border-border bg-card p-4 shadow-sm"
        >
            <div className="space-y-3 animate-pulse">
                <div className="h-2.5 w-32 rounded-full bg-muted" />
                <div className="h-2 w-40 rounded-full bg-muted/70" />
                <div className="h-2 w-full rounded-full bg-muted/60" />
            </div>
        </motion.div>
    );
}

const ChatTab = () => {
    const { t } = useI18n();
    const { collections, isLoading: collectionsLoading } = useCollections();
    const [messages, setMessages] = useState<Message[]>([]);
    const [input, setInput] = useState("");
    const [isLoading, setIsLoading] = useState(false);
    const [activeSubQueries, setActiveSubQueries] = useState<SubQuery[]>([]);
    const [selectedCollectionId, setSelectedCollectionId] = useState<number | null>(null);
    const [conversations, setConversations] = useState<ChatConversationSummary[]>([]);
    const [activeConversationId, setActiveConversationId] = useState<number | null>(null);
    const [isConversationsLoading, setIsConversationsLoading] = useState(false);
    const [isConversationLoading, setIsConversationLoading] = useState(false);
    const [streamError, setStreamError] = useState<string | null>(null);
    const [deltaCount, setDeltaCount] = useState(0);
    const [showStreamingCard, setShowStreamingCard] = useState(false);
    const scrollRef = useRef<HTMLDivElement>(null);
    const abortRef = useRef<AbortController | null>(null);
    const pendingDeltaRef = useRef("");
    const streamMessageIdRef = useRef<string | null>(null);
    const flushTimerRef = useRef<number | null>(null);
    const streamingCardTimerRef = useRef<number | null>(null);
    const hasShownDeltaRef = useRef(false);

    const stopDeltaFlush = () => {
        if (flushTimerRef.current !== null) {
            window.clearTimeout(flushTimerRef.current);
            flushTimerRef.current = null;
        }
    };

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
        setDeltaCount((current) => current + 1);

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
        setDeltaCount(0);
    };

    const hideStreamingCard = () => {
        if (streamingCardTimerRef.current !== null) {
            window.clearTimeout(streamingCardTimerRef.current);
            streamingCardTimerRef.current = null;
        }
        setShowStreamingCard(false);
    };

    useEffect(() => {
        scrollRef.current?.scrollIntoView({ behavior: "smooth" });
    }, [messages, activeSubQueries, isLoading]);

    useEffect(() => {
        if (selectedCollectionId !== null) {
            return;
        }

        const firstCollection = collections.find((collection) => typeof collection.id === "number");
        if (firstCollection && typeof firstCollection.id === "number") {
            setSelectedCollectionId(firstCollection.id);
        }
    }, [collections, selectedCollectionId]);

    useEffect(() => {
        return () => {
            abortRef.current?.abort();
            stopDeltaFlush();
            hideStreamingCard();
        };
    }, []);

    useEffect(() => {
        if (!selectedCollectionId) {
            setConversations([]);
            setActiveConversationId(null);
            setMessages([]);
            setStreamError(null);
            setActiveSubQueries([]);
            setDeltaCount(0);
            return;
        }

        const controller = new AbortController();
        setIsConversationsLoading(true);
        setConversations([]);
        setActiveConversationId(null);
        setMessages([]);
        setStreamError(null);
        setActiveSubQueries([]);

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
        if (!activeConversationId) {
            setMessages([]);
            setStreamError(null);
            setActiveSubQueries([]);
            return;
        }

        if (isLoading) {
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
                break;
            case "error":
                flushAllPendingDelta();
                hideStreamingCard();
                setStreamError(event.message);
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

    const handleSend = async () => {
        const prompt = input.trim();
        if (!prompt || isLoading || !selectedCollectionId) {
            return;
        }

        setIsLoading(true);
        setStreamError(null);
        setActiveSubQueries(buildInitialSteps(t));
        displayStreamingCard();

        let conversation: ChatConversationSummary;
        try {
            conversation = await ensureConversation(prompt);
        } catch (error) {
            setIsLoading(false);
            setActiveSubQueries([]);
            setStreamError(error instanceof Error ? error.message : t("dashboard.chat.error"));
            return;
        }

        const userMessage: Message = {
            id: crypto.randomUUID(),
            role: "user",
            content: prompt,
        };
        const assistantMessageId = crypto.randomUUID();
        const assistantMessage: Message = {
            id: assistantMessageId,
            role: "assistant",
            content: "",
            sources: [],
        };

        pendingDeltaRef.current = "";
        streamMessageIdRef.current = assistantMessageId;
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
                onEvent: handleStreamEvent(assistantMessageId),
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
                    current.filter((message) => !(message.id === assistantMessageId && !message.content.trim()))
                );
                return;
            }

            const message =
                error instanceof Error ? error.message : t("dashboard.chat.error");
            setStreamError(message);
            setMessages((current) =>
                current.map((item) =>
                    item.id === assistantMessageId && !item.content.trim()
                        ? { ...item, content: t("dashboard.chat.fallbackError") }
                        : item
                )
            );
        } finally {
            flushAllPendingDelta();
            if (!controller.signal.aborted) {
                setIsLoading(false);
                setActiveSubQueries([]);
            }
            if (abortRef.current === controller) {
                abortRef.current = null;
            }
        }
    };

    const handleCancel = () => {
        abortRef.current?.abort();
        abortRef.current = null;
        flushAllPendingDelta();
        setIsLoading(false);
        setActiveSubQueries([]);
        setStreamError(t("dashboard.chat.cancelled"));
        hideStreamingCard();
        setDeltaCount(0);
    };

    const handleNewConversation = () => {
        if (abortRef.current) {
            abortRef.current.abort();
            abortRef.current = null;
        }

        pendingDeltaRef.current = "";
        streamMessageIdRef.current = null;
        stopDeltaFlush();
        setActiveConversationId(null);
        setMessages([]);
        setInput("");
        setIsLoading(false);
        setActiveSubQueries([]);
        setStreamError(null);
        hideStreamingCard();
        setDeltaCount(0);
    };

    const handleDeleteConversation = async () => {
        if (!activeConversationId || isLoading) {
            handleNewConversation();
            return;
        }

        try {
            const nextConversations = conversations.filter((conversation) => conversation.id !== activeConversationId);
            await deleteChatConversation(activeConversationId);
            setConversations(nextConversations);
            setActiveConversationId(nextConversations[0]?.id ?? null);
            setMessages([]);
            setInput("");
            setStreamError(null);
            setActiveSubQueries([]);
        } catch (error) {
            setStreamError(error instanceof Error ? error.message : t("dashboard.chat.error"));
        }
    };

    const hasCollections = collections.some((collection) => typeof collection.id === "number");

    return (
        <div className="surface-elevated flex h-[600px] flex-col overflow-hidden rounded-lg border border-border shadow-lg">
            <div className="border-b border-border bg-muted/20 px-4 py-3">
                <div className="flex flex-col gap-3">
                    <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                        <div>
                            <p className="text-[10px] font-bold uppercase tracking-[0.25em] text-primary">
                                {t("dashboard.chat.collectionLabel")}
                            </p>
                            <p className="mt-1 text-xs text-muted-foreground">
                                {t("dashboard.chat.collectionHint")}
                            </p>
                        </div>

                        <div className="flex flex-wrap items-center gap-2">
                            <label className="flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2">
                                <FolderOpen className="h-3.5 w-3.5 text-primary" />
                                <select
                                    value={selectedCollectionId ?? ""}
                                    onChange={(event) => {
                                        const value = Number(event.target.value);
                                        setSelectedCollectionId(Number.isFinite(value) && value > 0 ? value : null);
                                    }}
                                    disabled={!hasCollections || collectionsLoading || isLoading}
                                    className="min-w-52 bg-transparent text-xs font-mono outline-none"
                                >
                                    {!hasCollections && (
                                        <option value="">{t("dashboard.chat.noCollections")}</option>
                                    )}
                                    {collections
                                        .filter((collection) => typeof collection.id === "number")
                                        .map((collection) => (
                                            <option key={String(collection.id)} value={collection.id}>
                                                {collection.name}
                                            </option>
                                        ))}
                                </select>
                                {collectionsLoading && <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />}
                            </label>

                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={handleNewConversation}
                                disabled={!selectedCollectionId || isLoading}
                                className="gap-2"
                            >
                                <MessageSquarePlus className="h-3.5 w-3.5" />
                                {t("dashboard.chat.newConversation")}
                            </Button>

                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() => void handleDeleteConversation()}
                                disabled={!activeConversationId}
                                className="gap-2"
                            >
                                <Trash2 className="h-3.5 w-3.5" />
                                {t("dashboard.chat.clear")}
                            </Button>
                        </div>
                    </div>

                    {selectedCollectionId && (
                        <div className="flex flex-wrap items-center gap-2">
                            <label className="flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2">
                                <Hash className="h-3.5 w-3.5 text-primary" />
                                <select
                                    value={activeConversationId ?? ""}
                                    onChange={(event) => {
                                        const value = Number(event.target.value);
                                        setActiveConversationId(Number.isFinite(value) && value > 0 ? value : null);
                                    }}
                                    disabled={isLoading || isConversationsLoading || conversations.length === 0}
                                    className="min-w-60 bg-transparent text-xs font-mono outline-none"
                                >
                                    {conversations.length === 0 && (
                                        <option value="">{t("dashboard.chat.noConversations")}</option>
                                    )}
                                    {conversations.map((conversation) => (
                                        <option key={conversation.id} value={conversation.id}>
                                            {conversation.title}
                                        </option>
                                    ))}
                                </select>
                                {(isConversationsLoading || isConversationLoading) && (
                                    <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />
                                )}
                            </label>
                            <p className="text-[10px] uppercase tracking-widest text-muted-foreground">
                                {t("dashboard.chat.conversationsLabel")}
                            </p>
                        </div>
                    )}
                </div>
            </div>

            <ScrollArea className="flex-1 bg-background/30 p-6">
                <div className="space-y-6">
                    {messages.length === 0 && !isConversationLoading && (
                        <div className="flex h-96 flex-col items-center justify-center text-center opacity-40">
                            <Sparkles className="mb-4 h-12 w-12 text-primary" />
                            <h3 className="text-sm font-semibold">{t("dashboard.chat.emptyTitle")}</h3>
                            <p className="mt-2 max-w-xs text-xs leading-relaxed">
                                {hasCollections ? t("dashboard.chat.emptyDescription") : t("dashboard.chat.emptyNoCollection")}
                            </p>
                        </div>
                    )}

                    {isConversationLoading && (
                        <div className="flex h-96 items-center justify-center gap-3 text-sm text-muted-foreground">
                            <Loader2 className="h-4 w-4 animate-spin" />
                            {t("dashboard.chat.loadingConversation")}
                        </div>
                    )}

                    <AnimatePresence>
                        {showStreamingCard && <StreamingCard />}
                        {messages.map((msg) => (
                            <motion.div
                                key={msg.id}
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                className={cn("flex gap-4", msg.role === "user" ? "flex-row-reverse" : "flex-row")}
                            >
                                <div
                                    className={cn(
                                        "flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border",
                                        msg.role === "user"
                                            ? "border-primary bg-primary text-primary-foreground"
                                            : "border-border bg-card text-primary"
                                    )}
                                >
                                    {msg.role === "user" ? <User className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
                                </div>

                                <div
                                    className={cn(
                                        "max-w-[85%] space-y-2",
                                        msg.role === "user" ? "items-end text-right" : "items-start text-left"
                                    )}
                                >
                                    <div
                                        className={cn(
                                            "rounded-2xl px-4 py-3 text-sm shadow-sm transition-all break-words",
                                            msg.role === "user"
                                                ? "rounded-tr-none bg-primary text-primary-foreground whitespace-pre-wrap leading-relaxed"
                                                : cn(
                                                    "rounded-tl-none border border-border bg-card text-foreground",
                                                    isLoading && !msg.content.trim() && msg.id === streamMessageIdRef.current && "min-w-56"
                                                )
                                        )}
                                    >
                                        {msg.role === "assistant" ? (
                                            <div className="space-y-3 text-[15px] leading-7 text-foreground/95">
                                                {msg.content.trim() ? (
                                                    <>
                                                        {renderFormattedText(msg.content)}
                                                        {isLoading && msg.id === streamMessageIdRef.current && (
                                                            <span className="inline-block h-5 w-[2px] translate-y-1 animate-pulse rounded-full bg-primary/70 align-bottom" />
                                                        )}
                                                    </>
                                                ) : (
                                                    isLoading && msg.id === streamMessageIdRef.current
                                                        ? <StreamingPlaceholder label={t("dashboard.chat.streaming")} />
                                                        : null
                                                )}
                                            </div>
                                        ) : (
                                            msg.content
                                        )}
                                    </div>

                                    {msg.role === "assistant" && isLoading && msg.id === streamMessageIdRef.current && (
                                        <StreamingStatus deltaCount={deltaCount} />
                                    )}

                                    {msg.sources && msg.sources.length > 0 && (
                                        <div className="flex flex-wrap gap-2 pt-1">
                                            {msg.sources.map((source) => (
                                                <SourceReference key={source.id} source={source} />
                                            ))}
                                        </div>
                                    )}
                                </div>
                            </motion.div>
                        ))}
                    </AnimatePresence>

                    {activeSubQueries.length > 0 && (
                        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
                            <SubQueryProgress queries={activeSubQueries} />
                        </motion.div>
                    )}

                    {streamError && (
                        <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive">
                            {streamError}
                        </div>
                    )}

                    <div ref={scrollRef} />
                </div>
            </ScrollArea>

            <div className="border-t border-border bg-muted/20 p-4">
                <div className="relative flex items-end gap-2 pr-12">
                    <Textarea
                        placeholder={hasCollections ? t("dashboard.chat.placeholder") : t("dashboard.chat.disabledPlaceholder")}
                        value={input}
                        onChange={(e) => setInput(e.target.value)}
                        onKeyDown={(e) => {
                            if (e.key === "Enter" && !e.shiftKey) {
                                e.preventDefault();
                                void handleSend();
                            }
                        }}
                        disabled={!hasCollections || isLoading}
                        className="min-h-[44px] max-h-32 resize-none bg-background py-3 text-sm focus:ring-1 focus:ring-primary/30"
                        rows={1}
                    />
                    <Button
                        onClick={() => void handleSend()}
                        disabled={isLoading || !input.trim() || !selectedCollectionId}
                        size="icon"
                        className="absolute bottom-0.5 right-0 h-10 w-10 shadow-gold transition-transform active:scale-95"
                    >
                        {isLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
                    </Button>
                </div>
                <div className="mt-2 flex items-center justify-between">
                    <p className="flex items-center gap-1.5 font-mono text-[10px] uppercase text-muted-foreground">
                        <AlertCircle className="h-3 w-3" /> {t("dashboard.chat.hallucination")}
                    </p>
                    <div className="flex items-center gap-2">
                        {isLoading && (
                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={handleCancel}
                                className="gap-2 px-2.5 py-1 text-[10px] uppercase tracking-widest"
                            >
                                <Square className="h-3 w-3" />
                                {t("dashboard.chat.cancel")}
                            </Button>
                        )}
                        <p className="font-mono text-[9px] uppercase tracking-widest text-muted-foreground">
                            {hasCollections ? t("dashboard.chat.newLineHint") : t("dashboard.chat.createCollectionHint")}
                        </p>
                    </div>
                </div>
                {selectedCollectionId && (
                    <div className="mt-3 flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <Hash className="h-3 w-3 text-primary" />
                        {t("dashboard.chat.activeCollection")}{" "}
                        <span className="font-mono text-foreground">
                            {collections.find((collection) => collection.id === selectedCollectionId)?.name || selectedCollectionId}
                        </span>
                    </div>
                )}
            </div>
        </div>
    );
};

export default ChatTab;
