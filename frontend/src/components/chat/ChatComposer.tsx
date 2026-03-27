"use client";

import { type RefObject, useEffect } from "react";
import { AlertCircle, Send, Square } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useI18n } from "@/lib/i18n/provider";

type ChatQuickAction = {
    id: string;
    label: string;
    detail: string;
    prompt: string;
};

export function ChatComposer({
    value,
    onChange,
    onSubmit,
    onCancel,
    isLoading,
    canType,
    canSubmit,
    hasCollections,
    hasChatRuntime,
    collectionName,
    textareaRef,
    quickActions,
    showQuickActions,
}: {
    value: string;
    onChange: (value: string) => void;
    onSubmit: () => void;
    onCancel: () => void;
    isLoading: boolean;
    canType: boolean;
    canSubmit: boolean;
    hasCollections: boolean;
    hasChatRuntime: boolean;
    collectionName: string | null;
    textareaRef: RefObject<HTMLTextAreaElement | null>;
    quickActions: ChatQuickAction[];
    showQuickActions: boolean;
}) {
    const { t } = useI18n();

    useEffect(() => {
        const element = textareaRef.current;
        if (!element) {
            return;
        }

        element.style.height = "0px";
        element.style.height = `${Math.min(element.scrollHeight, 220)}px`;
    }, [textareaRef, value]);

    const footerHint = !hasCollections
        ? t("dashboard.chat.createCollectionHint")
        : !hasChatRuntime
            ? t("providerRuntime.chat.missing")
            : collectionName
                ? `${t("dashboard.chat.activeCollection")} ${collectionName}`
                : t("dashboard.chat.collectionHint");

    return (
        <div className="border-t border-border bg-card/90 px-4 py-4 sm:px-7 sm:py-5">
            <div className="mx-auto max-w-5xl space-y-4">
                {showQuickActions ? (
                    <div className="rounded-[28px] border border-border bg-background/70 p-4">
                        <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                            <div>
                                <p className="text-[10px] font-bold uppercase tracking-[0.24em] text-primary">
                                    {t("dashboard.chat.quickActionsTitle")}
                                </p>
                                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                                    {t("dashboard.chat.quickActionsDescription")}
                                </p>
                            </div>
                        </div>
                        <div className="mt-4 flex flex-wrap gap-2">
                            {quickActions.map((action) => (
                                <button
                                    key={action.id}
                                    type="button"
                                    onClick={() => onChange(action.prompt)}
                                    className="rounded-full border border-border bg-card px-3 py-2 text-left transition-colors hover:border-primary/30 hover:text-primary"
                                >
                                    <span className="block text-xs font-semibold">{action.label}</span>
                                    <span className="block text-[11px] text-muted-foreground">{action.detail}</span>
                                </button>
                            ))}
                        </div>
                    </div>
                ) : null}

                <div className="relative rounded-[28px] border border-border bg-background/80 p-3 pr-16 shadow-sm">
                    <label htmlFor="chat-composer" className="sr-only">
                        {t("dashboard.chat.composerLabel")}
                    </label>
                    <Textarea
                        id="chat-composer"
                        ref={textareaRef}
                        placeholder={hasCollections ? t("dashboard.chat.placeholder") : t("dashboard.chat.disabledPlaceholder")}
                        value={value}
                        onChange={(event) => onChange(event.target.value)}
                        onKeyDown={(event) => {
                            if (event.key === "Enter" && !event.shiftKey) {
                                event.preventDefault();
                                onSubmit();
                            }
                        }}
                        disabled={!canType}
                        aria-label={t("dashboard.chat.composerLabel")}
                        className="min-h-[104px] max-h-[220px] resize-none border-0 bg-transparent px-2 py-2 text-base leading-7 focus-visible:ring-0 focus-visible:ring-offset-0"
                        rows={3}
                    />

                    <Button
                        type="button"
                        onClick={onSubmit}
                        disabled={!canSubmit}
                        size="icon"
                        aria-label={t("dashboard.chat.sendLabel")}
                        className="absolute bottom-3 right-3 h-12 w-12 rounded-2xl shadow-gold transition-transform active:scale-95"
                    >
                        <Send className="h-4 w-4" />
                    </Button>
                </div>

                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <p className="flex items-center gap-1.5 text-[11px] leading-relaxed text-muted-foreground">
                        <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                        {footerHint}
                    </p>

                    <div className="flex items-center gap-3">
                        {isLoading ? (
                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={onCancel}
                                className="gap-2 rounded-2xl px-3 py-2 text-[10px] uppercase tracking-widest"
                            >
                                <Square className="h-3 w-3" />
                                {t("dashboard.chat.cancel")}
                            </Button>
                        ) : null}
                        <p className="font-mono text-[9px] uppercase tracking-widest text-muted-foreground">
                            {t("dashboard.chat.newLineHint")}
                        </p>
                    </div>
                </div>
            </div>
        </div>
    );
}
