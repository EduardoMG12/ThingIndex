import type { AppLocale } from "@/lib/i18n/config";

export type OpenSourceOnboardingResource = {
    title: string;
    description: string;
    href?: string;
    pathHint?: string;
};

export type OpenSourceOnboardingStep = {
    title: string;
    command: string;
    detail: string;
};

export type OpenSourceOnboardingTrack = {
    kind: "self-hosted" | "hosted";
    badge: string;
    title: string;
    description: string;
    steps: OpenSourceOnboardingStep[];
    notes: string[];
};

export type OpenSourceTroubleshootingItem = {
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

export function getDocumentationResources(apiBaseUrl?: string, locale: AppLocale = "en"): OpenSourceOnboardingResource[] {
    const apiReferenceBase = resolveApiReferenceBase(apiBaseUrl);

    if (locale === "pt-BR") {
        return [
            {
                title: "README raiz",
                description: "Visão canônica do projeto, escopo do runtime e variáveis de ambiente.",
                pathHint: "README.md",
            },
            {
                title: "Guia de onboarding de código aberto",
                description: "Caminhos auto-hospedados e de API publicada, scripts utilitários e resolução de problemas.",
                pathHint: "docs/open-source-onboarding.md",
            },
            {
                title: "Referência Scalar da API",
                description: "Documentação interativa principal servida pela implantação atual.",
                href: `${apiReferenceBase}/docs`,
            },
            {
                title: "Swagger UI",
                description: "Visualizador interativo de fallback quando o Scalar não renderiza.",
                href: `${apiReferenceBase}/swagger/index.html`,
            },
            {
                title: "OpenAPI JSON",
                description: "Contrato bruto para geradores, SDKs e ferramentas de CI.",
                href: `${apiReferenceBase}/swagger/doc.json`,
            },
            {
                title: "Matriz de suporte de armazenamento",
                description: "Classificação pública do suporte real de backends de armazenamento, configuração e notas operacionais.",
                href: `${apiReferenceBase}/api/v1/providers/support?family=STORAGE`,
            },
            {
                title: "Matriz de suporte de LLM",
                description: "Classificação pública do suporte real de LLMs, bridges e notas operacionais do runtime.",
                href: `${apiReferenceBase}/api/v1/providers/support?family=LLM`,
            },
            {
                title: "Exemplos de requisições HTTP",
                description: "Requisições reutilizáveis para saúde, autenticação, coleções, documentos, runtime e provedores.",
                pathHint: "examples/http/thingindex-api.http",
            },
            {
                title: "CLI de importação de índice",
                description: "Comando Go para importar exports versionados de índice em um PostgreSQL externo com esquema auto-criado.",
                pathHint: "api/cmd/index-import",
            },
            {
                title: "Contrato OpenAPI versionado",
                description: "Artefatos YAML/JSON versionados no repositório e usados por Scalar, Swagger e ferramental externo.",
                pathHint: "api/docs/swagger.yaml",
            },
        ];
    }

    if (locale === "es") {
        return [
            {
                title: "README raíz",
                description: "Visión canónica del proyecto, alcance del runtime y variables de entorno.",
                pathHint: "README.md",
            },
            {
                title: "Guía de onboarding open source",
                description: "Entradas self-hosted y hosted API, wrappers y troubleshooting.",
                pathHint: "docs/open-source-onboarding.md",
            },
            {
                title: "Referencia Scalar de la API",
                description: "Documentación interactiva principal servida por el deployment actual.",
                href: `${apiReferenceBase}/docs`,
            },
            {
                title: "Swagger UI",
                description: "Visualizador interactivo de fallback cuando Scalar no puede renderizar.",
                href: `${apiReferenceBase}/swagger/index.html`,
            },
            {
                title: "OpenAPI JSON",
                description: "Contrato bruto para generadores, SDKs y tooling de CI.",
                href: `${apiReferenceBase}/swagger/doc.json`,
            },
            {
                title: "Matriz de soporte de storage",
                description: "Clasificación pública del soporte real de backends de storage, setup y notas operativas.",
                href: `${apiReferenceBase}/api/v1/providers/support?family=STORAGE`,
            },
            {
                title: "Matriz de soporte de LLM",
                description: "Clasificación pública del soporte real de LLMs, bridges y notas de runtime.",
                href: `${apiReferenceBase}/api/v1/providers/support?family=LLM`,
            },
            {
                title: "Ejemplos de requests HTTP",
                description: "Requests reutilizables para health, auth, collections, documentos, runtime y providers.",
                pathHint: "examples/http/thingindex-api.http",
            },
            {
                title: "CLI de importación de índice",
                description: "Comando Go para importar exports versionados de índice en un PostgreSQL externo con schema auto-creado.",
                pathHint: "api/cmd/index-import",
            },
            {
                title: "Contrato OpenAPI versionado",
                description: "Artefactos YAML/JSON versionados en el repositorio y usados por Scalar, Swagger y tooling externo.",
                pathHint: "api/docs/swagger.yaml",
            },
        ];
    }

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
            title: "Index import CLI",
            description: "Go command that imports versioned document index exports into an external PostgreSQL database with auto-created schema.",
            pathHint: "api/cmd/index-import",
        },
        {
            title: "Versioned OpenAPI contract",
            description: "Repository-tracked YAML/JSON artifacts used by Scalar, Swagger, and external tooling.",
            pathHint: "api/docs/swagger.yaml",
        },
    ];
}

