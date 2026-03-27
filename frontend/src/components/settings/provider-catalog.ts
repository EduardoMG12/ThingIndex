import { Bot, Database, ServerCog, ShieldCheck, Sparkles, type LucideIcon } from "lucide-react";

import type { PricingProviderType } from "@/lib/api/pricing";
import type { ProviderFamily, ProviderType } from "@/lib/api/providers";

export type ProviderField = {
    key: string;
    label: string;
    placeholder: string;
    type?: "text" | "password";
    optional?: boolean;
    kind?: "text" | "model";
    suggestions?: string[];
};

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

const openRouterModelSuggestions = [
    "openai/gpt-4o-mini",
    "openai/gpt-4.1-mini",
    "anthropic/claude-3.7-sonnet",
    "anthropic/claude-3.5-sonnet",
    "google/gemini-2.5-pro",
    "google/gemini-2.5-flash",
    "meta-llama/llama-3.1-70b-instruct",
];

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

export const PRICING_PROVIDERS: Array<{ providerType: PricingProviderType; name: string; sourceURL: string }> = [
    { providerType: "OPENAI", name: "OpenAI API Pricing", sourceURL: "https://developers.openai.com/api/pricing/" },
    { providerType: "DEEP_SEEK", name: "DeepSeek Pricing", sourceURL: "https://deepseek.ai/pricing" },
    { providerType: "GOOGLE_GEMINI", name: "Gemini API Pricing", sourceURL: "https://ai.google.dev/gemini-api/docs/pricing" },
    { providerType: "OLLAMA", name: "Ollama Pricing", sourceURL: "https://ollama.com/pricing" },
];

