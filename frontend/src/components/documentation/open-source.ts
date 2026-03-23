export type DocumentationResource = {
    title: string;
    description: string;
    href?: string;
    pathHint?: string;
};

export type DocumentationStep = {
    title: string;
    command: string;
    detail: string;
};

export type DocumentationTrack = {
    badge: string;
    title: string;
    description: string;
    steps: DocumentationStep[];
    notes: string[];
};

export type DocumentationTroubleshootingItem = {
    title: string;
    detail: string;
};

export function resolveApiReferenceBase(apiBaseUrl?: string): string {
    const normalized = (apiBaseUrl ?? "").trim();
    if (!normalized) {
        return "http://localhost:8000";
    }

    return normalized.replace(/\/api\/v1\/?$/i, "");
}

export function resolveHostedAPIBase(apiBaseUrl?: string): string {
    return `${resolveApiReferenceBase(apiBaseUrl)}/api/v1`;
}

export function getDocumentationResources(apiBaseUrl?: string): DocumentationResource[] {
    const apiReferenceBase = resolveApiReferenceBase(apiBaseUrl);

    return [
        {
            title: "Root README",
            description: "Canonical project overview, runtime scope, and environment variables.",
            pathHint: "README.md",
        },
        {
            title: "Open-source onboarding guide",
            description: "Self-hosted and hosted-API entry paths, wrappers, and troubleshooting.",
            pathHint: "docs/open-source-onboarding.md",
        },
        {
            title: "Scalar API reference",
            description: "Primary interactive API documentation served by the current deployment.",
            href: `${apiReferenceBase}/docs`,
        },
        {
            title: "Swagger UI",
            description: "Fallback interactive viewer when Scalar cannot render.",
            href: `${apiReferenceBase}/swagger/index.html`,
        },
        {
            title: "OpenAPI JSON",
            description: "Raw contract for generators, SDKs, and CI tooling.",
            href: `${apiReferenceBase}/swagger/doc.json`,
        },
        {
            title: "Storage support matrix",
            description: "Public support classification for storage backends, setup hints, and operational notes.",
            href: `${apiReferenceBase}/api/v1/providers/support?family=STORAGE`,
        },
        {
            title: "LLM support matrix",
            description: "Public support classification for LLM providers, bridge requirements, and runtime notes.",
            href: `${apiReferenceBase}/api/v1/providers/support?family=LLM`,
        },
        {
            title: "HTTP request examples",
            description: "Reusable requests for health, auth, collections, documents, runtime overview, and provider support.",
            pathHint: "examples/http/thingindex-api.http",
        },
        {
            title: "Versioned OpenAPI contract",
            description: "Repository-tracked YAML/JSON artifacts used by Scalar, Swagger, and external tooling.",
            pathHint: "api/docs/swagger.yaml",
        },
    ];
}

export function getDocumentationTracks(apiBaseUrl?: string): DocumentationTrack[] {
    const apiReferenceBase = resolveApiReferenceBase(apiBaseUrl);
    const hostedAPIBase = resolveHostedAPIBase(apiBaseUrl);

    return [
        {
            badge: "GitHub checkout",
            title: "Self-hosted runtime",
            description: "Clone the repository when you need the full stack locally: frontend, API, workers, troubleshooting scripts, and versioned docs.",
            steps: [
                {
                    title: "Prepare env and dependencies",
                    command: "./scripts/bootstrap.sh",
                    detail: "Creates .env when missing and installs backend/frontend dependencies.",
                },
                {
                    title: "Start the local stack",
                    command: "./scripts/stack.sh up",
                    detail: "Boots Docker Compose with frontend, API, PostgreSQL, MinIO, and RabbitMQ.",
                },
                {
                    title: "Run a quick diagnosis",
                    command: "./scripts/doctor.sh",
                    detail: "Checks required commands, local files, frontend reachability, and API docs endpoints.",
                },
            ],
            notes: [
                "UI: http://localhost:3000",
                "API base: http://localhost:8000/api/v1",
                "Primary docs: http://localhost:8000/docs",
            ],
        },
        {
            badge: "Current deployment",
            title: "Hosted API consumption",
            description: "Use the current deployment directly when the API is already online and you do not need the monorepo running locally.",
            steps: [
                {
                    title: "Start with the deployment docs",
                    command: `${apiReferenceBase}/docs`,
                    detail: "Open Scalar on the current deployment first; Swagger UI and the raw JSON stay on the same origin.",
                },
                {
                    title: "Use the deployment API base",
                    command: hostedAPIBase,
                    detail: "Point your client, SDK generation, Postman collection, or integration code at this base URL.",
                },
                {
                    title: "Reuse the versioned request examples",
                    command: "examples/http/thingindex-api.http",
                    detail: "If you cloned the repo, only change @apiHost at the top of the file to target the hosted origin.",
                },
            ],
            notes: [
                `Swagger UI fallback: ${apiReferenceBase}/swagger/index.html`,
                `Raw OpenAPI JSON: ${apiReferenceBase}/swagger/doc.json`,
                `Optional repo-side diagnosis: ./scripts/doctor.sh --api-base ${apiReferenceBase}`,
            ],
        },
    ];
}

export const DOCUMENTATION_TROUBLESHOOTING_ITEMS: DocumentationTroubleshootingItem[] = [
    {
        title: "API docs did not load",
        detail: "Open /docs first, then fall back to /swagger/index.html or /swagger/doc.json on the same origin when Scalar is unavailable.",
    },
    {
        title: "Hosted requests are still hitting localhost",
        detail: "Change @apiHost in examples/http/thingindex-api.http or point your client configuration to the deployed origin before retrying.",
    },
    {
        title: "Frontend cannot talk to the API locally",
        detail: "Verify NEXT_PUBLIC_API_URL and API_INTERNAL_URL in .env and rerun ./scripts/doctor.sh.",
    },
    {
        title: "Manual host run cannot reach storage",
        detail: "Use MINIO_ENDPOINT=localhost:9000 on the host and MINIO_ENDPOINT=minio:9000 inside Docker Compose.",
    },
];
