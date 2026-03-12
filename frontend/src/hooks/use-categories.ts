"use client";

import { useEffect, useState } from "react";
import { createCategory, fetchCategories, type Category } from "@/lib/api/categories";

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

    const mergeCategory = (category: Category) => {
        setCategories((current) => {
            const alreadyExists = current.some(
                (item) => item.name.trim().toLowerCase() === category.name.trim().toLowerCase()
            );

            if (alreadyExists) {
                return current.map((item) =>
                    item.name.trim().toLowerCase() === category.name.trim().toLowerCase() ? category : item
                );
            }

            return [category, ...current];
        });
    };

    const createAndStoreCategory = async (name: string) => {
        const category = await createCategory(name.trim());
        mergeCategory(category);
        return category;
    };

    return {
        categories,
        isLoading,
        error,
        createAndStoreCategory,
    };
}
