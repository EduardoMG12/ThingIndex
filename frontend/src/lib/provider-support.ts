import type { ProviderSupport } from "@/lib/api/providers";

export function getProviderSupportLabel(level?: ProviderSupport["support_level"] | null) {
    switch (level) {
        case "READY":
            return "Ready for main runtime";
        case "PARTIAL":
            return "Partial runtime support";
        case "EXPERIMENTAL":
            return "Experimental bridge support";
        default:
            return "Support status unavailable";
    }
}

export function getProviderSupportBadgeClass(level?: ProviderSupport["support_level"] | null) {
    switch (level) {
        case "READY":
            return "border-emerald-500/25 bg-emerald-500/10 text-emerald-300";
        case "PARTIAL":
            return "border-amber-500/25 bg-amber-500/10 text-amber-300";
        case "EXPERIMENTAL":
            return "border-fuchsia-500/25 bg-fuchsia-500/10 text-fuchsia-300";
        default:
            return "border-border bg-background/70 text-muted-foreground";
    }
}

export function formatProviderSupportTransport(transport?: ProviderSupport["transport"] | null) {
    switch (transport) {
        case "NATIVE_OPENAI":
            return "Native OpenAI path";
        case "NATIVE_OLLAMA":
            return "Native Ollama path";
        case "OPENAI_COMPATIBLE":
            return "Shared OpenAI-compatible path";
        case "OPENAI_COMPATIBLE_BRIDGE":
            return "OpenAI-compatible bridge";
        default:
            return "Unknown transport";
    }
}

export function getProviderSupportCapabilityState(value: boolean) {
    return value ? "Yes" : "No";
}
