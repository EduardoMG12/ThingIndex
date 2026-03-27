"use client";

import { useEffect, useState } from "react";

import { ProviderSupportMatrix } from "@/components/providers/ProviderSupportMatrix";
import { fetchProviderSupportMatrix, type ProviderSupport } from "@/lib/api/providers";

function countByLevel(items: ProviderSupport[], level: ProviderSupport["support_level"]) {
    return items.filter((item) => item.support_level === level).length;
}

export function ProviderSupportSection() {
    const [items, setItems] = useState<ProviderSupport[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();

        async function load() {
            try {
                setIsLoading(true);
                setError(null);
                setItems(await fetchProviderSupportMatrix({ family: "LLM", signal: controller.signal }));
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
                A matriz abaixo deixa explícito o nível real de suporte de cada provider LLM. O objetivo é separar o que já
                está pronto para o runtime principal, o que funciona por um caminho parcial e o que ainda depende de bridge
                experimental.
            </p>

            <div className="grid gap-4 md:grid-cols-3">
                {[
                    { label: "Ready", value: countByLevel(items, "READY"), detail: "Main runtime paths" },
                    { label: "Partial", value: countByLevel(items, "PARTIAL"), detail: "Shared adapter support" },
                    { label: "Experimental", value: countByLevel(items, "EXPERIMENTAL"), detail: "Bridge-only paths" },
                ].map((item) => (
                    <div key={item.label} className="rounded-xl border border-border bg-background/50 p-4">
                        <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">{item.label}</p>
                        <p className="mt-2 text-2xl font-semibold text-foreground">{item.value}</p>
                        <p className="mt-1 text-sm text-muted-foreground">{item.detail}</p>
                    </div>
                ))}
            </div>

            {isLoading ? (
                <div className="rounded-xl border border-border bg-background/50 p-5 text-sm text-muted-foreground">
                    Loading provider support matrix...
                </div>
            ) : error ? (
                <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-5 text-sm text-destructive">
                    {error}
                </div>
            ) : (
                <ProviderSupportMatrix items={items} />
            )}
        </div>
    );
}
