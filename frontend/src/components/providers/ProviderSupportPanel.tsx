"use client";

import type { ProviderSupport } from "@/lib/api/providers";
import {
    formatProviderSupportSetupLabel,
    formatProviderSupportTransport,
    getProviderSupportCapabilityState,
} from "@/lib/provider-support";
import { ProviderSupportBadge } from "@/components/providers/ProviderSupportBadge";

const LLM_CAPABILITY_ROWS: Array<{
    key: keyof ProviderSupport["capabilities"];
    label: string;
}> = [
    { key: "supports_connection_test", label: "Connection test" },
    { key: "supports_model_discovery", label: "Model discovery" },
    { key: "supports_streaming", label: "Streaming" },
    { key: "supports_optional_live_tests", label: "Optional live tests" },
    { key: "requires_explicit_model", label: "Explicit model required" },
    { key: "requires_explicit_base_url", label: "Explicit base URL required" },
    { key: "requires_openai_compatible_bridge", label: "OpenAI-compatible bridge required" },
];

const STORAGE_CAPABILITY_ROWS: Array<{
    key: keyof ProviderSupport["capabilities"];
    label: string;
}> = [
    { key: "supports_credential_validation", label: "Credential validation" },
    { key: "supports_connection_test", label: "Connection test" },
    { key: "supports_indexing", label: "Document indexing" },
    { key: "supports_optional_live_tests", label: "Optional live tests" },
];

export function ProviderSupportPanel({
    support,
}: {
    support: ProviderSupport | null;
}) {
    if (!support) {
        return (
            <div className="rounded-2xl border border-border bg-background/60 p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Real support level</p>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                    Support metadata is unavailable for this provider in the current page load.
                </p>
            </div>
        );
    }

    return (
        <div className="rounded-2xl border border-border bg-background/60 p-4">
            <div className="flex flex-wrap items-center gap-2">
                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">Real support level</p>
                <ProviderSupportBadge level={support.support_level} />
            </div>
            <p className="mt-3 text-sm leading-relaxed text-foreground/85">{support.summary}</p>

            <div className="mt-4 grid gap-3 md:grid-cols-2">
                <div className="rounded-xl border border-border/80 bg-background/70 px-3 py-3">
                    <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Transport</p>
                    <p className="mt-2 text-sm text-foreground">{formatProviderSupportTransport(support.transport)}</p>
                </div>
                {Object.keys(support.suggested_setup).length > 0 ? (
                    <div className="rounded-xl border border-border/80 bg-background/70 px-3 py-3">
                        <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Suggested setup</p>
                        <div className="mt-2 space-y-1 text-sm text-foreground">
                            {Object.entries(support.suggested_setup).map(([key, value]) => (
                                <p key={key}>
                                    {formatProviderSupportSetupLabel(key)}: <span className="font-medium">{value}</span>
                                </p>
                            ))}
                        </div>
                    </div>
                ) : null}
            </div>

            <div className="mt-4 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
                {(support.family === "STORAGE" ? STORAGE_CAPABILITY_ROWS : LLM_CAPABILITY_ROWS).map((capability) => (
                    <div
                        key={capability.key}
                        className="rounded-xl border border-border/70 bg-background/70 px-3 py-3"
                    >
                        <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                            {capability.label}
                        </p>
                        <p className="mt-2 text-sm text-foreground">
                            {getProviderSupportCapabilityState(support.capabilities[capability.key])}
                        </p>
                    </div>
                ))}
            </div>

            {support.notes.length > 0 ? (
                <div className="mt-4 rounded-xl border border-border/80 bg-background/70 px-4 py-3">
                    <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Operational notes</p>
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
