"use client";

import { useEffect, useId, type ReactNode } from "react";
import { AlertTriangle, Loader2, Trash2 } from "lucide-react";

type DangerConfirmModalProps = {
    isOpen: boolean;
    eyebrow: string;
    title: string;
    description: string;
    confirmLabel: string;
    cancelLabel?: string;
    closeLabel?: string;
    expectedValue?: string;
    confirmationValue?: string;
    confirmationPrompt?: ReactNode;
    confirmationHint?: ReactNode;
    inputLabel?: string;
    isPending?: boolean;
    onConfirmationChange?: (value: string) => void;
    onCancel: () => void;
    onConfirm: () => void;
};

export function DangerConfirmModal({
    isOpen,
    eyebrow,
    title,
    description,
    confirmLabel,
    cancelLabel = "Cancel",
    closeLabel = "Close confirmation",
    expectedValue,
    confirmationValue = "",
    confirmationPrompt,
    confirmationHint,
    inputLabel = "Confirmation value",
    isPending = false,
    onConfirmationChange,
    onCancel,
    onConfirm,
}: DangerConfirmModalProps) {
    const titleId = useId();
    const descriptionId = useId();
    const inputId = useId();

    useEffect(() => {
        if (!isOpen || isPending) {
            return;
        }

        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                event.preventDefault();
                onCancel();
            }
        };

        window.addEventListener("keydown", handleKeyDown);
        return () => {
            window.removeEventListener("keydown", handleKeyDown);
        };
    }, [isOpen, isPending, onCancel]);

    if (!isOpen) {
        return null;
    }

    const requiresConfirmation = typeof expectedValue === "string" && expectedValue.length > 0;
    const isValid = !requiresConfirmation || confirmationValue.trim() === expectedValue;
    const resolvedConfirmationPrompt = requiresConfirmation
        ? confirmationPrompt ?? (
            <>
                Type <span className="font-semibold text-destructive">{expectedValue}</span> to confirm.
            </>
        )
        : null;

    return (
        <div className="fixed inset-0 z-[70] flex items-center justify-center p-4">
            <button
                type="button"
                onClick={onCancel}
                disabled={isPending}
                className="absolute inset-0 bg-black/60 backdrop-blur-sm"
                aria-label={closeLabel}
            />

            <div
                role="dialog"
                aria-modal="true"
                aria-labelledby={titleId}
                aria-describedby={descriptionId}
                className="relative z-10 w-full max-w-xl rounded-3xl border border-destructive/30 bg-card p-6 shadow-2xl"
            >
                <div className="flex items-start gap-4">
                    <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl border border-destructive/30 bg-destructive/10 text-destructive">
                        <AlertTriangle className="h-5 w-5" />
                    </div>
                    <div className="min-w-0">
                        <p className="text-[10px] font-bold uppercase tracking-[0.3em] text-destructive">{eyebrow}</p>
                        <h3 id={titleId} className="mt-2 text-lg font-semibold">{title}</h3>
                        <p id={descriptionId} className="mt-2 text-sm leading-relaxed text-muted-foreground">{description}</p>
                        {resolvedConfirmationPrompt ? <p className="mt-3 text-sm text-foreground/90">{resolvedConfirmationPrompt}</p> : null}
                        {confirmationHint ? (
                            <div className="mt-3 text-sm text-foreground/90">{confirmationHint}</div>
                        ) : null}
                    </div>
                </div>

                {requiresConfirmation ? (
                    <>
                        <label htmlFor={inputId} className="sr-only">
                            {inputLabel}
                        </label>
                        <input
                            id={inputId}
                            aria-label={inputLabel}
                            value={confirmationValue}
                            onChange={(event) => onConfirmationChange?.(event.target.value)}
                            placeholder={expectedValue}
                            className="mt-5 w-full rounded-2xl border border-border bg-background px-4 py-3 text-sm outline-none transition focus:border-destructive/40"
                        />
                    </>
                ) : null}

                <div className="mt-5 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
                    <button
                        type="button"
                        onClick={onCancel}
                        disabled={isPending}
                        className="inline-flex items-center justify-center rounded-xl border border-border px-4 py-2.5 text-sm font-medium text-foreground transition hover:bg-muted disabled:opacity-60"
                    >
                        {cancelLabel}
                    </button>
                    <button
                        type="button"
                        onClick={onConfirm}
                        disabled={!isValid || isPending}
                        className="inline-flex items-center justify-center gap-2 rounded-xl border border-destructive/30 bg-destructive px-4 py-2.5 text-sm font-semibold text-destructive-foreground transition hover:opacity-95 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                        {confirmLabel}
                    </button>
                </div>
            </div>
        </div>
    );
}
