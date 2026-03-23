"use client";

import { AnimatePresence, motion } from "framer-motion";
import { FolderOpen, Loader2 } from "lucide-react";

import { ChatComposer } from "@/components/chat/ChatComposer";
import { ChatConversationRail } from "@/components/chat/ChatConversationRail";
import { ChatEmptyState } from "@/components/chat/ChatEmptyState";
import { ChatMessageCard } from "@/components/chat/ChatMessageCard";
import SubQueryProgress from "@/components/chat/SubQueryProgress";
import { ChatWorkspaceHero } from "@/components/chat/ChatWorkspaceHero";
import { ScrollArea } from "@/components/ui/scroll-area";
import ProviderRuntimeCallout from "@/components/runtime/ProviderRuntimeCallout";
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

const ChatTab = ({ className }: { className?: string }) => {
    const { t } = useI18n();
    const {
        activeCollection,
        activeConversation,
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
        selectedCollectionId,
        sendMessage,
        selectConversation,
        setInput,
        setSelectedCollectionId,
        showStreamingCard,
        startNewConversation,
        streamError,
        streamMessageId,
        textareaRef,
    } = useChatWorkspace();

    const conversationTitle =
        activeConversation?.title
        ?? activeCollection?.name
        ?? t("chatPage.panelDescription");

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
        <div className={cn("chat-panel flex h-[600px] overflow-hidden rounded-[32px]", className)}>
            <div className="flex min-h-0 flex-1 flex-col bg-background/35">
                <div className="shrink-0 border-b border-border/80 px-4 py-4 sm:px-6 sm:py-5">
                    <ChatWorkspaceHero
                        conversationTitle={conversationTitle}
                        activeCollectionName={activeCollection?.name ?? null}
                        conversationCount={conversations.length}
                    />

                    <div className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,20rem)_minmax(0,1fr)]">
                        <section className="rounded-[28px] border border-border bg-card/70 p-4 shadow-sm">
                            <label
                                htmlFor="chat-collection-select"
                                className="text-[10px] font-bold uppercase tracking-[0.22em] text-muted-foreground"
                            >
                                {t("dashboard.chat.collectionSelectLabel")}
                            </label>
                            <div className="mt-3 flex items-center gap-3 rounded-2xl border border-border bg-background/70 px-4 py-3">
                                <FolderOpen className="h-4 w-4 text-primary" />
                                <select
                                    id="chat-collection-select"
                                    value={selectedCollectionId ?? ""}
                                    onChange={(event) => {
                                        const value = Number(event.target.value);
                                        setSelectedCollectionId(Number.isFinite(value) && value > 0 ? value : null);
                                    }}
                                    disabled={!hasCollections || collectionsLoading || isLoading}
                                    aria-label={t("dashboard.chat.collectionSelectLabel")}
                                    className="min-w-0 flex-1 bg-transparent text-sm font-medium outline-none"
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
                                {(collectionsLoading || isConversationsLoading || isConversationLoading) ? (
                                    <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                                ) : null}
                            </div>
                            <p className="mt-3 text-xs leading-relaxed text-muted-foreground">
                                {activeCollection
                                    ? `${activeCollection.documentCount} document(s) • ${conversations.length} ${t("dashboard.chat.conversationsLabel").toLowerCase()}`
                                    : t("dashboard.chat.collectionHint")}
                            </p>
                        </section>

                        <ProviderRuntimeCallout
                            role="chat"
                            assignment={providerRuntimeOverview.chat}
                            isLoading={isProviderRuntimeLoading}
                            error={providerRuntimeError}
                            className="h-full"
                        />
                    </div>
                </div>

                <div className="flex min-h-0 flex-1 flex-col gap-4 px-4 py-5 sm:px-6 sm:py-6">
                    <div className="grid min-h-0 flex-1 gap-4 xl:grid-cols-[22rem_minmax(0,1fr)]">
                        <ChatConversationRail
                            conversations={conversations}
                            activeConversationId={activeConversation?.id ?? null}
                            isLoading={isConversationsLoading}
                            deletingConversationId={deletingConversationId}
                            isWorkspaceBusy={isLoading}
                            onSelectConversation={selectConversation}
                            onStartNewConversation={startNewConversation}
                            onDeleteConversation={deleteConversation}
                        />

                        <section className="surface-elevated flex min-h-[30rem] min-w-0 flex-col overflow-hidden rounded-[32px] border border-border">
                            <ScrollArea ref={scrollAreaRef} className="flex-1 px-4 py-5 sm:px-7">
                                <div className="mx-auto max-w-5xl space-y-7">
                                    {messages.length === 0 && !isConversationLoading ? (
                                        <ChatEmptyState
                                            hasCollections={hasCollections}
                                            hasChatRuntime={hasChatRuntime}
                                            activeCollectionName={activeCollection?.name ?? null}
                                        />
                                    ) : null}

                                    {isConversationLoading ? (
                                        <div className="flex h-96 items-center justify-center gap-3 text-sm text-muted-foreground">
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
                                        <div className="rounded-2xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
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
                                showQuickActions={messages.length === 0 && !isConversationLoading && hasCollections}
                            />
                        </section>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default ChatTab;
