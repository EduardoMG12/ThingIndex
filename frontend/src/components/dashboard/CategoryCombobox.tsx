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
    const [isCreating, setIsCreating] = useState(false);
    const rootRef = useRef<HTMLDivElement | null>(null);
    const { categories, isLoading, error, createAndStoreCategory } = useCategories();

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

    const commitCategory = (nextValue: string) => {
        const trimmedValue = nextValue.trim();

        if (!trimmedValue) {
            onChange("");
            setQuery("");
            setIsOpen(false);
            return;
        }

        onChange(trimmedValue);
        setQuery(trimmedValue);
        setIsOpen(false);
    };

    const handleCreateCategory = async () => {
        const trimmedValue = query.trim();

        if (!trimmedValue || isCreating) {
            return;
        }

        try {
            setIsCreating(true);
            const category = await createAndStoreCategory(trimmedValue);
            commitCategory(category.name);
            toast.success("Categoria criada com sucesso.");
        } catch (err) {
            toast.error(err instanceof Error ? err.message : "Nao foi possivel criar a categoria.");
        } finally {
            setIsCreating(false);
        }
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
                onBlur={() => {
                    if (!canCreate) {
                        commitCategory(query);
                    }
                }}
                onKeyDown={(event) => {
                    if (event.key === "Enter") {
                        event.preventDefault();

                        if (filteredCategories.length > 0) {
                            commitCategory(filteredCategories[0].name);
                            return;
                        }

                        if (canCreate) {
                            void handleCreateCategory();
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
                        {(isLoading || isCreating) && (
                            <div className="flex items-center gap-2 px-3 py-2 text-xs text-muted-foreground">
                                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                {isCreating ? "Criando categoria..." : "Carregando categorias..."}
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
                                disabled={isCreating}
                                onMouseDown={(event) => event.preventDefault()}
                                onClick={() => void handleCreateCategory()}
                                className="flex w-full items-center gap-2 rounded px-3 py-2 text-left text-xs text-primary transition-colors hover:bg-primary/5 disabled:cursor-not-allowed disabled:opacity-60"
                            >
                                {isCreating ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Plus className="h-3.5 w-3.5" />}
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
                        Ao criar, o frontend envia a requisicao imediatamente para o backend.
                    </div>
                </div>
            )}
        </div>
    );
}
