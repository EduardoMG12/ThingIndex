import type { LucideIcon } from "lucide-react";

import type { PricingProviderType } from "@/lib/api/pricing";
import type { ProviderFamily, ProviderType } from "@/lib/api/providers";
import type { AppLocale } from "@/lib/i18n/config";
import { getProviderBranding } from "@/lib/provider-branding";

export type ProviderField = {
    key: string;
    label: string;
    placeholder: string;
    type?: "text" | "password" | "textarea";
    optional?: boolean;
    kind?: "text" | "model" | "price";
    suggestions?: string[];
};

/** US$ por 1M de tokens, opcional. Sem ele o custo fica indisponivel. */
export const LLM_PRICING_FIELD_KEYS = ["input_cost_per_million_usd", "output_cost_per_million_usd"] as const;

const llmPricingFields: ProviderField[] = [
    { key: "input_cost_per_million_usd", label: "Price per 1M input tokens (USD)", placeholder: "0.15", optional: true, kind: "price" },
    { key: "output_cost_per_million_usd", label: "Price per 1M output tokens (USD)", placeholder: "0.60", optional: true, kind: "price" },
];

const openAIModelSuggestions = [
    "gpt-4o-mini",
    "gpt-4o",
    "gpt-4.1-mini",
    "gpt-4.1",
];

const geminiModelSuggestions = [
    "gemini-2.5-flash",
    "gemini-2.5-pro",
    "gemini-2.0-flash",
];

const deepSeekModelSuggestions = [
    "deepseek-chat",
    "deepseek-reasoner",
];

const grokModelSuggestions = [
    "grok-2-latest",
    "grok-2-vision-latest",
];

const claudeModelSuggestions = [
    "claude-3-5-sonnet",
    "claude-3-7-sonnet",
    "claude-3-5-haiku",
];

const ollamaModelSuggestions = [
    "deepseek-v3.1:671b",
    "llama3.1:8b",
    "llama3.1:70b",
    "llama3.2",
    "qwen2.5:7b",
    "mistral:7b",
    "deepseek-r1:8b",
    "gemma2:9b",
];

const OLLAMA_CLOUD_BASE_URL = "https://ollama.com/api";

const openRouterModelSuggestions = [
    "openai/gpt-4o-mini",
    "openai/gpt-4.1-mini",
    "anthropic/claude-3.7-sonnet",
    "anthropic/claude-3.5-sonnet",
    "google/gemini-2.5-pro",
    "google/gemini-2.5-flash",
    "meta-llama/llama-3.1-70b-instruct",
];

/**
* Escondidos porque so MinIO, Ollama e OpenAI foram testados; o runtime
* suporta. Reaparecem se ja houver credencial salva.
*/
export const HIDDEN_PROVIDER_TYPES: ReadonlySet<string> = new Set([
    "AWS_S3",
    "CLOUDFLARE_R2",
    "DIGITALOCEAN_SPACES",
    "WASABI",
    "BACKBLAZE_B2",
    "GCP_STORAGE",
    "DEEP_SEEK",
    "GROK",
    "OPENROUTER",
    "GOOGLE_GEMINI",
    "CLAUDE",
]);

export type ProviderDefinition = {
    family: ProviderFamily;
    providerType: ProviderType;
    name: string;
    tagline: string;
    description: string;
    icon: LucideIcon;
    accentClass: string;
    fields: ProviderField[];
};

type LocalizedFieldOverride = {
    label?: string;
    placeholder?: string;
};

type LocalizedProviderCopy = {
    name?: string;
    tagline: string;
    description: string;
};

