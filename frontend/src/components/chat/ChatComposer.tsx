"use client";

import { type RefObject, useEffect } from "react";
import { Send, Square } from "lucide-react";

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
        <div className="border-t border-border/70 bg-background px-4 py-4 sm:px-6">
            <div className="mx-auto max-w-3xl space-y-3">
                {showQuickActions ? (
                    <div className="flex flex-wrap gap-2">
                            {quickActions.map((action) => (
                                <button
                                    key={action.id}
                                    type="button"
                                    onClick={() => onChange(action.prompt)}
                                    title={action.detail}
                                    className="rounded-full border border-border/70 bg-background px-3 py-1.5 text-sm text-foreground transition-colors hover:border-foreground/20 hover:bg-muted/40"
                                >
                                    {action.label}
                                </button>
                            ))}
                    </div>
                ) : null}

                <div className="relative rounded-[28px] border border-border/70 bg-muted/15 p-3 pr-14">
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
                        className="min-h-[72px] max-h-[220px] resize-none border-0 bg-transparent px-1 py-1 text-[15px] leading-7 focus-visible:ring-0 focus-visible:ring-offset-0"
                        rows={2}
                    />

                    <Button
                        type="button"
                        onClick={onSubmit}
                        disabled={!canSubmit}
                        size="icon"
                        aria-label={t("dashboard.chat.sendLabel")}
                        className="absolute bottom-3 right-3 h-10 w-10 rounded-full transition-transform active:scale-95"
                    >
                        <Send className="h-4 w-4" />
                    </Button>
                </div>

                <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                    <p className="text-xs leading-relaxed text-muted-foreground">
                        {footerHint}
                    </p>

                    <div className="flex items-center gap-3">
                        {isLoading ? (
                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={onCancel}
                                className="gap-2 rounded-full px-3 py-2 text-[11px]"
                            >
                                <Square className="h-3 w-3" />
                                {t("dashboard.chat.cancel")}
                            </Button>
                        ) : null}
                        <p className="text-[11px] text-muted-foreground">
                            {t("dashboard.chat.newLineHint")}
                        </p>
                    </div>
                </div>
            </div>
        </div>
    );
}
