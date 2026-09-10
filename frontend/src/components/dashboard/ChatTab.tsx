"use client";

import { useState } from "react";

import { AnimatePresence, motion } from "framer-motion";
import { FolderOpen, Loader2, MessageSquarePlus } from "lucide-react";

import { ChatComposer } from "@/components/chat/ChatComposer";
import { ChatEmptyState } from "@/components/chat/ChatEmptyState";
import { ChatMessageCard } from "@/components/chat/ChatMessageCard";
import type { Source } from "@/components/chat/SourceReference";
import SubQueryProgress from "@/components/chat/SubQueryProgress";
import ProviderRuntimeCallout from "@/components/runtime/ProviderRuntimeCallout";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChatWorkspace } from "@/hooks/use-chat-workspace";
import { useI18n } from "@/lib/i18n/provider";
import { cn } from "@/lib/utils";

function StreamingCard() {
    return (
        <motion.div
            key="streaming-card"
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -6 }}
            className="rounded-2xl border border-border/60 bg-background/80 px-4 py-3"
        >
            <div className="space-y-2 animate-pulse">
                <div className="h-2.5 w-28 rounded-full bg-muted" />
                <div className="h-2 w-36 rounded-full bg-muted/70" />
                <div className="h-2 w-full rounded-full bg-muted/60" />
            </div>
        </motion.div>
    );
}

type ChatTabProps = {
    className?: string;
    /** Colecao escolhida por quem embute o chat; esconde o seletor proprio. */
    collectionId?: number | null;
    /** Cabecalho de uma linha, sem acoes rapidas: para caber num painel. */
    compacto?: boolean;
    onVerNoDocumento?: (source: Source) => void;
    /** Data do indice atual por documento (para marcar fonte de indice antigo). */
    indexadoEm?: ReadonlyMap<number, string>;
};

