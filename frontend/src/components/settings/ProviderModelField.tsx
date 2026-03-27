"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Check, ChevronDown } from "lucide-react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useI18n } from "@/lib/i18n/provider";

type ProviderModelFieldProps = {
    providerType: string;
    label: string;
    optional?: boolean;
    placeholder: string;
    value: string;
    suggestions?: string[];
    onChange: (value: string) => void;
};

function normalizeModel(value: string) {
    return value.trim().toLowerCase();
}

export default function ProviderModelField({
    providerType,
    label,
    optional,
    placeholder,
    value,
    suggestions = [],
    onChange,
}: ProviderModelFieldProps) {
    const { t } = useI18n();
    const [query, setQuery] = useState(value);
    const [isOpen, setIsOpen] = useState(false);
    const rootRef = useRef<HTMLDivElement | null>(null);
    const inputRef = useRef<HTMLInputElement | null>(null);

    useEffect(() => {
        setQuery(value);
    }, [value]);

    useEffect(() => {
        function handlePointerDown(event: MouseEvent) {
            if (!rootRef.current?.contains(event.target as Node)) {
                setIsOpen(false);
            }
        }

        document.addEventListener("mousedown", handlePointerDown);
        return () => document.removeEventListener("mousedown", handlePointerDown);
    }, []);

    const filteredSuggestions = useMemo(() => {
        const normalizedQuery = normalizeModel(query);
        if (!normalizedQuery) {
            return suggestions;
        }

        return suggestions.filter((suggestion) =>
            normalizeModel(suggestion).includes(normalizedQuery)
        );
    }, [suggestions, query]);

    const quickSuggestions = filteredSuggestions.slice(0, 8);

    function commitValue(nextValue: string) {
        const trimmedValue = nextValue.trim();
        onChange(trimmedValue);
        setQuery(trimmedValue);
        setIsOpen(false);
    }

    return (
        <div ref={rootRef} className="space-y-2">
            <Label htmlFor={`${providerType}-model`}>
                {label}
                {optional ? (
                    <span className="ml-2 text-[10px] uppercase tracking-[0.18em] text-muted-foreground">{t("common.optional")}</span>
                ) : null}
            </Label>

            <div className="relative">
                <Input
                    ref={inputRef}
                    id={`${providerType}-model`}
                    value={query}
                    onChange={(event) => {
                        const nextValue = event.target.value;
                        setQuery(nextValue);
                        onChange(nextValue);
                        setIsOpen(true);
                    }}
                    onFocus={() => setIsOpen(true)}
                    onBlur={() => commitValue(query)}
                    onKeyDown={(event) => {
                        if (event.key === "Enter") {
                            event.preventDefault();
                            if (filteredSuggestions.length > 0) {
                                commitValue(filteredSuggestions[0]);
                                return;
                            }
                            commitValue(query);
                        }

                        if (event.key === "Escape") {
                            setIsOpen(false);
                        }
                    }}
                    placeholder={placeholder}
                    autoComplete="off"
                    spellCheck={false}
                    className="pr-10"
                />

                <button
                    type="button"
                    aria-label={t("settings.toggleModelSuggestions")}
                    onMouseDown={(event) => event.preventDefault()}
                    onClick={() => {
                        setIsOpen((current) => !current);
                        inputRef.current?.focus();
                    }}
                    className="absolute inset-y-0 right-0 inline-flex w-10 items-center justify-center text-muted-foreground transition hover:text-foreground"
                >
                    <ChevronDown className={`h-4 w-4 transition-transform ${isOpen ? "rotate-180" : ""}`} />
                </button>

                {isOpen ? (
                    <div className="absolute left-0 right-0 top-[calc(100%+0.35rem)] z-30 overflow-hidden rounded-md border border-border bg-background shadow-xl">
                        <div className="max-h-64 overflow-y-auto p-1">
                            {quickSuggestions.length > 0 ? (
                                quickSuggestions.map((suggestion) => {
                                    const isSelected = normalizeModel(suggestion) === normalizeModel(value);

                                    return (
                                        <button
                                            key={suggestion}
                                            type="button"
                                            onMouseDown={(event) => event.preventDefault()}
                                            onClick={() => commitValue(suggestion)}
                                            className="flex w-full items-center justify-between rounded px-3 py-2 text-left text-xs transition-colors hover:bg-muted"
                                        >
                                            <span className="truncate font-mono">{suggestion}</span>
                                            {isSelected ? <Check className="h-3.5 w-3.5 text-primary" /> : null}
                                        </button>
                                    );
                                })
                            ) : (
                                <div className="px-3 py-2 text-xs text-muted-foreground">
                                    {t("settings.noSuggestedModels")}
                                </div>
                            )}
                        </div>
                    </div>
                ) : null}
            </div>

        </div>
    );
}