const LOCALIZED_PROVIDER_COPY: Record<AppLocale, Record<ProviderType, LocalizedProviderCopy>> = {
    "pt-BR": {
        MINIO: {
            name: "MinIO / compatível com S3",
            tagline: "Storage de objetos para os binários documentais",
            description: "Configure endpoint, access key, secret key e bucket para uploads, preview e indexação.",
        },
        AWS_S3: {
            tagline: "Buckets gerenciados na AWS",
            description: "Use um bucket S3 com região definida. Também é possível sobrescrever o endpoint em gateways compatíveis.",
        },
        CLOUDFLARE_R2: {
            tagline: "Storage de objetos R2",
            description: "Conecte a conta Cloudflare R2 com account ID, access key, secret key e bucket.",
        },
        DIGITALOCEAN_SPACES: {
            tagline: "Storage gerenciado compatível com S3",
            description: "Configure endpoint, região, chaves e bucket para usar Spaces como storage ativo dos documentos.",
        },
        WASABI: {
            tagline: "Storage compatível com S3 de baixo custo",
            description: "Conecte um bucket Wasabi com endpoint regional, chaves e nome do bucket.",
        },
        BACKBLAZE_B2: {
            tagline: "Buckets B2 compatíveis com S3",
            description: "Use o Backblaze B2 via endpoint compatível com S3, com região, chaves e bucket.",
        },
        GCP_STORAGE: {
            tagline: "Buckets GCS nativos e suporte a emulador",
            description: "Use um bucket nativo do Google Cloud Storage com projeto, bucket, localização opcional, JSON de service account ou host de emulador em desenvolvimento.",
        },
        OPENAI: {
            tagline: "Configuração padrão compatível com OpenAI",
            description: "Ideal para o fluxo padrão da OpenAI com modelo e base URL opcionais.",
        },
        GOOGLE_GEMINI: {
            tagline: "Gemini por ponte compatível com OpenAI",
            description: "Use quando o endpoint do Gemini estiver exposto por uma superfície compatível com OpenAI.",
        },
        DEEP_SEEK: {
            tagline: "LLM focada em raciocínio",
            description: "Configure a API key da DeepSeek com overrides opcionais de modelo e base URL.",
        },
        GROK: {
            tagline: "Modelos Grok da xAI",
            description: "Conecte o Grok pela API da xAI ou por um proxy compatível.",
        },
        CLAUDE: {
            tagline: "Anthropic ou bridge compatível",
            description: "Use uma bridge compatível com OpenAI para implantações do Claude.",
        },
        OLLAMA: {
            tagline: "Bridge local ou cloud compatível com OpenAI",
            description: "Use endpoints do Ollama com seleção de modelo, token opcional e base URL customizada quando o deployment não for local.",
        },
        OPENROUTER: {
            tagline: "Gateway unificado com vários modelos",
            description: "Roteie indexação e chat pelo OpenRouter com model id nativo do provider e overrides de endpoint quando necessário.",
        },
    },
    en: {
        MINIO: {
            tagline: "Object storage for document binaries",
            description: "Configure endpoint, access key, secret key and bucket for uploads, previews and indexing.",
        },
        AWS_S3: {
            tagline: "Managed Amazon S3 buckets",
            description: "Use an S3 bucket with region-based routing. You can also override the endpoint for custom-compatible gateways.",
        },
        CLOUDFLARE_R2: {
            tagline: "R2 object storage",
            description: "Connect your Cloudflare R2 account by account ID, access key, secret key and bucket.",
        },
        DIGITALOCEAN_SPACES: {
            tagline: "Managed S3-compatible object storage",
            description: "Configure your Spaces endpoint, region, keys and bucket to use DigitalOcean as the active document store.",
        },
        WASABI: {
            tagline: "Low-cost S3-compatible storage",
            description: "Connect a Wasabi bucket with the regional endpoint, keys and bucket name.",
        },
        BACKBLAZE_B2: {
            tagline: "B2 S3-compatible buckets",
            description: "Use Backblaze B2 through its S3-compatible endpoint with region, keys and bucket name.",
        },
        GCP_STORAGE: {
            tagline: "Native GCS buckets and emulator support",
            description: "Use a native Google Cloud Storage bucket with project, bucket, optional location, service-account JSON or a local emulator host for development.",
        },
        OPENAI: {
            tagline: "Default OpenAI-compatible setup",
            description: "Ideal when you want the standard OpenAI API flow with optional custom model and base URL.",
        },
        GOOGLE_GEMINI: {
            tagline: "Gemini through an OpenAI-compatible bridge",
            description: "Use when your Gemini endpoint is exposed behind an OpenAI-compatible API surface.",
        },
        DEEP_SEEK: {
            tagline: "Reasoning-focused LLM provider",
            description: "Configure your DeepSeek API key with optional model and base URL overrides.",
        },
        GROK: {
            tagline: "xAI Grok models",
            description: "Connect Grok through the xAI API or a compatible proxy.",
        },
        CLAUDE: {
            tagline: "Anthropic or compatible bridge",
            description: "Use an OpenAI-compatible bridge for Claude-compatible deployments.",
        },
        OLLAMA: {
            tagline: "OpenAI-compatible local or cloud bridge",
            description: "Use Ollama endpoints with model selection, optional token and a custom base URL when your deployment is not local.",
        },
        OPENROUTER: {
            tagline: "Unified gateway with many upstream models",
            description: "Route indexing and chat through OpenRouter with a provider-native model ID and custom endpoint overrides when needed.",
        },
    },
    es: {
        MINIO: {
            name: "MinIO / compatible con S3",
            tagline: "Storage de objetos para binarios documentales",
            description: "Configura endpoint, access key, secret key y bucket para cargas, preview e indexación.",
        },
        AWS_S3: {
            tagline: "Buckets administrados en AWS",
            description: "Usa un bucket S3 con región definida. También puedes sobrescribir el endpoint en gateways compatibles.",
        },
        CLOUDFLARE_R2: {
            tagline: "Storage de objetos R2",
            description: "Conecta tu cuenta de Cloudflare R2 con account ID, access key, secret key y bucket.",
        },
        DIGITALOCEAN_SPACES: {
            tagline: "Storage administrado compatible con S3",
            description: "Configura endpoint, región, claves y bucket para usar Spaces como storage activo de documentos.",
        },
        WASABI: {
            tagline: "Storage compatible con S3 de bajo costo",
            description: "Conecta un bucket de Wasabi con endpoint regional, claves y nombre del bucket.",
        },
        BACKBLAZE_B2: {
            tagline: "Buckets B2 compatibles con S3",
            description: "Usa Backblaze B2 mediante endpoint compatible con S3, con región, claves y bucket.",
        },
        GCP_STORAGE: {
            tagline: "Buckets GCS nativos y soporte de emulador",
            description: "Usa un bucket nativo de Google Cloud Storage con proyecto, bucket, ubicación opcional, JSON de service account o host de emulador en desarrollo.",
        },
        OPENAI: {
            tagline: "Configuración estándar compatible con OpenAI",
            description: "Ideal para el flujo estándar de OpenAI con modelo y base URL opcionales.",
        },
        GOOGLE_GEMINI: {
            tagline: "Gemini mediante bridge compatible con OpenAI",
            description: "Úsalo cuando tu endpoint de Gemini esté expuesto detrás de una superficie compatible con OpenAI.",
        },
        DEEP_SEEK: {
            tagline: "LLM enfocada en razonamiento",
            description: "Configura la API key de DeepSeek con overrides opcionales de modelo y base URL.",
        },
        GROK: {
            tagline: "Modelos Grok de xAI",
            description: "Conecta Grok mediante la API de xAI o un proxy compatible.",
        },
        CLAUDE: {
            tagline: "Anthropic o bridge compatible",
            description: "Usa un bridge compatible con OpenAI para despliegues de Claude.",
        },
        OLLAMA: {
            tagline: "Bridge local o cloud compatible con OpenAI",
            description: "Usa endpoints de Ollama con selección de modelo, token opcional y base URL personalizada cuando el despliegue no sea local.",
        },
        OPENROUTER: {
            tagline: "Gateway unificado con muchos modelos",
            description: "Enruta indexación y chat por OpenRouter con model ID nativo del provider y overrides de endpoint cuando sea necesario.",
        },
    },
};