export function getDocumentationTracks(apiBaseUrl?: string, locale: AppLocale = "en"): OpenSourceOnboardingTrack[] {
    const apiReferenceBase = resolveApiReferenceBase(apiBaseUrl);
    const hostedAPIBase = resolveHostedAPIBase(apiBaseUrl);

    if (locale === "pt-BR") {
        return [
            {
                kind: "self-hosted",
                badge: "GitHub / auto-hospedado",
                title: "Runtime local",
                description: "Clone o repositório quando precisar da stack completa localmente: interface web, API, workers, scripts de diagnóstico e documentação versionada.",
                steps: [
                    {
                        title: "Preparar ambiente e dependências",
                        command: "./scripts/bootstrap.sh",
                        detail: "Cria `.env` quando necessário e instala dependências de backend e interface web.",
                    },
                    {
                        title: "Subir o stack local",
                        command: "./scripts/stack.sh up",
                        detail: "Sobe Docker Compose com interface web, API, PostgreSQL, MinIO e RabbitMQ.",
                    },
                    {
                        title: "Rodar um diagnóstico rápido",
                        command: "./scripts/doctor.sh",
                        detail: "Verifica comandos obrigatórios, arquivos locais, disponibilidade da interface web e endpoints de docs da API.",
                    },
                ],
                notes: [
                    "UI: http://localhost:3000",
                    "API base: http://localhost:8000/api/v1",
                    "Docs principais: http://localhost:8000/docs",
                ],
            },
            {
                kind: "hosted",
                badge: "API publicada",
                title: "Consumo de API hospedada",
                description: "Use a implantação atual diretamente quando a API já estiver online e você não precisar do monorepo rodando localmente.",
                steps: [
                    {
                        title: "Comece pela documentação da implantação",
                        command: `${apiReferenceBase}/docs`,
                        detail: "Abra o Scalar na implantação atual primeiro; Swagger UI e JSON bruto ficam na mesma origem.",
                    },
                    {
                        title: "Use a base pública da API",
                        command: hostedAPIBase,
                        detail: "Aponte seu cliente, geração de SDK, Postman ou integração para essa base URL.",
                    },
                    {
                        title: "Reaproveite os exemplos versionados",
                        command: "examples/http/thingindex-api.http",
                        detail: "Se você clonou o repositório, basta alterar `@apiHost` no topo do arquivo para a origem publicada.",
                    },
                ],
                notes: [
                    `Alternativa via Swagger UI: ${apiReferenceBase}/swagger/index.html`,
                    `OpenAPI JSON: ${apiReferenceBase}/swagger/doc.json`,
                    "Exporte `/documents/:id/index/export` pela UI ou API e importe com `go run ./api/cmd/index-import --database-url <dsn> --path ./exports`.",
                    `Diagnóstico opcional do repositório: ./scripts/doctor.sh --api-base ${apiReferenceBase}`,
                ],
            },
        ];
    }

    if (locale === "es") {
        return [
            {
                kind: "self-hosted",
                badge: "GitHub / self-hosted",
                title: "Runtime local",
                description: "Clona el repositorio cuando necesites el stack completo localmente: frontend, API, workers, scripts de troubleshooting y documentación versionada.",
                steps: [
                    {
                        title: "Preparar env y dependencias",
                        command: "./scripts/bootstrap.sh",
                        detail: "Crea `.env` cuando haga falta e instala dependencias de backend y frontend.",
                    },
                    {
                        title: "Levantar el stack local",
                        command: "./scripts/stack.sh up",
                        detail: "Levanta Docker Compose con frontend, API, PostgreSQL, MinIO y RabbitMQ.",
                    },
                    {
                        title: "Ejecutar un diagnóstico rápido",
                        command: "./scripts/doctor.sh",
                        detail: "Verifica comandos obligatorios, archivos locales, reachability del frontend y endpoints de docs de la API.",
                    },
                ],
                notes: [
                    "UI: http://localhost:3000",
                    "API base: http://localhost:8000/api/v1",
                    "Docs principales: http://localhost:8000/docs",
                ],
            },
            {
                kind: "hosted",
                badge: "Deployment actual",
                title: "Consumo de API publicada",
                description: "Usa el deployment actual directamente cuando la API ya esté online y no necesites el monorepo corriendo localmente.",
                steps: [
                    {
                        title: "Empieza por la documentación del deployment",
                        command: `${apiReferenceBase}/docs`,
                        detail: "Abre Scalar en el deployment actual primero; Swagger UI y el JSON bruto quedan en el mismo origen.",
                    },
                    {
                        title: "Usa la base pública de la API",
                        command: hostedAPIBase,
                        detail: "Apunta tu client, generación de SDK, Postman o integración a esa base URL.",
                    },
                    {
                        title: "Reutiliza los ejemplos versionados",
                        command: "examples/http/thingindex-api.http",
                        detail: "Si clonaste el repo, solo cambia `@apiHost` al inicio del archivo por el origen publicado.",
                    },
                ],
                notes: [
                    `Swagger UI fallback: ${apiReferenceBase}/swagger/index.html`,
                    `OpenAPI JSON: ${apiReferenceBase}/swagger/doc.json`,
                    "Exporta `/documents/:id/index/export` desde la UI o la API e importa con `go run ./api/cmd/index-import --database-url <dsn> --path ./exports`.",
                    `Diagnóstico opcional del repo: ./scripts/doctor.sh --api-base ${apiReferenceBase}`,
                ],
            },
        ];
    }

    return [
        {
            kind: "self-hosted",
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
            kind: "hosted",
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
                "Export `/documents/:id/index/export` from the UI or API, then import with `go run ./api/cmd/index-import --database-url <dsn> --path ./exports`.",
                `Optional repo-side diagnosis: ./scripts/doctor.sh --api-base ${apiReferenceBase}`,
            ],
        },
    ];
}