const ChatTab = ({ className, collectionId = null, compacto = false, onVerNoDocumento, indexadoEm }: ChatTabProps) => {
    // Fonte focada no canvas, pelo id: sobrevive ao recarregamento da conversa apos o stream.
    const [fonteAtivaId, setFonteAtivaId] = useState<string | null>(null);
    const { t } = useI18n();
    const controlado = collectionId !== null;
    const {
        activeCollection,
        activeConversation,
        activeConversationId,
        activeSubQueries,
        conversations,
        selectConversation,
        startNewConversation,
        canSubmit,
        canType,
        cancelStreaming,
        collections,
        collectionsLoading,
        hasChatRuntime,
        hasCollections,
        input,
        isConversationLoading,
        isLoading,
        isProviderRuntimeLoading,
        isStreaming,
        messages,
        providerRuntimeError,
        providerRuntimeOverview,
        scrollAreaRef,
        scrollRef,
        selectedCollectionId,
        sendMessage,
        setInput,
        setSelectedCollectionId,
        showStreamingCard,
        streamError,
        streamMessageId,
        textareaRef,
    } = useChatWorkspace({ collectionId });

    const showsRuntimeCallout = messages.length === 0 && !isConversationLoading;

    const conversationTitle =
        activeConversation?.title
        ?? activeCollection?.name
        ?? t("chatPage.title");

    const quickActions = [
        {
            id: "coverage",
            label: t("dashboard.chat.quickActions.coverage.label"),
            detail: t("dashboard.chat.quickActions.coverage.detail"),
            prompt: t("dashboard.chat.quickActions.coverage.prompt"),
        },
        {
            id: "deadlines",
            label: t("dashboard.chat.quickActions.deadlines.label"),
            detail: t("dashboard.chat.quickActions.deadlines.detail"),
            prompt: t("dashboard.chat.quickActions.deadlines.prompt"),
        },
        {
            id: "risks",
            label: t("dashboard.chat.quickActions.risks.label"),
            detail: t("dashboard.chat.quickActions.risks.detail"),
            prompt: t("dashboard.chat.quickActions.risks.prompt"),
        },
    ];

    return (
        <section className={cn("flex h-full min-h-0 min-w-0 flex-col bg-background", className)}>
            <header className={cn("border-b border-border/70", compacto ? "px-3 py-2" : "px-4 py-3 sm:px-6")}>
                <div className={cn("mx-auto flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between", compacto ? "" : "max-w-4xl")}>
                    <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-foreground">
                            {conversationTitle}
                        </p>
                        {/* Sem o callout na tela, este aviso e a unica pista de
                            que o chat nao tem LLM. Com ele, seria a mesma frase
                            tres vezes na mesma tela. */}
                        {!hasChatRuntime && !isProviderRuntimeLoading && !showsRuntimeCallout ? (
                            <p className="mt-1 text-sm text-destructive">
                                {providerRuntimeError || t("providerRuntime.chat.missing")}
                            </p>
                        ) : null}
                    </div>

                    {compacto ? (
                        // Sem o rail do menu lateral, a lista de conversas vive aqui.
                        <div className="flex items-center gap-1">
                            <label htmlFor="chat-conversation-select" className="sr-only">
                                {t("dashboard.chat.conversationsLabel")}
                            </label>
                            <select
                                id="chat-conversation-select"
                                value={activeConversationId ?? ""}
                                onChange={(event) => {
                                    const id = Number(event.target.value);
                                    if (Number.isFinite(id) && id > 0) {
                                        selectConversation(id);
                                    }
                                }}
                                disabled={isLoading}
                                className="max-w-[160px] rounded-md border border-border/70 bg-background px-2 py-1 text-xs"
                            >
                                <option value="">{t("dashboard.chat.newConversation")}</option>
                                {conversations.map((conversation) => (
                                    <option key={conversation.id} value={conversation.id}>
                                        {conversation.title}
                                    </option>
                                ))}
                            </select>
                            <button
                                type="button"
                                onClick={startNewConversation}
                                disabled={isLoading}
                                aria-label={t("dashboard.chat.newConversation")}
                                title={t("dashboard.chat.newConversation")}
                                className="rounded-md border border-border/70 p-1 text-primary hover:bg-primary/10 disabled:opacity-50"
                            >
                                <MessageSquarePlus className="h-4 w-4" aria-hidden />
                            </button>
                        </div>
                    ) : null}
                    {controlado ? null : (
                    <div className="w-full max-w-sm">
                        <label htmlFor="chat-collection-select" className="sr-only">
                            {t("dashboard.chat.collectionSelectLabel")}
                        </label>
                        <div className="flex items-center gap-3 rounded-full border border-border/70 bg-background px-4 py-2.5">
                            <FolderOpen className="h-4 w-4 text-muted-foreground" />
                            <select
                                id="chat-collection-select"
                                value={selectedCollectionId ?? ""}
                                onChange={(event) => {
                                    const value = Number(event.target.value);
                                    setSelectedCollectionId(Number.isFinite(value) && value > 0 ? value : null);
                                }}
                                disabled={!hasCollections || collectionsLoading || isLoading}
                                aria-label={t("dashboard.chat.collectionSelectLabel")}
                                className="min-w-0 flex-1 bg-transparent text-sm outline-none"
                            >
                                {!hasCollections ? (
                                    <option value="">{t("dashboard.chat.noCollections")}</option>
                                ) : null}
                                {collections
                                    .filter((collection) => typeof collection.id === "number")
                                    .map((collection) => (
                                        <option key={String(collection.id)} value={collection.id}>
                                            {collection.name}
                                        </option>
                                    ))}
                            </select>
                            {(collectionsLoading || isConversationLoading) ? (
                                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                            ) : null}
                        </div>
                    </div>
                    )}
                </div>
            </header>

            <ScrollArea ref={scrollAreaRef} className={cn("flex-1", compacto ? "px-3 py-3" : "px-4 py-6 sm:px-6")}>
                <div className={cn("mx-auto space-y-6", compacto ? "" : "max-w-3xl")}>
                    {/* Ate aqui a tela de documentos mostrava qual LLM estava
                        ativo e o chat nao — o usuario nao tinha como saber quem
                        estava respondendo. O componente, a copia e o teste
                        unitario do papel "chat" ja existiam; so a pagina nunca
                        renderizava. So aparece com a conversa vazia para nao
                        empurrar as mensagens para baixo o tempo todo. */}
                    {showsRuntimeCallout ? (
                        <>
                            <ProviderRuntimeCallout
                                role="chat"
                                assignment={providerRuntimeOverview.chat}
                                isLoading={isProviderRuntimeLoading}
                                error={providerRuntimeError}
                                compacto={compacto}
                            />
                            <ChatEmptyState
                                hasCollections={hasCollections}
                                hasChatRuntime={hasChatRuntime}
                                activeCollectionName={activeCollection?.name ?? null}
                            />
                        </>
                    ) : null}

                    {isConversationLoading ? (
                        <div className="flex h-72 items-center justify-center gap-3 text-sm text-muted-foreground">
                            <Loader2 className="h-5 w-5 animate-spin" />
                            {t("dashboard.chat.loadingConversation")}
                        </div>
                    ) : null}

                    <AnimatePresence>
                        {showStreamingCard ? <StreamingCard /> : null}
                        {messages.map((message) => (
                            <motion.div
                                key={message.id}
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                            >
                                <ChatMessageCard
                                    message={message}
                                    isStreaming={isStreaming && message.id === streamMessageId}
                                    onVerNoDocumento={onVerNoDocumento}
                                    fonteAtivaId={fonteAtivaId}
                                    onFonteAtiva={setFonteAtivaId}
                                    indexadoEm={indexadoEm}
                                />
                            </motion.div>
                        ))}
                    </AnimatePresence>

                    {activeSubQueries.length > 0 ? (
                        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
                            <SubQueryProgress queries={activeSubQueries} />
                        </motion.div>
                    ) : null}

                    {streamError ? (
                        <div className="rounded-2xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
                            {streamError}
                        </div>
                    ) : null}

                    <div ref={scrollRef} />
                </div>
            </ScrollArea>

            <ChatComposer
                value={input}
                onChange={(value) => {
                    setInput(value);
                    if (value && textareaRef.current) {
                        textareaRef.current.focus();
                    }
                }}
                onSubmit={() => void sendMessage()}
                onCancel={cancelStreaming}
                isLoading={isLoading}
                canType={canType}
                canSubmit={canSubmit}
                hasCollections={hasCollections}
                hasChatRuntime={hasChatRuntime}
                collectionName={activeCollection?.name ?? null}
                textareaRef={textareaRef}
                quickActions={quickActions}
                showQuickActions={!compacto && messages.length === 0 && !isConversationLoading && hasCollections}
            />
        </section>
    );
};

export default ChatTab;