const LOCALIZED_FIELD_LABELS: Record<AppLocale, Partial<Record<string, string>>> = {
    "pt-BR": {
        access_key: "Chave de acesso",
        secret_key: "Chave secreta",
        bucket_name: "Bucket",
        use_ssl: "Usar SSL",
        region: "Região",
        account_id: "ID da conta",
        project_id: "ID do projeto",
        location: "Localização",
        credentials_json: "JSON da service account",
        emulator_host: "Host do emulador",
        api_key: "API key",
        model: "Modelo",
        input_cost_per_million_usd: "Preço por 1 milhão de tokens de entrada (US$)",
        output_cost_per_million_usd: "Preço por 1 milhão de tokens de saída (US$)",
    },
    en: {
        access_key: "Access key",
        secret_key: "Secret key",
        bucket_name: "Bucket name",
        use_ssl: "Use SSL",
        region: "Region",
        account_id: "Account ID",
        project_id: "Project ID",
        location: "Location",
        credentials_json: "Service account JSON",
        emulator_host: "Emulator host",
        api_key: "API key",
        model: "Model",
        input_cost_per_million_usd: "Price per 1 million input tokens (US$)",
        output_cost_per_million_usd: "Price per 1 million output tokens (US$)",
    },
    es: {
        access_key: "Clave de acceso",
        secret_key: "Clave secreta",
        bucket_name: "Bucket",
        use_ssl: "Usar SSL",
        region: "Región",
        account_id: "ID de la cuenta",
        project_id: "ID del proyecto",
        location: "Ubicación",
        credentials_json: "JSON de service account",
        emulator_host: "Host del emulador",
        api_key: "API key",
        model: "Modelo",
        input_cost_per_million_usd: "Precio por 1 millón de tokens de entrada (US$)",
        output_cost_per_million_usd: "Precio por 1 millón de tokens de salida (US$)",
    },
};