export const PROVIDERS: ProviderDefinition[] = [
    {
        family: "STORAGE",
        providerType: "MINIO",
        name: "MinIO / S3-compatible",
        tagline: "Object storage for document binaries",
        description: "Configure endpoint, access key, secret key and bucket for uploads, previews and indexing.",
        icon: Database,
        accentClass: "from-amber-500/20 via-orange-500/10 to-transparent",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "minio.example.com:9000" },
            { key: "access_key", label: "Access key", placeholder: "Access key" },
            { key: "secret_key", label: "Secret key", placeholder: "Secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
            { key: "use_ssl", label: "Use SSL", placeholder: "true or false", optional: true },
        ],
    },
    {
        family: "STORAGE",
        providerType: "AWS_S3",
        name: "AWS S3",
        tagline: "Managed Amazon S3 buckets",
        description: "Use an S3 bucket with region-based routing. You can also override the endpoint for custom-compatible gateways.",
        icon: ServerCog,
        accentClass: "from-orange-500/20 via-amber-500/10 to-transparent",
        fields: [
            { key: "access_key", label: "Access key", placeholder: "AWS access key" },
            { key: "secret_key", label: "Secret key", placeholder: "AWS secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "my-documents-bucket" },
            { key: "region", label: "Region", placeholder: "us-east-1" },
            { key: "endpoint", label: "Custom endpoint", placeholder: "Optional override", optional: true },
        ],
    },
    {
        family: "STORAGE",
        providerType: "CLOUDFLARE_R2",
        name: "Cloudflare R2",
        tagline: "R2 object storage",
        description: "Connect your Cloudflare R2 account by account ID, access key, secret key and bucket.",
        icon: ShieldCheck,
        accentClass: "from-sky-500/20 via-blue-500/10 to-transparent",
        fields: [
            { key: "account_id", label: "Account ID", placeholder: "Cloudflare account ID" },
            { key: "access_key", label: "Access key", placeholder: "R2 access key" },
            { key: "secret_key", label: "Secret key", placeholder: "R2 secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
            { key: "endpoint", label: "Custom endpoint", placeholder: "Optional override", optional: true },
        ],
    },
    {
        family: "STORAGE",
        providerType: "DIGITALOCEAN_SPACES",
        name: "DigitalOcean Spaces",
        tagline: "Managed S3-compatible object storage",
        description: "Configure your Spaces endpoint, region, keys and bucket to use DigitalOcean as the active document store.",
        icon: Database,
        accentClass: "from-cyan-500/20 via-sky-500/10 to-transparent",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "nyc3.digitaloceanspaces.com" },
            { key: "region", label: "Region", placeholder: "nyc3" },
            { key: "access_key", label: "Access key", placeholder: "Spaces access key" },
            { key: "secret_key", label: "Secret key", placeholder: "Spaces secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
        ],
    },
    {
        family: "STORAGE",
        providerType: "WASABI",
        name: "Wasabi",
        tagline: "Low-cost S3-compatible storage",
        description: "Connect a Wasabi bucket with the regional endpoint, keys and bucket name.",
        icon: ServerCog,
        accentClass: "from-lime-500/20 via-emerald-500/10 to-transparent",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "s3.us-east-1.wasabisys.com" },
            { key: "region", label: "Region", placeholder: "us-east-1" },
            { key: "access_key", label: "Access key", placeholder: "Wasabi access key" },
            { key: "secret_key", label: "Secret key", placeholder: "Wasabi secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
        ],
    },
    {
        family: "STORAGE",
        providerType: "BACKBLAZE_B2",
        name: "Backblaze B2",
        tagline: "B2 S3-compatible buckets",
        description: "Use Backblaze B2 through its S3-compatible endpoint with region, keys and bucket name.",
        icon: ShieldCheck,
        accentClass: "from-rose-500/20 via-red-500/10 to-transparent",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "s3.us-west-004.backblazeb2.com" },
            { key: "region", label: "Region", placeholder: "us-west-004" },
            { key: "access_key", label: "Access key", placeholder: "B2 key ID" },
            { key: "secret_key", label: "Secret key", placeholder: "B2 application key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
        ],
    },
    {
        family: "LLM",
        providerType: "OPENAI",
        name: "OpenAI",
        tagline: "Default OpenAI-compatible setup",
        description: "Ideal when you want the standard OpenAI API flow with optional custom model/base URL.",
        icon: Bot,
        accentClass: "from-emerald-500/20 via-teal-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "sk-...", type: "password" },
            { key: "model", label: "Model", placeholder: "gpt-4o-mini", optional: true, kind: "model", suggestions: openAIModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "https://api.openai.com/v1", optional: true },
        ],
    },
    {
        family: "LLM",
        providerType: "GOOGLE_GEMINI",
        name: "Google Gemini",
        tagline: "Gemini through an OpenAI-compatible bridge",
        description: "Use when your Gemini endpoint is exposed behind an OpenAI-compatible API surface.",
        icon: Sparkles,
        accentClass: "from-sky-500/20 via-cyan-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "Gemini API key", type: "password" },
            { key: "model", label: "Model", placeholder: "gemini-2.5-flash", kind: "model", suggestions: geminiModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "OpenAI-compatible endpoint" },
        ],
    },
    {
        family: "LLM",
        providerType: "DEEP_SEEK",
        name: "DeepSeek",
        tagline: "Reasoning-focused LLM provider",
        description: "Configure your DeepSeek API key with optional model and base URL overrides.",
        icon: ServerCog,
        accentClass: "from-indigo-500/20 via-blue-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "DeepSeek API key", type: "password" },
            { key: "model", label: "Model", placeholder: "deepseek-chat", optional: true, kind: "model", suggestions: deepSeekModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "https://api.deepseek.com", optional: true },
        ],
    },
    {
        family: "LLM",
        providerType: "GROK",
        name: "Grok",
        tagline: "xAI Grok models",
        description: "Connect Grok through the xAI API or a compatible proxy.",
        icon: Sparkles,
        accentClass: "from-fuchsia-500/20 via-pink-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "xAI API key", type: "password" },
            { key: "model", label: "Model", placeholder: "grok-2-latest", optional: true, kind: "model", suggestions: grokModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "https://api.x.ai/v1", optional: true },
        ],
    },
    {
        family: "LLM",
        providerType: "CLAUDE",
        name: "Claude",
        tagline: "Anthropic or compatible bridge",
        description: "Use an OpenAI-compatible bridge for Claude-compatible deployments.",
        icon: ShieldCheck,
        accentClass: "from-violet-500/20 via-purple-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "Anthropic API key", type: "password" },
            { key: "model", label: "Model", placeholder: "claude-3-5-sonnet", kind: "model", suggestions: claudeModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "OpenAI-compatible endpoint" },
        ],
    },
    {
        family: "LLM",
        providerType: "OLLAMA",
        name: "Ollama",
        tagline: "OpenAI-compatible local or cloud bridge",
        description: "Use Ollama endpoints with model selection, optional token and a custom base URL when your deployment is not local.",
        icon: ServerCog,
        accentClass: "from-orange-500/20 via-yellow-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "Required for Ollama Cloud", type: "password", optional: true },
            { key: "model", label: "Model", placeholder: "deepseek-v3.1:671b", kind: "model", suggestions: ollamaModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "https://ollama.com/v1 or http://host.docker.internal:11434/v1", optional: true },
        ],
    },
    {
        family: "LLM",
        providerType: "OPENROUTER",
        name: "OpenRouter",
        tagline: "Unified gateway with many upstream models",
        description: "Route indexing and chat through OpenRouter with a provider-native model id and custom endpoint overrides when needed.",
        icon: Bot,
        accentClass: "from-blue-500/20 via-indigo-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "OpenRouter API key", type: "password" },
            { key: "model", label: "Model", placeholder: "openai/gpt-4o-mini", kind: "model", suggestions: openRouterModelSuggestions },
            { key: "base_url", label: "Base URL", placeholder: "https://openrouter.ai/api/v1", optional: true },
        ],
    },
];

export function getProvidersByFamily(family: ProviderFamily) {
    return PROVIDERS.filter((provider) => provider.family === family);
}

export function getInitialFormState() {
    return PROVIDERS.reduce<Record<ProviderType, Record<string, string>>>((state, provider) => {
        state[provider.providerType] = provider.fields.reduce<Record<string, string>>((fieldState, field) => {
            fieldState[field.key] = "";
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
