import type { ProviderRuntimeAssignment, ProviderType } from "@/lib/api/providers";

const PROVIDER_LABELS: Record<ProviderType, string> = {
    MINIO: "MinIO",
    AWS_S3: "AWS S3",
    CLOUDFLARE_R2: "Cloudflare R2",
    DIGITALOCEAN_SPACES: "DigitalOcean Spaces",
    WASABI: "Wasabi",
    BACKBLAZE_B2: "Backblaze B2",
    GCP_STORAGE: "Google Cloud Storage",
    OPENAI: "OpenAI",
    GOOGLE_GEMINI: "Google Gemini",
    DEEP_SEEK: "DeepSeek",
    GROK: "Grok",
    CLAUDE: "Claude",
    OLLAMA: "Ollama",
    OPENROUTER: "OpenRouter",
};

export function formatProviderTypeLabel(providerType?: string | null) {
    if (!providerType) {
        return "-";
    }

    return PROVIDER_LABELS[providerType as ProviderType] ?? providerType.replaceAll("_", " ");
}

export function getProviderRuntimeModel(assignment?: ProviderRuntimeAssignment | null) {
    const model = assignment?.public_metadata?.model?.trim();
    return model ? model : null;
}
