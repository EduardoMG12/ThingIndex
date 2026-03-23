"use client";

import { useEffect, useState } from "react";

import { ProviderSupportMatrix } from "@/components/providers/ProviderSupportMatrix";
import { fetchProviderSupportMatrix, type ProviderSupport } from "@/lib/api/providers";

function countByLevel(items: ProviderSupport[], level: ProviderSupport["support_level"]) {
    return items.filter((item) => item.support_level === level).length;
}

export function ProviderSupportSection() {
    const [storageItems, setStorageItems] = useState<ProviderSupport[]>([]);
    const [llmItems, setLLMItems] = useState<ProviderSupport[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();

        async function load() {
            try {
                setIsLoading(true);
                setError(null);
                const [storage, llm] = await Promise.all([
                    fetchProviderSupportMatrix({ family: "STORAGE", signal: controller.signal }),
                    fetchProviderSupportMatrix({ family: "LLM", signal: controller.signal }),
                ]);
                setStorageItems(storage);
                setLLMItems(llm);
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }
                setError(err instanceof Error ? err.message : "Could not load the provider support matrix.");
            } finally {
                if (!controller.signal.aborted) {
                    setIsLoading(false);
                }
            }
        }

        void load();
        return () => controller.abort();
    }, []);

    return (
        <div className="space-y-5">
            <p>
                As matrizes abaixo deixam explícito o nível real de suporte dos providers de storage e LLM. O objetivo é separar
                o que já está pronto para o runtime principal, o que funciona por um caminho compartilhado e o que ainda
                depende de bridge ou validação adicional.
            </p>

            {isLoading ? (
                <div className="rounded-xl border border-border bg-background/50 p-5 text-sm text-muted-foreground">
                    Loading provider support matrix...
                </div>
            ) : error ? (
                <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-5 text-sm text-destructive">
                    {error}
                </div>
            ) : (
                <div className="space-y-8">
                    {[
                        {
                            title: "Storage runtime support",
                            description: "Concrete storage backends available for uploads, previews, downloads and indexing.",
                            items: storageItems,
                        },
                        {
                            title: "LLM runtime support",
                            description: "Inference paths available for indexing enrichment, model discovery and grounded chat.",
                            items: llmItems,
                        },
                    ].map((section) => (
                        <div key={section.title} className="space-y-4">
                            <div>
                                <h3 className="text-lg font-semibold text-foreground">{section.title}</h3>
                                <p className="mt-1 text-sm text-muted-foreground">{section.description}</p>
                            </div>

                            <div className="grid gap-4 md:grid-cols-3">
                                {[
                                    { label: "Ready", value: countByLevel(section.items, "READY"), detail: "Main runtime paths" },
                                    { label: "Partial", value: countByLevel(section.items, "PARTIAL"), detail: "Shared adapter support" },
                                    { label: "Experimental", value: countByLevel(section.items, "EXPERIMENTAL"), detail: "Bridge-only paths" },
                                ].map((item) => (
                                    <div key={`${section.title}-${item.label}`} className="rounded-xl border border-border bg-background/50 p-4">
                                        <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">{item.label}</p>
                                        <p className="mt-2 text-2xl font-semibold text-foreground">{item.value}</p>
                                        <p className="mt-1 text-sm text-muted-foreground">{item.detail}</p>
                                    </div>
                                ))}
                            </div>

                            <ProviderSupportMatrix items={section.items} />
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
