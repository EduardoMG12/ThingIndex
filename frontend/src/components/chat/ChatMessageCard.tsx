"use client";

import { Bot, Loader2, User } from "lucide-react";

import { ChatRichText } from "@/components/chat/ChatRichText";
import SourceReference, { type Source } from "@/components/chat/SourceReference";
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
    onVerNoDocumento,
}: {
    message: ChatWorkspaceMessage;
    isStreaming: boolean;
    onVerNoDocumento?: (source: Source) => void;
}) {
    const { t, locale } = useI18n();
    const isUser = message.role === "user";
    const timestamp = new Intl.DateTimeFormat(locale, {
        hour: "2-digit",
        minute: "2-digit",
    }).format(new Date(message.createdAt));

    return (
        <div className={cn("flex gap-3", isUser ? "justify-end" : "justify-start")}>
            {!isUser ? (
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted/70 text-muted-foreground">
                    <Bot className="h-4 w-4" />
                </div>
            ) : null}

            <div className={cn("max-w-[92%] space-y-2 sm:max-w-[85%]", isUser ? "items-end text-right" : "text-left")}>
                <div className={cn("flex items-center gap-2 text-[11px] text-muted-foreground", isUser ? "justify-end" : "justify-start")}>
                    {isUser ? (
                        <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-muted/70 text-muted-foreground">
                            <User className="h-3.5 w-3.5" />
                        </span>
                    ) : null}
                    <span>{isUser ? t("dashboard.chat.userLabel") : t("dashboard.chat.assistantLabel")}</span>
                    <time dateTime={message.createdAt}>
                        {timestamp}
                    </time>
                </div>

                <div
                    className={cn(
                        "break-words transition-all",
                        isUser
                            ? "rounded-[24px] bg-muted px-5 py-3 text-[15px] leading-7 text-foreground"
                            : cn(
                                "px-1 text-foreground",
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

                {!isUser && message.usage ? (
                    <p
                        className="px-1 text-[11px] text-muted-foreground"
                        data-testid="consumo-da-resposta"
                        title={message.usage.usageReported ? undefined : t("dashboard.chat.usage.estimated")}
                    >
                        {message.usage.usageReported ? "" : "≈ "}
                        {t("dashboard.chat.usage.tokens", { count: new Intl.NumberFormat(locale).format(message.usage.promptTokens + message.usage.completionTokens) })}
                        {message.usage.model ? ` · ${message.usage.model}` : ""}
                        {typeof message.usage.estimatedCostUsd === "number"
                            ? ` · ${new Intl.NumberFormat(locale, { style: "currency", currency: "USD", maximumFractionDigits: 6 }).format(message.usage.estimatedCostUsd)}`
                            : ""}
                    </p>
                ) : null}
                {message.sources && message.sources.length > 0 ? (
                    <div className="grid gap-2 pt-1 sm:grid-cols-2">
                        {message.sources.map((source) => (
                            <SourceReference key={source.id} source={source} onVerNoDocumento={onVerNoDocumento} />
                        ))}
                    </div>
                ) : null}
            </div>
        </div>
    );
}
