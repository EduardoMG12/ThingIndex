"use client";

import { useEffect, useState } from "react";
import { fetchCategories, type Category } from "@/lib/api/categories";

export function useCategories() {
    const [categories, setCategories] = useState<Category[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();

        async function loadCategories() {
            try {
                setIsLoading(true);
                setError(null);

                const data = await fetchCategories(controller.signal);
                setCategories(data);
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }

                setError(err instanceof Error ? err.message : "Nao foi possivel carregar as categorias.");
            } finally {
                if (!controller.signal.aborted) {
                    setIsLoading(false);
                }
            }
        }

        loadCategories();

        return () => controller.abort();
    }, []);

    const addCategoryOption = (name: string) => {
        const trimmedName = name.trim();

        if (!trimmedName) {
            return;
        }

        setCategories((current) => {
            const alreadyExists = current.some(
                (category) => category.name.trim().toLowerCase() === trimmedName.toLowerCase()
            );

            if (alreadyExists) {
                return current;
            }

            return [
                {
                    id: -Date.now(),
                    user_id: 0,
                    name: trimmedName,
                    description: "Categoria criada localmente e persistida no upload.",
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                },
                ...current,
            ];
        });
    };

    return {
        categories,
        isLoading,
        error,
        addCategoryOption,
    };
}
