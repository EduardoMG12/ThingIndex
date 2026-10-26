import type { ProviderSupport } from "@/lib/api/providers";
import type { AppLocale } from "@/lib/i18n/config";

const PROVIDER_SUPPORT_LABELS: Record<AppLocale, Record<string, string>> = {
    "pt-BR": {
        READY: "Pronto para o sistema principal",
        PARTIAL: "Suporte parcial",
        EXPERIMENTAL: "Suporte experimental via bridge",
        unknown: "Status de suporte indisponível",
        NATIVE_OPENAI: "Caminho nativo OpenAI",
        NATIVE_OLLAMA: "Caminho nativo Ollama",
        NATIVE_GCS: "Caminho nativo Google Cloud Storage",
        OPENAI_COMPATIBLE: "Caminho compartilhado compatível com OpenAI",
        OPENAI_COMPATIBLE_BRIDGE: "Bridge compatível com OpenAI",
        S3_COMPATIBLE: "Caminho compartilhado compatível com S3",
        unknownTransport: "Transporte desconhecido",
        yes: "Sim",
        no: "Não",
        account_id: "ID da conta",
        base_url: "Base URL",
        bucket_name: "Bucket",
        credentials_json: "JSON de credenciais",
        emulator_host: "Host do emulador",
        endpoint: "Endpoint",
        location: "Localização",
        model: "Modelo",
        project_id: "ID do projeto",
        region: "Região",
    },
    en: {
        READY: "Ready for main runtime",
        PARTIAL: "Partial runtime support",
        EXPERIMENTAL: "Experimental bridge support",
        unknown: "Support status unavailable",
        NATIVE_OPENAI: "Native OpenAI path",
        NATIVE_OLLAMA: "Native Ollama path",
        NATIVE_GCS: "Native Google Cloud Storage path",
        OPENAI_COMPATIBLE: "Shared OpenAI-compatible path",
        OPENAI_COMPATIBLE_BRIDGE: "OpenAI-compatible bridge",
        S3_COMPATIBLE: "Shared S3-compatible path",
        unknownTransport: "Unknown transport",
        yes: "Yes",
        no: "No",
        account_id: "Account ID",
        base_url: "Base URL",
        bucket_name: "Bucket",
        credentials_json: "Credentials JSON",
        emulator_host: "Emulator host",
        endpoint: "Endpoint",
        location: "Location",
        model: "Model",
        project_id: "Project ID",
        region: "Region",
    },
    es: {
        READY: "Listo para el runtime principal",
        PARTIAL: "Soporte parcial de runtime",
        EXPERIMENTAL: "Soporte experimental por bridge",
        unknown: "Estado de soporte no disponible",
        NATIVE_OPENAI: "Ruta nativa OpenAI",
        NATIVE_OLLAMA: "Ruta nativa Ollama",
        NATIVE_GCS: "Ruta nativa Google Cloud Storage",
        OPENAI_COMPATIBLE: "Ruta compartida compatible con OpenAI",
        OPENAI_COMPATIBLE_BRIDGE: "Bridge compatible con OpenAI",
        S3_COMPATIBLE: "Ruta compartida compatible con S3",
        unknownTransport: "Transporte desconocido",
        yes: "Sí",
        no: "No",
        account_id: "ID de la cuenta",
        base_url: "Base URL",
        bucket_name: "Bucket",
        credentials_json: "JSON de credenciales",
        emulator_host: "Host del emulador",
        endpoint: "Endpoint",
        location: "Ubicación",
        model: "Modelo",
        project_id: "ID del proyecto",
        region: "Región",
    },
};

export function getProviderSupportLabel(locale: AppLocale, level?: ProviderSupport["support_level"] | null) {
    const copy = PROVIDER_SUPPORT_LABELS[locale];

    switch (level) {
        case "READY":
            return copy.READY;
        case "PARTIAL":
            return copy.PARTIAL;
        case "EXPERIMENTAL":
            return copy.EXPERIMENTAL;
        default:
            return copy.unknown;
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

export function formatProviderSupportTransport(locale: AppLocale, transport?: ProviderSupport["transport"] | null) {
    const copy = PROVIDER_SUPPORT_LABELS[locale];

    switch (transport) {
        case "NATIVE_OPENAI":
            return copy.NATIVE_OPENAI;
        case "NATIVE_OLLAMA":
            return copy.NATIVE_OLLAMA;
        case "NATIVE_GCS":
            return copy.NATIVE_GCS;
        case "OPENAI_COMPATIBLE":
            return copy.OPENAI_COMPATIBLE;
        case "OPENAI_COMPATIBLE_BRIDGE":
            return copy.OPENAI_COMPATIBLE_BRIDGE;
        case "S3_COMPATIBLE":
            return copy.S3_COMPATIBLE;
        default:
            return copy.unknownTransport;
    }
}

export function getProviderSupportCapabilityState(locale: AppLocale, value: boolean) {
    const copy = PROVIDER_SUPPORT_LABELS[locale];
    return value ? copy.yes : copy.no;
}

const SUPPORT_SETUP_LABELS: Record<string, string> = {
    account_id: "Account ID",
    base_url: "Base URL",
    bucket_name: "Bucket",
    credentials_json: "Credentials JSON",
    emulator_host: "Emulator host",
    endpoint: "Endpoint",
    location: "Location",
    model: "Model",
    project_id: "Project ID",
    region: "Region",
};

export function formatProviderSupportSetupLabel(locale: AppLocale, key: string) {
    const copy = PROVIDER_SUPPORT_LABELS[locale];

    return copy[key] ?? SUPPORT_SETUP_LABELS[key] ?? key.replaceAll("_", " ").replace(/\b\w/g, (value) => value.toUpperCase());
}
