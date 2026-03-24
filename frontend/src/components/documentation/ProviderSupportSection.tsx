"use client";

import { localizeProviderSupportItems } from "@/components/documentation/documentation-resource-localization";
import { DocumentationMetricCard } from "@/components/documentation/DocumentationMetricCard";
import { DocumentationStatusMessage } from "@/components/documentation/DocumentationStatusMessage";
import { useDocumentationResource } from "@/components/documentation/useDocumentationResource";
import { ProviderSupportMatrix } from "@/components/providers/ProviderSupportMatrix";
import { fetchProviderSupportMatrix, type ProviderSupport } from "@/lib/api/providers";
import { useI18n } from "@/lib/i18n/provider";

type DocumentationProviderSupportSnapshot = {
    storageItems: ProviderSupport[];
    llmItems: ProviderSupport[];
};

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
    const { locale } = useI18n();
    const copy = locale === "pt-BR"
        ? {
            fallbackError: "Nao foi possivel carregar a matriz publica de suporte dos provedores.",
            intro:
                "As matrizes abaixo deixam explícito o nível real de suporte dos provedores de armazenamento e LLM. O objetivo é separar o que já está pronto para o runtime principal, o que funciona por um caminho compartilhado e o que ainda depende de bridge ou validação adicional.",
            loading: "Carregando matriz pública de suporte dos provedores...",
                groups: [
                    {
                        key: "storage",
                        title: "Suporte de runtime para armazenamento",
                        description: "Backends concretos de armazenamento disponíveis para upload, preview, download e indexação.",
                    },
                    {
                        key: "llm",
                        title: "Suporte de runtime para LLM",
                        description: "Caminhos de inferência disponíveis para enriquecimento, descoberta de modelos e chat grounded.",
                    },
                ] as const,
            metrics: {
                ready: "Pronto",
                partial: "Parcial",
                experimental: "Experimental",
                readyDetail: "Caminhos principais do runtime",
                partialDetail: "Suporte por adaptador compartilhado",
                experimentalDetail: "Caminhos dependentes de bridge",
            },
        }
        : locale === "es"
            ? {
                fallbackError: "No se pudo cargar la matriz pública de soporte de providers.",
                intro:
                    "Las matrices siguientes dejan explícito el nivel real de soporte de los providers de storage y LLM. El objetivo es separar lo que ya está listo para el runtime principal, lo que funciona por una vía compartida y lo que todavía depende de bridge o validación adicional.",
                loading: "Cargando matriz pública de soporte de providers...",
                groups: [
                    {
                        key: "storage",
                        title: "Soporte runtime de storage",
                        description: "Backends concretos de storage disponibles para carga, preview, descarga e indexación.",
                    },
                    {
                        key: "llm",
                        title: "Soporte runtime de LLM",
                        description: "Rutas de inferencia disponibles para enriquecimiento, discovery de modelos y chat grounded.",
                    },
                ] as const,
                metrics: {
                    ready: "Listo",
                    partial: "Parcial",
                    experimental: "Experimental",
                    readyDetail: "Rutas principales del runtime",
                    partialDetail: "Soporte por adaptador compartido",
                    experimentalDetail: "Rutas dependientes de bridge",
                },
            }
            : {
                fallbackError: "Could not load the provider support matrix.",
                intro:
                    "The matrices below make the actual support level of storage and LLM providers explicit. The goal is to separate what is already ready for the main runtime, what works through a shared path, and what still depends on a bridge or additional validation.",
                loading: "Loading provider support matrix...",
                groups: [
                    {
                        key: "storage",
                        title: "Storage runtime support",
                        description: "Concrete storage backends available for uploads, previews, downloads, and indexing.",
                    },
                    {
                        key: "llm",
                        title: "LLM runtime support",
                        description: "Inference paths available for indexing enrichment, model discovery, and grounded chat.",
                    },
                ] as const,
                metrics: {
                    ready: "Ready",
                    partial: "Partial",
                    experimental: "Experimental",
                    readyDetail: "Main runtime paths",
                    partialDetail: "Shared adapter support",
                    experimentalDetail: "Bridge-only paths",
                },
            };
    const {
        resource,
        isLoading,
        error,
    } = useDocumentationResource<DocumentationProviderSupportSnapshot>(
        fetchDocumentationProviderSupport,
        copy.fallbackError
    );

    return (
        <div className="space-y-5">
            <p>{copy.intro}</p>

            {isLoading ? (
                <DocumentationStatusMessage>
                    {copy.loading}
                </DocumentationStatusMessage>
            ) : error ? (
                <DocumentationStatusMessage tone="error">{error}</DocumentationStatusMessage>
            ) : (
                <div className="space-y-8">
                    {copy.groups.map((section) => {
                        const items = localizeProviderSupportItems(
                            section.key === "storage" ? resource?.storageItems ?? [] : resource?.llmItems ?? [],
                            locale
                        );

                        return (
                            <div key={section.title} className="space-y-4">
                                <div>
                                    <h3 className="text-lg font-semibold text-foreground">{section.title}</h3>
                                    <p className="mt-1 text-sm text-muted-foreground">{section.description}</p>
                                </div>

                                <div className="grid gap-4 md:grid-cols-3">
                                    {[
                                        { label: copy.metrics.ready, value: countProvidersBySupportLevel(items, "READY"), detail: copy.metrics.readyDetail },
                                        { label: copy.metrics.partial, value: countProvidersBySupportLevel(items, "PARTIAL"), detail: copy.metrics.partialDetail },
                                        { label: copy.metrics.experimental, value: countProvidersBySupportLevel(items, "EXPERIMENTAL"), detail: copy.metrics.experimentalDetail },
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
