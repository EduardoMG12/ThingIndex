"use client";

import { useEffect, useRef, useState } from "react";
import { Check, Hash, Loader2 } from "lucide-react";
import { Input } from "@/components/ui/input";
import { useCollections } from "@/hooks/use-collections";
import { getCollectionName } from "@/lib/api/contracts/document-context";
import { useI18n } from "@/lib/i18n/provider";
import { cn } from "@/lib/utils";

type CollectionComboboxProps = {
    value: string;
    onChange: (value: string) => void;
    className?: string;
};

function normalizeCollection(value: string) {
    return value.trim().toLowerCase();
}

export default function CollectionCombobox({ value, onChange, className }: CollectionComboboxProps) {
    const { t } = useI18n();
    const [query, setQuery] = useState(value);
    const [isOpen, setIsOpen] = useState(false);
    const rootRef = useRef<HTMLDivElement | null>(null);
    const { collections, isLoading, error, reloadCollections } = useCollections();

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

    const filteredCollections = collections.filter((collection) => {
        const normalizedQuery = normalizeCollection(query);

        if (!normalizedQuery) {
            return true;
        }

        return normalizeCollection(collection.name).includes(normalizedQuery);
    });

    const commitCollection = (nextValue: string) => {
        const trimmedValue = nextValue.trim();
        onChange(trimmedValue);
        setQuery(trimmedValue);
        setIsOpen(false);
    };

    return (
        <div ref={rootRef} className="relative">
            <Input
                value={query}
                onChange={(event) => {
                    const nextValue = event.target.value;
                    setQuery(nextValue);
                    onChange(nextValue);
                    setIsOpen(true);
                }}
                onFocus={() => {
                    setIsOpen(true);
                    if (!isLoading && error) {
                        void reloadCollections();
                    }
                }}
                onBlur={() => commitCollection(query)}
                onKeyDown={(event) => {
                    if (event.key === "Enter") {
                        event.preventDefault();

                        if (filteredCollections.length > 0) {
                            commitCollection(getCollectionName(filteredCollections[0]));
                            return;
                        }

                        commitCollection(query);
                    }

                    if (event.key === "Escape") {
                        setIsOpen(false);
                    }
                }}
                placeholder={t("dashboard.upload.selectOrTypeCollection")}
                className={cn("h-7 bg-background font-mono text-xs", className)}
            />

            {isOpen && (
                <div className="absolute left-0 right-0 top-[calc(100%+0.35rem)] z-30 overflow-hidden rounded-md border border-border bg-background shadow-xl">
                    <div className="max-h-64 overflow-y-auto p-1">
                        {isLoading && (
                            <div className="flex items-center gap-2 px-3 py-2 text-xs text-muted-foreground">
                                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                {t("dashboard.upload.loadingCollections")}
                            </div>
                        )}

                        {!isLoading && error && (
                            <div className="rounded px-3 py-2 text-xs">
                                <p className="text-destructive">{t("dashboard.upload.collectionLoadError")}</p>
                                <div className="mt-2 flex flex-wrap items-center gap-3">
                                    <button
                                        type="button"
                                        onMouseDown={(event) => event.preventDefault()}
                                        onClick={() => void reloadCollections()}
                                        className="font-medium text-primary transition-colors hover:text-primary/80"
                                    >
                                        {t("common.refresh")}
                                    </button>
                                    <span className="text-[10px] text-muted-foreground">{error}</span>
                                </div>
                            </div>
                        )}

                        {!isLoading && filteredCollections.map((collection) => {
                            const collectionName = getCollectionName(collection);
                            const isSelected = normalizeCollection(collectionName) === normalizeCollection(value);

                            return (
                                <button
                                    key={collection.id}
                                    type="button"
                                    onMouseDown={(event) => event.preventDefault()}
                                    onClick={() => commitCollection(collectionName)}
                                    className="flex w-full items-center justify-between rounded px-3 py-2 text-left text-xs transition-colors hover:bg-muted"
                                >
                                    <span className="flex items-center gap-2 truncate">
                                        <Hash className="h-3.5 w-3.5 text-muted-foreground" />
                                        <span className="truncate">{collectionName}</span>
                                    </span>
                                    {isSelected && <Check className="h-3.5 w-3.5 text-primary" />}
                                </button>
                            );
                        })}

                        {!isLoading && !error && filteredCollections.length === 0 && (
                            <div className="px-3 py-2 text-xs text-muted-foreground">
                                {t("dashboard.upload.noCollections")}
                            </div>
                        )}
                    </div>

                    <div className="border-t border-border bg-muted/20 px-3 py-2 text-[10px] text-muted-foreground">
                        {t("dashboard.upload.collectionHint")}
                    </div>
                </div>
            )}
        </div>
    );
}