const LOCALIZED_FIELD_PLACEHOLDERS: Record<AppLocale, Partial<Record<string, string>>> = {
    "pt-BR": {
        access_key: "Chave de acesso",
        secret_key: "Chave secreta",
        use_ssl: "true ou false",
        account_id: "ID da conta",
        credentials_json: "{\"type\":\"service_account\",...}",
        api_key: "API key",
    },
    en: {
        access_key: "Access key",
        secret_key: "Secret key",
        use_ssl: "true or false",
        account_id: "Account ID",
        credentials_json: "{\"type\":\"service_account\",...}",
        api_key: "API key",
    },
    es: {
        access_key: "Clave de acceso",
        secret_key: "Clave secreta",
        use_ssl: "true o false",
        account_id: "ID de la cuenta",
        credentials_json: "{\"type\":\"service_account\",...}",
        api_key: "API key",
    },
};

const LOCALIZED_FIELD_OVERRIDES: Record<AppLocale, Partial<Record<`${ProviderType}:${string}`, LocalizedFieldOverride>>> = {
    "pt-BR": {
        "AWS_S3:endpoint": { label: "Endpoint customizado", placeholder: "Sobrescrita opcional" },
        "CLOUDFLARE_R2:endpoint": { label: "Endpoint customizado", placeholder: "Sobrescrita opcional" },
        "GOOGLE_GEMINI:base_url": { placeholder: "Endpoint compatível com OpenAI" },
        "CLAUDE:base_url": { placeholder: "Endpoint compatível com OpenAI" },
        "OLLAMA:api_key": { placeholder: "Obrigatória no Ollama Cloud" },
    },
    en: {
        "AWS_S3:endpoint": { label: "Custom endpoint", placeholder: "Optional override" },
        "CLOUDFLARE_R2:endpoint": { label: "Custom endpoint", placeholder: "Optional override" },
        "GOOGLE_GEMINI:base_url": { placeholder: "OpenAI-compatible endpoint" },
        "CLAUDE:base_url": { placeholder: "OpenAI-compatible endpoint" },
        "OLLAMA:api_key": { placeholder: "Required for Ollama Cloud" },
    },
    es: {
        "AWS_S3:endpoint": { label: "Endpoint personalizado", placeholder: "Sobrescritura opcional" },
        "CLOUDFLARE_R2:endpoint": { label: "Endpoint personalizado", placeholder: "Sobrescritura opcional" },
        "GOOGLE_GEMINI:base_url": { placeholder: "Endpoint compatible con OpenAI" },
        "CLAUDE:base_url": { placeholder: "Endpoint compatible con OpenAI" },
        "OLLAMA:api_key": { placeholder: "Obligatoria en Ollama Cloud" },
    },
};

