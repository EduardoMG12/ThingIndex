"use client";

import { useState } from "react";
import { History, Loader2, MessageSquarePlus, Trash2 } from "lucide-react";

import { ScrollArea } from "@/components/ui/scroll-area";
import { DangerConfirmModal } from "@/components/ui/danger-confirm-modal";
import { useI18n } from "@/lib/i18n/provider";
import type { ChatConversationSummary } from "@/lib/api/chat";
import { cn } from "@/lib/utils";

function formatConversationTimestamp(locale: string, value: string | null) {
    if (!value) {
        return null;
    }

    const timestamp = new Date(value);
    if (Number.isNaN(timestamp.getTime())) {
        return null;
    }

    return new Intl.DateTimeFormat(locale, {
        month: "short",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
    }).format(timestamp);
}

export function ChatConversationRail({
    conversations,
    activeConversationId,
    isLoading,
    deletingConversationId,
    isWorkspaceBusy,
    onSelectConversation,
    onStartNewConversation,
    onDeleteConversation,
}: {
    conversations: ChatConversationSummary[];
    activeConversationId: number | null;
    isLoading: boolean;
    deletingConversationId: number | null;
    isWorkspaceBusy: boolean;
    onSelectConversation: (conversationId: number) => void;
    onStartNewConversation: () => void;
    onDeleteConversation: (conversationId: number) => Promise<void>;
}) {
    const { t, locale } = useI18n();
    const [conversationPendingDelete, setConversationPendingDelete] = useState<ChatConversationSummary | null>(null);
    const [confirmationValue, setConfirmationValue] = useState("");

    const confirmDeleteConversation = async () => {
        if (!conversationPendingDelete) {
            return;
        }

        try {
            await onDeleteConversation(conversationPendingDelete.id);
            setConversationPendingDelete(null);
            setConfirmationValue("");
        } catch {
            return;
        }
    };

    return (
        <>
            <aside className="surface-elevated flex min-h-[18rem] flex-col overflow-hidden rounded-[32px] border border-border p-4 shadow-sm">
                <div className="flex flex-wrap items-start justify-between gap-3">
                    <div>
                        <div className="flex items-center gap-2">
                            <div className="flex h-9 w-9 items-center justify-center rounded-2xl bg-primary/12">
                                <History className="h-4 w-4 text-primary" />
                            </div>
                            <div>
                                <p className="text-[10px] font-bold uppercase tracking-[0.24em] text-muted-foreground">
                                    {t("dashboard.chat.conversationsLabel")}
                                </p>
                                <p className="mt-1 text-sm font-semibold text-foreground">
                                    {conversations.length}
                                </p>
                            </div>
                        </div>
                        <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
                            {t("dashboard.chat.historyDescription")}
                        </p>
                    </div>

                    <button
                        type="button"
                        onClick={onStartNewConversation}
                        disabled={isWorkspaceBusy}
                        className="inline-flex items-center gap-2 rounded-full border border-border bg-transparent px-3 py-2 text-xs font-medium text-foreground transition-colors hover:border-primary/50 disabled:pointer-events-none disabled:opacity-50"
                    >
                        <MessageSquarePlus className="h-4 w-4" />
                        {t("dashboard.chat.newConversation")}
                    </button>
                </div>

                <ScrollArea className="mt-5 flex-1 pr-2">
                    <div className="space-y-2">
                        {isLoading ? (
                            <div className="flex items-center gap-3 rounded-2xl border border-border/70 bg-background/60 px-4 py-3 text-sm text-muted-foreground">
                                <Loader2 className="h-4 w-4 animate-spin" />
                                {t("dashboard.chat.loadingConversation")}
                            </div>
                        ) : conversations.length === 0 ? (
                            <div className="rounded-[24px] border border-dashed border-border/80 bg-background/55 px-4 py-5 text-sm leading-relaxed text-muted-foreground">
                                {t("dashboard.chat.noConversations")}
                            </div>
                        ) : (
                            conversations.map((conversation) => {
                                const isActive = conversation.id === activeConversationId;
                                const timestamp = formatConversationTimestamp(locale, conversation.lastMessageAt ?? conversation.updatedAt);
                                const isDeleting = deletingConversationId === conversation.id;

                                return (
                                    <div
                                        key={conversation.id}
                                        className={cn(
                                            "rounded-[24px] border transition-colors",
                                            isActive
                                                ? "border-primary/30 bg-primary/8"
                                                : "border-border/70 bg-background/55 hover:border-primary/20"
                                        )}
                                    >
                                        <div className="flex items-start gap-3 px-4 py-3">
                                            <button
                                                type="button"
                                                onClick={() => onSelectConversation(conversation.id)}
                                                disabled={isWorkspaceBusy}
                                                aria-current={isActive ? "true" : undefined}
                                                className="min-w-0 flex-1 text-left"
                                            >
                                                <p className="line-clamp-2 text-sm font-semibold leading-6 text-foreground">
                                                    {conversation.title}
                                                </p>
                                                <div className="mt-2 flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
                                                    {isActive ? (
                                                        <span className="rounded-full bg-primary/12 px-2 py-1 font-medium text-primary">
                                                            {t("dashboard.chat.activeConversationBadge")}
                                                        </span>
                                                    ) : null}
                                                    {timestamp ? <span>{timestamp}</span> : null}
                                                </div>
                                            </button>

                                            <button
                                                type="button"
                                                onClick={() => {
                                                    setConversationPendingDelete(conversation);
                                                    setConfirmationValue("");
                                                }}
                                                disabled={isWorkspaceBusy || isDeleting}
                                                aria-label={t("dashboard.chat.deleteConversationLabel")}
                                                className="rounded-xl p-2 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
                                            >
                                                {isDeleting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                                            </button>
                                        </div>
                                    </div>
                                );
                            })
                        )}
                    </div>
                </ScrollArea>
            </aside>

            <DangerConfirmModal
                isOpen={conversationPendingDelete !== null}
                eyebrow={t("shell.chat.deleteConversation.eyebrow")}
                title={conversationPendingDelete
                    ? t("shell.chat.deleteConversation.titleWithName", { title: conversationPendingDelete.title })
                    : t("shell.chat.deleteConversation.title")}
                description={t("shell.chat.deleteConversation.description")}
                confirmLabel={t("shell.chat.deleteConversation.confirmLabel")}
                cancelLabel={t("common.dangerConfirm.cancelLabel")}
                closeLabel={t("common.dangerConfirm.closeLabel")}
                expectedValue={conversationPendingDelete?.title}
                confirmationValue={confirmationValue}
                confirmationPrompt={conversationPendingDelete
                    ? t("common.dangerConfirm.typeToConfirm", { value: conversationPendingDelete.title })
                    : undefined}
                inputLabel={t("common.dangerConfirm.inputLabel")}
                onConfirmationChange={setConfirmationValue}
                onCancel={() => {
                    if (deletingConversationId !== null) {
                        return;
                    }

                    setConversationPendingDelete(null);
                    setConfirmationValue("");
                }}
                onConfirm={() => void confirmDeleteConversation()}
                isPending={deletingConversationId === conversationPendingDelete?.id}
            />
        </>
    );
}
