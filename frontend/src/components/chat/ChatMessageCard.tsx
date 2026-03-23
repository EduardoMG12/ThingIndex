"use client";

import { Bot, Loader2, User } from "lucide-react";

import { ChatRichText } from "@/components/chat/ChatRichText";
import SourceReference from "@/components/chat/SourceReference";
import type { ChatWorkspaceMessage } from "@/components/chat/types";
import { useI18n } from "@/lib/i18n/provider";
import { cn } from "@/lib/utils";

function StreamingPlaceholder({ label }: { label: string }) {
    return (
        <div className="flex items-center gap-3 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin text-primary" />
            <span className="font-medium text-foreground/80">{label}</span>
        </div>
    );
}

export function ChatMessageCard({
    message,
    isStreaming,
}: {
    message: ChatWorkspaceMessage;
    isStreaming: boolean;
}) {
    const { t, locale } = useI18n();
    const isUser = message.role === "user";
    const timestamp = new Intl.DateTimeFormat(locale, {
        hour: "2-digit",
        minute: "2-digit",
    }).format(new Date(message.createdAt));

    return (
        <div className={cn("flex gap-4 sm:gap-5", isUser ? "flex-row-reverse" : "flex-row")}>
            <div
                className={cn(
                    "flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border",
                    isUser
                        ? "border-primary bg-primary text-primary-foreground"
                        : "border-border bg-card text-primary shadow-sm"
                )}
            >
                {isUser ? <User className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
            </div>

            <div className={cn("max-w-[94%] space-y-3 sm:max-w-[88%]", isUser ? "text-right" : "text-left")}>
                <div className={cn("flex items-center gap-2", isUser ? "justify-end" : "justify-start")}>
                    <p className="text-[10px] font-bold uppercase tracking-[0.24em] text-muted-foreground">
                        {isUser ? t("dashboard.chat.userLabel") : t("dashboard.chat.assistantLabel")}
                    </p>
                    <time dateTime={message.createdAt} className="text-[10px] text-muted-foreground">
                        {timestamp}
                    </time>
                </div>

                <div
                    className={cn(
                        "break-words rounded-[24px] px-5 py-4 shadow-sm transition-all",
                        isUser
                            ? "rounded-tr-md bg-primary text-primary-foreground whitespace-pre-wrap text-[15px] leading-7"
                            : cn(
                                "rounded-tl-md border border-border bg-card/95 text-foreground",
                                isStreaming && !message.content.trim() && "min-w-72"
                            )
                    )}
                >
                    {isUser ? (
                        message.content
                    ) : (
                        <div className="space-y-3 text-[15px] leading-7 text-foreground/95">
                            {message.content.trim() ? (
                                <>
                                    <ChatRichText content={message.content} />
                                    {isStreaming ? (
                                        <span className="inline-block h-5 w-[2px] translate-y-1 animate-pulse rounded-full bg-primary/70 align-bottom" />
                                    ) : null}
                                </>
                            ) : isStreaming ? (
                                <StreamingPlaceholder label={t("dashboard.chat.streaming")} />
                            ) : null}
                        </div>
                    )}
                </div>

                {message.sources && message.sources.length > 0 ? (
                    <div className="grid gap-2 pt-1 sm:grid-cols-2 xl:grid-cols-3">
                        {message.sources.map((source) => (
                            <SourceReference key={source.id} source={source} />
                        ))}
                    </div>
                ) : null}
            </div>
        </div>
    );
}
