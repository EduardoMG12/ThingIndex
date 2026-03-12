"use client";

import { useEffect, useRef, useState } from "react";
import { Check, Hash, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Input } from "@/components/ui/input";
import { useCollections } from "@/hooks/use-collections";
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
    const [query, setQuery] = useState(value);
    const [isOpen, setIsOpen] = useState(false);
    const rootRef = useRef<HTMLDivElement | null>(null);
    const { collections, isLoading, error, reloadCollections } = useCollections();

    useEffect(() => {
        setQuery(value);
    }, [value]);

    useEffect(() => {
        if (!error) {
            return;
        }

        toast.error(error);
    }, [error]);

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

        return normalizeCollection(collection).includes(normalizedQuery);
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
                    void reloadCollections();
                }}
                onBlur={() => commitCollection(query)}
                onKeyDown={(event) => {
                    if (event.key === "Enter") {
                        event.preventDefault();

                        if (filteredCollections.length > 0) {
                            commitCollection(filteredCollections[0]);
                            return;
                        }

                        commitCollection(query);
                    }

                    if (event.key === "Escape") {
                        setIsOpen(false);
                    }
                }}
                placeholder="Selecione ou digite uma collection"
                className={cn("h-7 bg-background font-mono text-xs", className)}
            />

            {isOpen && (
                <div className="absolute left-0 right-0 top-[calc(100%+0.35rem)] z-30 overflow-hidden rounded-md border border-border bg-background shadow-xl">
                    <div className="max-h-64 overflow-y-auto p-1">
                        {isLoading && (
                            <div className="flex items-center gap-2 px-3 py-2 text-xs text-muted-foreground">
                                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                Carregando collections...
                            </div>
                        )}

                        {!isLoading && filteredCollections.map((collection) => {
                            const isSelected = normalizeCollection(collection) === normalizeCollection(value);

                            return (
                                <button
                                    key={collection}
                                    type="button"
                                    onMouseDown={(event) => event.preventDefault()}
                                    onClick={() => commitCollection(collection)}
                                    className="flex w-full items-center justify-between rounded px-3 py-2 text-left text-xs transition-colors hover:bg-muted"
                                >
                                    <span className="flex items-center gap-2 truncate">
                                        <Hash className="h-3.5 w-3.5 text-muted-foreground" />
                                        <span className="truncate">{collection}</span>
                                    </span>
                                    {isSelected && <Check className="h-3.5 w-3.5 text-primary" />}
                                </button>
                            );
                        })}

                        {!isLoading && filteredCollections.length === 0 && (
                            <div className="px-3 py-2 text-xs text-muted-foreground">
                                Nenhuma collection encontrada. Pressione Enter para usar uma nova.
                            </div>
                        )}
                    </div>

                    <div className="border-t border-border bg-muted/20 px-3 py-2 text-[10px] text-muted-foreground">
                        O campo aceita uma collection nova, mas o dropdown mostra as que ja existem.
                    </div>
                </div>
            )}
        </div>
    );
}