function withBranding(definition: Omit<ProviderDefinition, "icon" | "accentClass">): ProviderDefinition {
    const branding = getProviderBranding(definition.providerType);

    return {
        ...definition,
        icon: branding.icon,
        accentClass: branding.accentClass,
    };
}

export const PRICING_PROVIDERS: Array<{ providerType: PricingProviderType; name: string; sourceURL: string }> = [
    { providerType: "OPENAI", name: "OpenAI API Pricing", sourceURL: "https://developers.openai.com/api/pricing/" },
    { providerType: "DEEP_SEEK", name: "DeepSeek Pricing", sourceURL: "https://deepseek.ai/pricing" },
    { providerType: "GOOGLE_GEMINI", name: "Gemini API Pricing", sourceURL: "https://ai.google.dev/gemini-api/docs/pricing" },
    { providerType: "OLLAMA", name: "Ollama Pricing", sourceURL: "https://ollama.com/pricing" },
];

export const PROVIDERS: ProviderDefinition[] = [
    withBranding({
        family: "STORAGE",
        providerType: "MINIO",
        name: "MinIO / S3-compatible",
        tagline: "Object storage for document binaries",
        description: "Configure endpoint, access key, secret key and bucket for uploads, previews and indexing.",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "minio.example.com:9000" },
            { key: "access_key", label: "Access key", placeholder: "Access key" },
            { key: "secret_key", label: "Secret key", placeholder: "Secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
            { key: "use_ssl", label: "Use SSL", placeholder: "true or false", optional: true },
        ],
    }),
    withBranding({
        family: "STORAGE",
        providerType: "AWS_S3",
        name: "AWS S3",
        tagline: "Managed Amazon S3 buckets",
        description: "Use an S3 bucket with region-based routing. You can also override the endpoint for custom-compatible gateways.",
        fields: [
            { key: "access_key", label: "Access key", placeholder: "AWS access key" },
            { key: "secret_key", label: "Secret key", placeholder: "AWS secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "my-documents-bucket" },
            { key: "region", label: "Region", placeholder: "us-east-1" },
            { key: "endpoint", label: "Custom endpoint", placeholder: "Optional override", optional: true },
        ],
    }),
    withBranding({
        family: "STORAGE",
        providerType: "CLOUDFLARE_R2",
        name: "Cloudflare R2",
        tagline: "R2 object storage",
        description: "Connect your Cloudflare R2 account by account ID, access key, secret key and bucket.",
        fields: [
            { key: "account_id", label: "Account ID", placeholder: "Cloudflare account ID" },
            { key: "access_key", label: "Access key", placeholder: "R2 access key" },
            { key: "secret_key", label: "Secret key", placeholder: "R2 secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
            { key: "endpoint", label: "Custom endpoint", placeholder: "Optional override", optional: true },
        ],
    }),
    withBranding({
        family: "STORAGE",
        providerType: "DIGITALOCEAN_SPACES",
        name: "DigitalOcean Spaces",
        tagline: "Managed S3-compatible object storage",
        description: "Configure your Spaces endpoint, region, keys and bucket to use DigitalOcean as the active document store.",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "nyc3.digitaloceanspaces.com" },
            { key: "region", label: "Region", placeholder: "nyc3" },
            { key: "access_key", label: "Access key", placeholder: "Spaces access key" },
            { key: "secret_key", label: "Secret key", placeholder: "Spaces secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
        ],
    }),
    withBranding({
        family: "STORAGE",
        providerType: "WASABI",
        name: "Wasabi",
        tagline: "Low-cost S3-compatible storage",
        description: "Connect a Wasabi bucket with the regional endpoint, keys and bucket name.",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "s3.us-east-1.wasabisys.com" },
            { key: "region", label: "Region", placeholder: "us-east-1" },
            { key: "access_key", label: "Access key", placeholder: "Wasabi access key" },
            { key: "secret_key", label: "Secret key", placeholder: "Wasabi secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
        ],
    }),
    withBranding({
        family: "STORAGE",
        providerType: "BACKBLAZE_B2",
        name: "Backblaze B2",
        tagline: "B2 S3-compatible buckets",
        description: "Use Backblaze B2 through its S3-compatible endpoint with region, keys and bucket name.",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "s3.us-west-004.backblazeb2.com" },
            { key: "region", label: "Region", placeholder: "us-west-004" },
            { key: "access_key", label: "Access key", placeholder: "B2 key ID" },
            { key: "secret_key", label: "Secret key", placeholder: "B2 application key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
        ],
    }),
    withBranding({
        family: "STORAGE",
        providerType: "GCP_STORAGE",
        name: "Google Cloud Storage",
        tagline: "Native GCS buckets and emulator support",
        description: "Use a native Google Cloud Storage bucket with project, bucket, optional location, service-account JSON or a local emulator host for development.",
        fields: [
            { key: "project_id", label: "Project ID", placeholder: "thingindex-dev" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
            { key: "location", label: "Location", placeholder: "us-central1", optional: true },
            { key: "credentials_json", label: "Service account JSON", placeholder: "{\"type\":\"service_account\",...}", type: "textarea", optional: true },
            { key: "emulator_host", label: "Emulator host", placeholder: "127.0.0.1:4443", optional: true },
        ],
    }),
    withBranding({
        family: "LLM",
        providerType: "OPENAI",
        name: "OpenAI",
        tagline: "Default OpenAI-compatible setup",
        description: "Ideal when you want the standard OpenAI API flow with optional custom model/base URL.",
        fields: [
            { key: "api_key", label: "API key", placeholder: "sk-...", type: "password" },
            { key: "model", label: "Model", placeholder: "gpt-4o-mini", optional: true, kind: "model", suggestions: openAIModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "https://api.openai.com/v1", optional: true },
            ...llmPricingFields,
        ],
    }),
    withBranding({
        family: "LLM",
        providerType: "GOOGLE_GEMINI",
        name: "Google Gemini",
        tagline: "Gemini through an OpenAI-compatible bridge",
        description: "Use when your Gemini endpoint is exposed behind an OpenAI-compatible API surface.",
        fields: [
            { key: "api_key", label: "API key", placeholder: "Gemini API key", type: "password" },
            { key: "model", label: "Model", placeholder: "gemini-2.5-flash", kind: "model", suggestions: geminiModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "OpenAI-compatible endpoint" },
            ...llmPricingFields,
        ],
    }),
    withBranding({
        family: "LLM",
        providerType: "DEEP_SEEK",
        name: "DeepSeek",
        tagline: "Reasoning-focused LLM provider",
        description: "Configure your DeepSeek API key with optional model and base URL overrides.",
        fields: [
            { key: "api_key", label: "API key", placeholder: "DeepSeek API key", type: "password" },
            { key: "model", label: "Model", placeholder: "deepseek-chat", optional: true, kind: "model", suggestions: deepSeekModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "https://api.deepseek.com", optional: true },
            ...llmPricingFields,
        ],
    }),
    withBranding({
        family: "LLM",
        providerType: "GROK",
        name: "Grok",
        tagline: "xAI Grok models",
        description: "Connect Grok through the xAI API or a compatible proxy.",
        fields: [
            { key: "api_key", label: "API key", placeholder: "xAI API key", type: "password" },
            { key: "model", label: "Model", placeholder: "grok-2-latest", optional: true, kind: "model", suggestions: grokModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "https://api.x.ai/v1", optional: true },
            ...llmPricingFields,
        ],
    }),
    withBranding({
        family: "LLM",
        providerType: "CLAUDE",
        name: "Claude",
        tagline: "Anthropic or compatible bridge",
        description: "Use an OpenAI-compatible bridge for Claude-compatible deployments.",
        fields: [
            { key: "api_key", label: "API key", placeholder: "Anthropic API key", type: "password" },
            { key: "model", label: "Model", placeholder: "claude-3-5-sonnet", kind: "model", suggestions: claudeModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "OpenAI-compatible endpoint" },
            ...llmPricingFields,
        ],
    }),
    withBranding({
        family: "LLM",
        providerType: "OLLAMA",
        name: "Ollama",
        tagline: "OpenAI-compatible local or cloud bridge",
        description: "Use Ollama endpoints with model selection, optional token and a custom base URL when your deployment is not local.",
        fields: [
            { key: "api_key", label: "API key", placeholder: "Required for Ollama Cloud", type: "password", optional: true },
            { key: "model", label: "Model", placeholder: "deepseek-v3.1:671b", kind: "model", suggestions: ollamaModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: `${OLLAMA_CLOUD_BASE_URL} or http://host.docker.internal:11434/v1`, optional: true },
            ...llmPricingFields,
        ],
    }),
    withBranding({
        family: "LLM",
        providerType: "OPENROUTER",
        name: "OpenRouter",
        tagline: "Unified gateway with many upstream models",
        description: "Route indexing and chat through OpenRouter with a provider-native model id and custom endpoint overrides when needed.",
        fields: [
            { key: "api_key", label: "API key", placeholder: "OpenRouter API key", type: "password" },
            { key: "model", label: "Model", placeholder: "openai/gpt-4o-mini", kind: "model", suggestions: openRouterModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "https://openrouter.ai/api/v1", optional: true },
            ...llmPricingFields,
        ],
    }),
];

