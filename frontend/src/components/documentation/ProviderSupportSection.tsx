"use client";

import { DocumentationMetricCard } from "@/components/documentation/DocumentationMetricCard";
import { DocumentationStatusMessage } from "@/components/documentation/DocumentationStatusMessage";
import { useDocumentationResource } from "@/components/documentation/useDocumentationResource";
import { ProviderSupportMatrix } from "@/components/providers/ProviderSupportMatrix";
import { fetchProviderSupportMatrix, type ProviderSupport } from "@/lib/api/providers";

type DocumentationProviderSupportSnapshot = {
    storageItems: ProviderSupport[];
    llmItems: ProviderSupport[];
};

const PROVIDER_SUPPORT_GROUPS = [
    {
        key: "storage",
        title: "Storage runtime support",
        description: "Concrete storage backends available for uploads, previews, downloads and indexing.",
    },
    {
        key: "llm",
        title: "LLM runtime support",
        description: "Inference paths available for indexing enrichment, model discovery and grounded chat.",
    },
] as const;

async function fetchDocumentationProviderSupport(
    signal: AbortSignal
): Promise<DocumentationProviderSupportSnapshot> {
    const [storageItems, llmItems] = await Promise.all([
        fetchProviderSupportMatrix({ family: "STORAGE", signal }),
        fetchProviderSupportMatrix({ family: "LLM", signal }),
    ]);

    return { storageItems, llmItems };
}

function countProvidersBySupportLevel(
    items: ProviderSupport[],
    level: ProviderSupport["support_level"]
) {
    return items.filter((item) => item.support_level === level).length;
}

export function ProviderSupportSection() {
    const {
        resource,
        isLoading,
        error,
    } = useDocumentationResource<DocumentationProviderSupportSnapshot>(
        fetchDocumentationProviderSupport,
        "Could not load the provider support matrix."
    );

    return (
        <div className="space-y-5">
            <p>
                As matrizes abaixo deixam explícito o nível real de suporte dos providers de storage e LLM. O objetivo é separar
                o que já está pronto para o runtime principal, o que funciona por um caminho compartilhado e o que ainda
                depende de bridge ou validação adicional.
            </p>

            {isLoading ? (
                <DocumentationStatusMessage>
                    Loading provider support matrix...
                </DocumentationStatusMessage>
            ) : error ? (
                <DocumentationStatusMessage tone="error">{error}</DocumentationStatusMessage>
            ) : (
                <div className="space-y-8">
                    {PROVIDER_SUPPORT_GROUPS.map((section) => {
                        const items = section.key === "storage" ? resource?.storageItems ?? [] : resource?.llmItems ?? [];

                        return (
                            <div key={section.title} className="space-y-4">
                                <div>
                                    <h3 className="text-lg font-semibold text-foreground">{section.title}</h3>
                                    <p className="mt-1 text-sm text-muted-foreground">{section.description}</p>
                                </div>

                                <div className="grid gap-4 md:grid-cols-3">
                                    {[
                                        { label: "Ready", value: countProvidersBySupportLevel(items, "READY"), detail: "Main runtime paths" },
                                        { label: "Partial", value: countProvidersBySupportLevel(items, "PARTIAL"), detail: "Shared adapter support" },
                                        { label: "Experimental", value: countProvidersBySupportLevel(items, "EXPERIMENTAL"), detail: "Bridge-only paths" },
                                    ].map((item) => (
                                        <DocumentationMetricCard
                                            key={`${section.title}-${item.label}`}
                                            label={item.label}
                                            value={item.value}
                                            detail={item.detail}
                                        />
                                    ))}
                                </div>

                                <ProviderSupportMatrix items={items} />
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
