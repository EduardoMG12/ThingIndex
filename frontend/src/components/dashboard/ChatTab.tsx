"use client";

import { useEffect, useRef, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Send, Bot, User, Loader2, Sparkles, AlertCircle, Hash, FolderOpen, Square, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import SourceReference, { type Source } from "@/components/chat/SourceReference";
import SubQueryProgress, { type SubQuery } from "@/components/chat/SubQueryProgress";
import { useCollections } from "@/hooks/use-collections";
import { streamCollectionChat, type ChatHistoryMessage, type ChatSource, type ChatStreamEvent } from "@/lib/api/chat";
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

function getChatStorageKey(collectionId: number | null) {
    return collectionId ? `thingindex:chat:${collectionId}` : "";
}

const ChatTab = () => {
    const { t } = useI18n();
    const { collections, isLoading: collectionsLoading } = useCollections();
    const [messages, setMessages] = useState<Message[]>([]);
    const [input, setInput] = useState("");
    const [isLoading, setIsLoading] = useState(false);
    const [activeSubQueries, setActiveSubQueries] = useState<SubQuery[]>([]);
    const [selectedCollectionId, setSelectedCollectionId] = useState<number | null>(null);
    const [streamError, setStreamError] = useState<string | null>(null);
    const scrollRef = useRef<HTMLDivElement>(null);
    const abortRef = useRef<AbortController | null>(null);

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
        };
    }, []);

    useEffect(() => {
        const storageKey = getChatStorageKey(selectedCollectionId);
        if (!storageKey || typeof window === "undefined") {
            setMessages([]);
            setStreamError(null);
            setActiveSubQueries([]);
            return;
        }

        try {
            const rawValue = window.sessionStorage.getItem(storageKey);
            if (!rawValue) {
                setMessages([]);
                setStreamError(null);
                setActiveSubQueries([]);
                return;
            }

            const parsed = JSON.parse(rawValue) as { messages?: Message[] };
            setMessages(Array.isArray(parsed.messages) ? parsed.messages : []);
            setStreamError(null);
            setActiveSubQueries([]);
        } catch {
            setMessages([]);
            setStreamError(null);
            setActiveSubQueries([]);
        }
    }, [selectedCollectionId]);

    useEffect(() => {
        const storageKey = getChatStorageKey(selectedCollectionId);
        if (!storageKey || typeof window === "undefined") {
            return;
        }

        window.sessionStorage.setItem(storageKey, JSON.stringify({ messages }));
    }, [messages, selectedCollectionId]);

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
                setMessages((current) =>
                    current.map((message) =>
                        message.id === assistantMessageId
                            ? { ...message, content: `${message.content}${event.delta}` }
                            : message
                    )
                );
                break;
            case "done":
                setActiveSubQueries(mapStageToSteps(event.stage, t));
                break;
            case "error":
                setStreamError(event.message);
                break;
            default:
                break;
        }
    };

    const handleSend = async () => {
        const prompt = input.trim();
        if (!prompt || isLoading || !selectedCollectionId) {
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

        const history: ChatHistoryMessage[] = messages
            .filter((message) => message.content.trim())
            .slice(-6)
            .map((message) => ({
                role: message.role,
                content: message.content,
            }));

        setMessages((current) => [...current, userMessage, assistantMessage]);
        setInput("");
        setIsLoading(true);
        setStreamError(null);
        setActiveSubQueries(buildInitialSteps(t));

        const controller = new AbortController();
        abortRef.current = controller;

        try {
            await streamCollectionChat({
                collectionId: selectedCollectionId,
                prompt,
                history,
                signal: controller.signal,
                onEvent: handleStreamEvent(assistantMessageId),
            });
        } catch (error) {
            if (controller.signal.aborted) {
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
        setIsLoading(false);
        setActiveSubQueries([]);
        setStreamError(t("dashboard.chat.cancelled"));
    };

    const handleClear = () => {
        if (abortRef.current) {
            abortRef.current.abort();
            abortRef.current = null;
        }
        setMessages([]);
        setInput("");
        setIsLoading(false);
        setActiveSubQueries([]);
        setStreamError(null);

        const storageKey = getChatStorageKey(selectedCollectionId);
        if (storageKey && typeof window !== "undefined") {
            window.sessionStorage.removeItem(storageKey);
        }
    };

    const hasCollections = collections.some((collection) => typeof collection.id === "number");

    return (
        <div className="surface-elevated flex h-[600px] flex-col overflow-hidden rounded-lg border border-border shadow-lg">
            <div className="border-b border-border bg-muted/20 px-4 py-3">
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
                            onClick={handleClear}
                            disabled={messages.length === 0 && !streamError}
                            className="gap-2"
                        >
                            <Trash2 className="h-3.5 w-3.5" />
                            {t("dashboard.chat.clear")}
                        </Button>
                    </div>
                </div>
            </div>

            <ScrollArea className="flex-1 bg-background/30 p-6">
                <div className="space-y-6">
                    {messages.length === 0 && (
                        <div className="flex h-96 flex-col items-center justify-center text-center opacity-40">
                            <Sparkles className="mb-4 h-12 w-12 text-primary" />
                            <h3 className="text-sm font-semibold">{t("dashboard.chat.emptyTitle")}</h3>
                            <p className="mt-2 max-w-xs text-xs leading-relaxed">
                                {hasCollections ? t("dashboard.chat.emptyDescription") : t("dashboard.chat.emptyNoCollection")}
                            </p>
                        </div>
                    )}

                    <AnimatePresence>
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
                                            "rounded-2xl px-4 py-2.5 text-sm leading-relaxed shadow-sm transition-all whitespace-pre-wrap",
                                            msg.role === "user"
                                                ? "rounded-tr-none bg-primary text-primary-foreground"
                                                : "rounded-tl-none border border-border bg-card text-foreground"
                                        )}
                                    >
                                        {msg.content || (isLoading && msg.role === "assistant" ? t("dashboard.chat.streaming") : "")}
                                    </div>

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