export function getLocalizedProviders(locale: AppLocale, configuredTypes: readonly string[] = []): ProviderDefinition[] {
    const copy = LOCALIZED_PROVIDER_COPY[locale];
    const fieldLabels = LOCALIZED_FIELD_LABELS[locale];
    const fieldPlaceholders = LOCALIZED_FIELD_PLACEHOLDERS[locale];
    const fieldOverrides = LOCALIZED_FIELD_OVERRIDES[locale];
    const configured = new Set(configuredTypes);

    return PROVIDERS.filter((provider) => !HIDDEN_PROVIDER_TYPES.has(provider.providerType) || configured.has(provider.providerType)).map((provider) => {
        const localizedProvider = copy[provider.providerType];

        return {
            ...provider,
            name: localizedProvider.name ?? provider.name,
            tagline: localizedProvider.tagline,
            description: localizedProvider.description,
            fields: provider.fields.map((field) => {
                const override = fieldOverrides[`${provider.providerType}:${field.key}`];

                return {
                    ...field,
                    label: override?.label ?? fieldLabels[field.key] ?? field.label,
                    placeholder: override?.placeholder ?? fieldPlaceholders[field.key] ?? field.placeholder,
                };
            }),
        };
    });
}

export function getProvidersByFamily(family: ProviderFamily) {
    return PROVIDERS.filter((provider) => provider.family === family);
}

export function getInitialFormState() {
    const defaultFieldValues: Partial<Record<ProviderType, Record<string, string>>> = {
        OLLAMA: {
            base_url: OLLAMA_CLOUD_BASE_URL,
        },
    };

    return PROVIDERS.reduce<Record<ProviderType, Record<string, string>>>((state, provider) => {
        state[provider.providerType] = provider.fields.reduce<Record<string, string>>((fieldState, field) => {
            fieldState[field.key] = defaultFieldValues[provider.providerType]?.[field.key] ?? "";
            return fieldState;
        }, {});
        return state;
    }, {} as Record<ProviderType, Record<string, string>>);
}

export function getInitialLLMSelectionState() {
    return PROVIDERS.reduce<Record<ProviderType, { enabledForIndexing: boolean; enabledForChat: boolean }>>((state, provider) => {
        state[provider.providerType] = {
            enabledForIndexing: false,
            enabledForChat: false,
        };
        return state;
    }, {} as Record<ProviderType, { enabledForIndexing: boolean; enabledForChat: boolean }>);
}
