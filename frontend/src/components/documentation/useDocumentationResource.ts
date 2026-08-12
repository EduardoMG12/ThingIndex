import { useEffect, useState } from "react";

export type DocumentationResourceLoader<T> = (signal: AbortSignal) => Promise<T>;

export function useDocumentationResource<T>(
    loadResource: DocumentationResourceLoader<T>,
    fallbackErrorMessage: string
) {
    const [resource, setResource] = useState<T | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();

        async function load() {
            try {
                setIsLoading(true);
                setError(null);
                setResource(await loadResource(controller.signal));
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }

                setError(err instanceof Error ? err.message : fallbackErrorMessage);
            } finally {
                if (!controller.signal.aborted) {
                    setIsLoading(false);
                }
            }
        }

        void load();

        return () => controller.abort();
    }, [fallbackErrorMessage, loadResource]);

    return { resource, isLoading, error };
}
