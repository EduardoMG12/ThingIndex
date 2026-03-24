"use client";

import type { ProviderSupport } from "@/lib/api/providers";
import { ProviderBrandMark } from "@/components/providers/ProviderBrandMark";
import { useI18n } from "@/lib/i18n/provider";
import {
    formatProviderSupportSetupLabel,
    formatProviderSupportTransport,
    getProviderSupportCapabilityState,
} from "@/lib/provider-support";
import { ProviderSupportBadge } from "@/components/providers/ProviderSupportBadge";

const LLM_CAPABILITY_ROWS: Record<string, Array<{
    key: keyof ProviderSupport["capabilities"];
    label: string;
}>> = {
    "pt-BR": [
        { key: "supports_connection_test", label: "Teste de conexão" },
        { key: "supports_model_discovery", label: "Descoberta de modelos" },
        { key: "supports_streaming", label: "Streaming" },
        { key: "supports_optional_live_tests", label: "Testes ao vivo opcionais" },
        { key: "requires_explicit_model", label: "Modelo explícito obrigatório" },
        { key: "requires_explicit_base_url", label: "Base URL explícita obrigatória" },
        { key: "requires_openai_compatible_bridge", label: "Bridge compatível com OpenAI obrigatória" },
    ],
    en: [
        { key: "supports_connection_test", label: "Connection test" },
        { key: "supports_model_discovery", label: "Model discovery" },
        { key: "supports_streaming", label: "Streaming" },
        { key: "supports_optional_live_tests", label: "Optional live tests" },
        { key: "requires_explicit_model", label: "Explicit model required" },
        { key: "requires_explicit_base_url", label: "Explicit base URL required" },
        { key: "requires_openai_compatible_bridge", label: "OpenAI-compatible bridge required" },
    ],
    es: [
        { key: "supports_connection_test", label: "Prueba de conexión" },
        { key: "supports_model_discovery", label: "Descubrimiento de modelos" },
        { key: "supports_streaming", label: "Streaming" },
        { key: "supports_optional_live_tests", label: "Pruebas en vivo opcionales" },
        { key: "requires_explicit_model", label: "Modelo explícito obligatorio" },
        { key: "requires_explicit_base_url", label: "Base URL explícita obligatoria" },
        { key: "requires_openai_compatible_bridge", label: "Bridge compatible con OpenAI obligatoria" },
    ],
};

const STORAGE_CAPABILITY_ROWS: Record<string, Array<{
    key: keyof ProviderSupport["capabilities"];
    label: string;
}>> = {
    "pt-BR": [
        { key: "supports_credential_validation", label: "Validação de credenciais" },
        { key: "supports_connection_test", label: "Teste de conexão" },
        { key: "supports_indexing", label: "Indexação documental" },
        { key: "supports_optional_live_tests", label: "Testes ao vivo opcionais" },
    ],
    en: [
        { key: "supports_credential_validation", label: "Credential validation" },
        { key: "supports_connection_test", label: "Connection test" },
        { key: "supports_indexing", label: "Document indexing" },
        { key: "supports_optional_live_tests", label: "Optional live tests" },
    ],
    es: [
        { key: "supports_credential_validation", label: "Validación de credenciales" },
        { key: "supports_connection_test", label: "Prueba de conexión" },
        { key: "supports_indexing", label: "Indexación documental" },
        { key: "supports_optional_live_tests", label: "Pruebas en vivo opcionales" },
    ],
};

