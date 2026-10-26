"use client";

import { useState } from "react";

import { Bot, ChevronLeft, ChevronRight, Loader2, User } from "lucide-react";

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
    fonteAtivaId,
    onFonteAtiva,
    indexadoEm,
}: {
    message: ChatWorkspaceMessage;
    isStreaming: boolean;
    onVerNoDocumento?: (source: Source) => void;
    /** Fonte ativa pelo id. Controlada pelo pai porque o card remonta depois do stream. */
    fonteAtivaId?: string | null;
    onFonteAtiva?: (id: string) => void;
    /** Data do indice atual de cada documento, para avisar fonte de indice antigo. */
    indexadoEm?: ReadonlyMap<number, string>;
}) {
    const { t, locale } = useI18n();
    const isUser = message.role === "user";
    const [fonteAtivaLocal, setFonteAtivaLocal] = useState<string | null>(null);
    const ativaId = fonteAtivaId !== undefined ? fonteAtivaId : fonteAtivaLocal;
    const fontes = message.sources ?? [];
    const fonteAtiva = fontes.findIndex((fonte) => fonte.id === ativaId);
    const irParaFonte = (indice: number) => {
        const fonte = fontes[indice];
        if (!fonte) {
            return;
        }
        setFonteAtivaLocal(fonte.id);
        onFonteAtiva?.(fonte.id);
        onVerNoDocumento?.(fonte);
    };
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
                            {message.status === "error" ? (
                                <div className="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" data-testid="resposta-com-erro">
                                    <p className="font-medium">{t("dashboard.chat.answerFailed")}</p>
                                    {message.errorDetail ? <p className="mt-1 break-words text-xs opacity-90">{message.errorDetail}</p> : null}
                                </div>
                            ) : null}
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
                {fontes.length > 0 ? (
                    <div className="space-y-2 pt-1">
                        {onVerNoDocumento ? (
                            <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
                                <span>{t("dashboard.chat.sourcesTitle", { count: fontes.length })}</span>
                                <span className="flex-1" />
                                <button
                                    type="button"
                                    data-testid="fonte-anterior"
                                    aria-label={t("dashboard.chat.previousSource")}
                                    title={t("dashboard.chat.previousSource")}
                                    disabled={fonteAtiva <= 0}
                                    onClick={() => irParaFonte(fonteAtiva - 1)}
                                    className="rounded-md border border-border/60 p-1 hover:bg-muted disabled:opacity-40"
                                >
                                    <ChevronLeft className="h-3.5 w-3.5" aria-hidden />
                                </button>
                                <span className="tabular-nums" data-testid="fonte-posicao">
                                    {fonteAtiva >= 0 ? `${fonteAtiva + 1}/${fontes.length}` : `–/${fontes.length}`}
                                </span>
                                <button
                                    type="button"
                                    data-testid="fonte-proxima"
                                    aria-label={t("dashboard.chat.nextSource")}
                                    title={t("dashboard.chat.nextSource")}
                                    disabled={fonteAtiva >= fontes.length - 1}
                                    onClick={() => irParaFonte(fonteAtiva + 1)}
                                    className="rounded-md border border-border/60 p-1 hover:bg-muted disabled:opacity-40"
                                >
                                    <ChevronRight className="h-3.5 w-3.5" aria-hidden />
                                </button>
                            </div>
                        ) : null}
                        {/* auto-fill pela largura do painel (~500px no workspace), nao da tela. */}
                        <div className="grid grid-cols-[repeat(auto-fill,minmax(16rem,1fr))] gap-2">
                            {fontes.map((source, indice) => (
                                <SourceReference
                                    key={source.id}
                                    source={source}
                                    numero={indice + 1}
                                    ativa={indice === fonteAtiva}
                                    reindexado={fonteDeIndiceAntigo(source, indexadoEm)}
                                    onVerNoDocumento={onVerNoDocumento ? () => irParaFonte(indice) : undefined}
                                />
                            ))}
                        </div>
                    </div>
                ) : null}
            </div>
        </div>
    );
}

/** Fonte de indice antigo: o documento foi reindexado depois da data gravada na fonte. */
function fonteDeIndiceAntigo(source: Source, indexadoEm?: ReadonlyMap<number, string>): boolean {
    if (!source.documentId || !source.indexedAt || !indexadoEm) {
        return false;
    }
    const atual = indexadoEm.get(source.documentId);
    if (!atual) {
        return false;
    }
    const a = Date.parse(atual), b = Date.parse(source.indexedAt);
    return Number.isFinite(a) && Number.isFinite(b) && a - b > 1000;
}
