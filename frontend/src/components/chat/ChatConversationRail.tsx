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
            <aside className="flex min-h-0 flex-col border-b border-border/70 bg-muted/15 lg:border-b-0 lg:border-r">
                <div className="border-b border-border/60 px-4 py-4">
                    <div className="flex items-center justify-between gap-3">
                        <div className="min-w-0">
                            <div className="flex items-center gap-2">
                                <History className="h-4 w-4 text-muted-foreground" />
                                <p className="truncate text-sm font-medium text-foreground">
                                    {t("dashboard.chat.conversationsLabel")}
                                </p>
                            </div>
                            <p className="mt-1 text-xs text-muted-foreground">
                                {conversations.length}
                            </p>
                        </div>

                        <button
                            type="button"
                            onClick={onStartNewConversation}
                            disabled={isWorkspaceBusy}
                            className="inline-flex items-center gap-2 rounded-full border border-border/70 bg-background px-3 py-2 text-xs font-medium text-foreground transition-colors hover:border-foreground/20 hover:bg-muted/40 disabled:pointer-events-none disabled:opacity-50"
                        >
                            <MessageSquarePlus className="h-4 w-4" />
                            {t("dashboard.chat.newConversation")}
                        </button>
                    </div>
                </div>

                <ScrollArea className="flex-1 px-3 py-3">
                    <div className="space-y-2">
                        {isLoading ? (
                            <div className="flex items-center gap-3 rounded-2xl px-3 py-3 text-sm text-muted-foreground">
                                <Loader2 className="h-4 w-4 animate-spin" />
                                {t("dashboard.chat.loadingConversation")}
                            </div>
                        ) : conversations.length === 0 ? (
                            <div className="rounded-2xl border border-dashed border-border/70 bg-background/70 px-3 py-4 text-sm leading-relaxed text-muted-foreground">
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
                                            "rounded-2xl border transition-colors",
                                            isActive
                                                ? "border-primary/25 bg-primary/5"
                                                : "border-transparent bg-transparent hover:border-border/70 hover:bg-background/70"
                                        )}
                                    >
                                        <div className="flex items-start gap-3 px-3 py-3">
                                            <button
                                                type="button"
                                                onClick={() => onSelectConversation(conversation.id)}
                                                disabled={isWorkspaceBusy}
                                                aria-current={isActive ? "true" : undefined}
                                                className="min-w-0 flex-1 text-left"
                                            >
                                                <p className="line-clamp-2 text-sm font-medium leading-6 text-foreground">
                                                    {conversation.title}
                                                </p>
                                                <div className="mt-1.5 flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
                                                    {isActive ? (
                                                        <span className="rounded-full bg-primary/10 px-2 py-0.5 font-medium text-primary">
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