export function getOpenSourceTroubleshootingItems(locale: AppLocale = "en"): OpenSourceTroubleshootingItem[] {
    if (locale === "pt-BR") {
        return [
            {
                title: "As docs da API não carregaram",
                detail: "Abra `/docs` primeiro e depois use `/swagger/index.html` ou `/swagger/doc.json` na mesma origem quando o Scalar estiver indisponível.",
            },
            {
                title: "Requests publicadas ainda batem em localhost",
                detail: "Troque `@apiHost` em `examples/http/thingindex-api.http` ou aponte a configuração do seu client para a origem publicada antes de testar de novo.",
            },
            {
                title: "O frontend local não fala com a API",
                detail: "Verifique `NEXT_PUBLIC_API_URL` e `API_INTERNAL_URL` no `.env` e rode `./scripts/doctor.sh` novamente.",
            },
            {
                title: "Execução manual no host não alcança storage",
                detail: "Use `MINIO_ENDPOINT=localhost:9000` no host e `MINIO_ENDPOINT=minio:9000` dentro do Docker Compose.",
            },
        ];
    }

    if (locale === "es") {
        return [
            {
                title: "La documentación de la API no cargó",
                detail: "Abre `/docs` primero y luego usa `/swagger/index.html` o `/swagger/doc.json` en el mismo origen cuando Scalar no esté disponible.",
            },
            {
                title: "Los requests publicados siguen apuntando a localhost",
                detail: "Cambia `@apiHost` en `examples/http/thingindex-api.http` o apunta la configuración de tu client al origen publicado antes de volver a probar.",
            },
            {
                title: "El frontend local no puede hablar con la API",
                detail: "Verifica `NEXT_PUBLIC_API_URL` y `API_INTERNAL_URL` en `.env` y vuelve a ejecutar `./scripts/doctor.sh`.",
            },
            {
                title: "La ejecución manual en el host no llega al storage",
                detail: "Usa `MINIO_ENDPOINT=localhost:9000` en el host y `MINIO_ENDPOINT=minio:9000` dentro de Docker Compose.",
            },
        ];
    }

    return [
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
}