const PANEL_COPY = {
    "pt-BR": {
        fallbackTitle: "Perfil de suporte",
        fallbackDescription: "Os metadados de suporte deste provider não ficaram disponíveis neste carregamento.",
        title: "Perfil de suporte do runtime",
        transport: "Transporte",
        suggestedSetup: "Configuração sugerida",
        notes: "Notas operacionais",
    },
    en: {
        fallbackTitle: "Support profile",
        fallbackDescription: "Support metadata is unavailable for this provider in the current page load.",
        title: "Runtime support profile",
        transport: "Transport",
        suggestedSetup: "Suggested setup",
        notes: "Operational notes",
    },
    es: {
        fallbackTitle: "Perfil de soporte",
        fallbackDescription: "Los metadatos de soporte de este provider no estuvieron disponibles en esta carga.",
        title: "Perfil de soporte del runtime",
        transport: "Transporte",
        suggestedSetup: "Configuración sugerida",
        notes: "Notas operativas",
    },
} as const;

export function ProviderSupportPanel({
    support,
}: {
    support: ProviderSupport | null;
}) {
    const { locale } = useI18n();
    const copy = PANEL_COPY[locale];

    if (!support) {
        return (
            <div className="rounded-2xl border border-border bg-background/60 p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{copy.fallbackTitle}</p>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                    {copy.fallbackDescription}
                </p>
            </div>
        );
    }

    const suggestedSetup = support.suggested_setup ?? {};
    const capabilityRows = support.family === "STORAGE" ? STORAGE_CAPABILITY_ROWS[locale] : LLM_CAPABILITY_ROWS[locale];
    const showRawSupportCopy = locale === "en";

    return (
        <div className="rounded-2xl border border-border bg-background/60 p-4">
            <div className="flex items-start gap-3">
                <ProviderBrandMark providerType={support.provider_type} size="sm" />
                <div className="min-w-0 flex-1 space-y-2">
                    <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{copy.title}</p>
                    <div className="flex flex-wrap items-center gap-2">
                        <p className="text-sm font-semibold text-foreground">{support.display_name}</p>
                        <code className="rounded-full border border-border bg-background/80 px-2 py-0.5 text-[10px] font-mono uppercase tracking-[0.18em] text-muted-foreground">
                            {support.provider_type}
                        </code>
                        <ProviderSupportBadge level={support.support_level} />
                    </div>
                </div>
            </div>
            {showRawSupportCopy ? (
                <p className="mt-3 text-sm leading-relaxed text-foreground/85">{support.summary}</p>
            ) : null}

            <div className="mt-4 grid gap-3 md:grid-cols-2">
                <div className="rounded-xl border border-border/80 bg-background/70 px-3 py-3">
                    <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">{copy.transport}</p>
                    <p className="mt-2 text-sm text-foreground">{formatProviderSupportTransport(locale, support.transport)}</p>
                </div>
                {Object.keys(suggestedSetup).length > 0 ? (
                    <div className="rounded-xl border border-border/80 bg-background/70 px-3 py-3">
                        <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">{copy.suggestedSetup}</p>
                        <div className="mt-2 space-y-1 text-sm text-foreground">
                            {Object.entries(suggestedSetup).map(([key, value]) => (
                                <p key={key}>
                                    {formatProviderSupportSetupLabel(locale, key)}: <span className="font-medium">{value}</span>
                                </p>
                            ))}
                        </div>
                    </div>
                ) : null}
            </div>

            <div className="mt-4 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
                {capabilityRows.map((capability) => (
                    <div
                        key={capability.key}
                        className="rounded-xl border border-border/70 bg-background/70 px-3 py-3"
                    >
                        <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                            {capability.label}
                        </p>
                        <p className="mt-2 text-sm text-foreground">
                            {getProviderSupportCapabilityState(locale, support.capabilities[capability.key])}
                        </p>
                    </div>
                ))}
            </div>

            {showRawSupportCopy && support.notes.length > 0 ? (
                <div className="mt-4 rounded-xl border border-border/80 bg-background/70 px-4 py-3">
                    <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">{copy.notes}</p>
                    <ul className="mt-3 space-y-2 text-sm leading-relaxed text-foreground/85">
                        {support.notes.map((note) => (
                            <li key={note}>{note}</li>
                        ))}
                    </ul>
                </div>
            ) : null}
        </div>
    );
}
