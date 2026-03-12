"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Check, FolderOpen, Loader2, Plus } from "lucide-react";
import { toast } from "sonner";
import { Input } from "@/components/ui/input";
import { useCategories } from "@/hooks/use-categories";
import { cn } from "@/lib/utils";

type CategoryComboboxProps = {
    value: string;
    onChange: (value: string) => void;
    className?: string;
};

function normalizeCategoryName(value: string) {
    return value.trim().toLowerCase();
}

export default function CategoryCombobox({ value, onChange, className }: CategoryComboboxProps) {
    const [query, setQuery] = useState(value);
    const [isOpen, setIsOpen] = useState(false);
    const rootRef = useRef<HTMLDivElement | null>(null);
    const { categories, isLoading, error, addCategoryOption } = useCategories();

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

    const filteredCategories = useMemo(() => {
        const normalizedQuery = normalizeCategoryName(query);

        if (!normalizedQuery) {
            return categories;
        }

        return categories.filter((category) =>
            normalizeCategoryName(category.name).includes(normalizedQuery)
        );
    }, [categories, query]);

    const hasExactMatch = useMemo(() => {
        const normalizedQuery = normalizeCategoryName(query);

        if (!normalizedQuery) {
            return false;
        }

        return categories.some(
            (category) => normalizeCategoryName(category.name) === normalizedQuery
        );
    }, [categories, query]);

    const canCreate = query.trim().length > 0 && !hasExactMatch;

    const commitCategory = (nextValue: string, isNew = false) => {
        const trimmedValue = nextValue.trim();

        if (!trimmedValue) {
            onChange("");
            setQuery("");
            setIsOpen(false);
            return;
        }

        if (isNew) {
            addCategoryOption(trimmedValue);
        }

        onChange(trimmedValue);
        setQuery(trimmedValue);
        setIsOpen(false);
    };

    return (
        <div ref={rootRef} className="relative">
            <Input
                value={query}
                onChange={(event) => {
                    setQuery(event.target.value);
                    onChange(event.target.value);
                    setIsOpen(true);
                }}
                onFocus={() => setIsOpen(true)}
                onBlur={() => commitCategory(query, canCreate)}
                onKeyDown={(event) => {
                    if (event.key === "Enter") {
                        event.preventDefault();

                        if (filteredCategories.length > 0) {
                            commitCategory(filteredCategories[0].name);
                            return;
                        }

                        if (canCreate) {
                            commitCategory(query, true);
                        }
                    }

                    if (event.key === "Escape") {
                        setIsOpen(false);
                    }
                }}
                placeholder="Selecione ou crie uma categoria"
                className={cn("h-7 bg-background font-mono text-xs", className)}
            />

            {isOpen && (
                <div className="absolute left-0 right-0 top-[calc(100%+0.35rem)] z-30 overflow-hidden rounded-md border border-border bg-background shadow-xl">
                    <div className="max-h-64 overflow-y-auto p-1">
                        {isLoading && (
                            <div className="flex items-center gap-2 px-3 py-2 text-xs text-muted-foreground">
                                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                Carregando categorias...
                            </div>
                        )}

                        {!isLoading && filteredCategories.map((category) => {
                            const isSelected = normalizeCategoryName(category.name) === normalizeCategoryName(value);

                            return (
                                <button
                                    key={`${category.id}-${category.name}`}
                                    type="button"
                                    onMouseDown={(event) => event.preventDefault()}
                                    onClick={() => commitCategory(category.name)}
                                    className="flex w-full items-center justify-between rounded px-3 py-2 text-left text-xs transition-colors hover:bg-muted"
                                >
                                    <span className="flex items-center gap-2 truncate">
                                        <FolderOpen className="h-3.5 w-3.5 text-muted-foreground" />
                                        <span className="truncate">{category.name}</span>
                                    </span>
                                    {isSelected && <Check className="h-3.5 w-3.5 text-primary" />}
                                </button>
                            );
                        })}

                        {!isLoading && canCreate && (
                            <button
                                type="button"
                                onMouseDown={(event) => event.preventDefault()}
                                onClick={() => commitCategory(query, true)}
                                className="flex w-full items-center gap-2 rounded px-3 py-2 text-left text-xs text-primary transition-colors hover:bg-primary/5"
                            >
                                <Plus className="h-3.5 w-3.5" />
                                Criar categoria &quot;{query.trim()}&quot;
                            </button>
                        )}

                        {!isLoading && filteredCategories.length === 0 && !canCreate && (
                            <div className="px-3 py-2 text-xs text-muted-foreground">
                                Nenhuma categoria encontrada.
                            </div>
                        )}
                    </div>

                    <div className="border-t border-border bg-muted/20 px-3 py-2 text-[10px] text-muted-foreground">
                        Categorias novas serao persistidas quando o documento for enviado.
                    </div>
                </div>
            )}
        </div>
    );
}
