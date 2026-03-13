import type { AppLocale } from "@/lib/i18n/config";

export type TranslationValue = string | TranslationTree;
export type TranslationTree = {
    [key: string]: TranslationValue;
};

type LocaleMessages = Record<AppLocale, TranslationTree>;

export const messages: LocaleMessages = {
    "pt-BR": {
        common: {
            appName: "ThingIndex",
            loading: "Carregando...",
            refresh: "Atualizar status",
            oneActive: "Um ativo por vez",
            saved: "Salvo",
            active: "Ativo",
            inactive: "Inativo",
            enabled: "Habilitado",
            disabled: "Desabilitado",
            optional: "Opcional",
            editConfig: "Editar esta configuração",
            saveAndEnable: "Salvar e habilitar {name}",
            language: "Idioma",
            skipToContent: "Pular para o conteúdo principal",
            beta: "beta",
        },
        seo: {
            siteTitle: "ThingIndex",
            siteDescription:
                "Plataforma de inteligência documental com RAG, indexação semântica e orquestração de documentos com IA.",
            pages: {
                home: {
                    title: "ThingIndex | Inteligência documental com IA",
                    description:
                        "Transforme documentos em conhecimento estruturado com pipeline RAG, indexação vetorial e automação de contexto.",
                },
                login: {
                    title: "Entrar | ThingIndex",
                    description: "Acesse sua workspace de documentos, vetorização e IA no ThingIndex.",
                },
                register: {
                    title: "Criar conta | ThingIndex",
                    description: "Crie sua conta e configure pipelines de documentos com storage e LLM dedicados.",
                },
                dashboard: {
                    title: "Dashboard | ThingIndex",
                    description: "Gerencie uploads, pipelines e consultas de documentos na sua workspace do ThingIndex.",
                },
                documents: {
                    title: "Documentos | ThingIndex",
                    description: "Explore previews, outlines e vetorização dos seus documentos processados.",
                },
                documentation: {
                    title: "Documentação | ThingIndex",
                    description: "Conheça a arquitetura, o pipeline e os fundamentos técnicos do ThingIndex.",
                },
                settings: {
                    title: "Configurações | ThingIndex",
                    description: "Gerencie storages, LLMs e credenciais ativas da sua conta.",
                },
            },
        },
        shell: {
            workspace: "Workspace",
            nav: {
                dashboard: "Dashboard",
                documents: "Documentos",
                documentation: "Documentação",
                settings: "Configurações",
            },
            user: {
                role: "Plano Explorer",
                logout: "Sair",
            },
        },
        landing: {
            nav: {
                docs: "Documentação",
                login: "Entrar",
                start: "Começar grátis",
            },
            hero: {
                badge: "Plataforma de inteligência documental",
                titleStart: "Transforme documentos em",
                titleHighlight: "conhecimento estruturado",
                description:
                    "Pipeline RAG adaptativo com cadeia de 3 prompts. Indexe, estruture e consulte qualquer documento com precisão superior.",
                primaryCta: "Começar agora",
                secondaryCta: "Ver documentação",
                terminalTitle: "pipeline thingindex",
            },
            metrics: {
                precision: "mais preciso que RAG tradicional",
                latency: "latência média de query",
                throughput: "páginas processadas/hora",
                uptime: "uptime da API gerenciada",
            },
            features: {
                title: "Por que ThingIndex?",
                description: "Não é só mais um wrapper de LLM. É uma pipeline completa de document intelligence.",
                items: {
                    adaptive: {
                        title: "Cadeia adaptativa de 3 prompts",
                        desc: "Cada chunk passa por resumo, estruturação e embedding. Não é um RAG genérico.",
                    },
                    async: {
                        title: "Pipeline assíncrono",
                        desc: "RabbitMQ e workers paralelos processam milhares de páginas sem travar.",
                    },
                    vector: {
                        title: "Busca vetorial",
                        desc: "Qdrant com embeddings de alta dimensão para retrieval preciso e rápido.",
                    },
                    taxonomy: {
                        title: "Categorias e collections",
                        desc: "Organize documentos com metadados ricos para filtragem inteligente.",
                    },
                    security: {
                        title: "Segurança primeiro",
                        desc: "ClamAV escaneia cada upload. Seus dados permanecem na sua infraestrutura.",
                    },
                    openSource: {
                        title: "Open source",
                        desc: "Licença MIT. Rode na sua VM com suas chaves ou use nossa API gerenciada.",
                    },
                },
            },
            howItWorks: {
                title: "Como funciona",
                steps: {
                    upload: {
                        title: "Upload e scan",
                        desc: "Envie PDFs, defina categoria e collection. O ClamAV valida a segurança automaticamente.",
                    },
                    pipeline: {
                        title: "Pipeline de 3 prompts",
                        desc: "Cada chunk passa por resumo contextual, estruturação semântica e geração de embedding vetorial.",
                    },
                    query: {
                        title: "Query e resposta",
                        desc: "Pergunte em linguagem natural. O sistema busca vetorialmente e gera respostas com fontes.",
                    },
                },
            },
            cta: {
                title: "Pronto para começar?",
                description: "Open source e gratuito para self-hosting. Ou use nossa API gerenciada com plano free.",
                primary: "Criar conta grátis",
            },
            footer: {
                docs: "docs",
            },
        },
        auth: {
            login: {
                strapline: "orquestração documental e rag",
                password: "Senha",
                submit: "Entrar",
                noAccount: "Não tem conta?",
                createAccount: "Criar conta",
                fallbackError: "Verifique suas credenciais e tente novamente.",
                showPassword: "Mostrar senha",
                hidePassword: "Esconder senha",
                pageDescription: "Faça login no ThingIndex para acessar a orquestração de documentos e RAG.",
            },
            register: {
                strapline: "configure seu pipeline em minutos",
                name: "Nome",
                password: "Senha",
                submit: "Criar conta",
                hasAccount: "Já tem conta?",
                login: "Fazer login",
                fallbackError: "Verifique os dados informados e tente novamente.",
                showPassword: "Mostrar senha",
                hidePassword: "Esconder senha",
                pageDescription: "Crie sua conta ThingIndex para configurar pipelines de documentos e IA.",
            },
            shared: {
                email: "Email",
                or: "ou",
                city: "sao paulo",
            },
        },
        dashboard: {
            eyebrow: "Workspace relacional",
            title: "Dashboard LegisContext",
            description:
                "Centralize sua legislação e documentos jurídicos. Preserve contexto por categoria, collection e estruturas normalizadas em PostgreSQL.",
            account: {
                apiStatus: "Status da API",
                connected: "Conectado",
                monthlyUsage: "Uso mensal",
                session: "Sessão",
                expiresIn: "Expira em 4h",
            },
            tabs: {
                pipeline: "Pipeline",
                assistant: "Assistente IA",
            },
            upload: {
                category: "Categoria",
                categoryHelp: "Opcional. Em branco, o arquivo sobe sem categoria.",
                collection: "Collection",
                collectionHelp: "Obrigatória. A collection centraliza o contexto relacional entre arquivos.",
                setCollectionFirst: "Informe uma collection antes de adicionar arquivos.",
                allNeedCollection: "Todos os arquivos precisam de uma collection antes do envio.",
                sending: "Enviando arquivos...",
                sendDocuments: "Enviar {count} documento",
                sendDocumentsPlural: "Enviar {count} documentos",
                defineCollectionForAll: "Defina uma collection para todos os arquivos antes de enviar.",
                dropTitle: "Arraste seus documentos jurídicos",
                dropSubtitle: "PDF, HTML ou Excel (máx. 20MB)",
                uncategorized: "Sem categoria",
                collectionRequired: "Collection obrigatória",
                ready: "Pronto",
                error: "Erro",
                selectOrCreateCategory: "Selecione ou crie uma categoria",
                creatingCategory: "Criando categoria...",
                loadingCategories: "Carregando categorias...",
                createCategory: "Criar categoria \"{name}\"",
                noCategories: "Nenhuma categoria encontrada.",
                categoryCreated: "Categoria criada com sucesso.",
                createCategoryFailed: "Não foi possível criar a categoria.",
                categoryBackendHint: "Ao criar, o frontend envia a requisição imediatamente para o backend.",
                selectOrTypeCollection: "Selecione ou digite uma collection",
                loadingCollections: "Carregando collections...",
                noCollections: "Nenhuma collection encontrada. Pressione Enter para usar uma nova.",
                collectionHint: "O campo aceita uma collection nova e o dropdown exibe os agrupamentos relacionais já catalogados.",
            },
            chat: {
                emptyTitle: "Inicie uma consulta contextual",
                emptyDescription: "Nossa IA irá decompor sua pergunta e reconstruir o contexto da collection a partir das tabelas relacionais.",
                answer: "Encontrei as informações solicitadas. Na Petição Inicial, a fundamentação se baseia nos atrasos sistemáticos. Na Contestação, a defesa foca na força maior. Recomendo revisar a cláusula 7.2 do contrato.",
                placeholder: "Pergunte sobre os processos indexados...",
                hallucination: "IA pode alucinar. Verifique as fontes.",
                newLineHint: "Shift + Enter para nova linha",
                queryDecomposition: "Decomposição de query",
                verifiedSource: "Fonte verificada",
                page: "Pág.",
            },
            cost: {
                title: "Estimativa de custos",
                llmProvider: "Provedor LLM",
                contextStore: "Context store",
                indexedFiles: "Arquivos indexados",
                estimatedTokens: "Tokens estimados",
                estimatedCost: "Custo estimado",
                hint: "* Valores baseados em taxas padrão do Google Cloud Vertex AI.",
            },
            guide: {
                title: "Guia rápido",
                categoriesTitle: "Categorias",
                categoriesDesc: "Organize por tipo de documento, como leis e processos.",
                collectionsTitle: "Collections",
                collectionsDesc: "Agrupamentos lógicos que unificam contexto entre vários arquivos.",
                contextTitle: "Consulta de contexto",
                contextDesc: "Consulte uma collection e agregue tabelas, parágrafos e metadados no mesmo fluxo.",
                accuracy: "Precisão estimada",
            },
        },
        settings: {
            heroEyebrow: "Controle de providers",
            heroTitle: "Configurações de storage e LLM",
            heroDescription:
                "Adicione os valores do seu ambiente e escolha qual backend de storage e qual LLM ficam ativos. A plataforma permite um storage e uma LLM ativos ao mesmo tempo.",
            family: {
                STORAGE: {
                    title: "Storage",
                    description: "Exatamente um backend de storage pode ficar habilitado para uploads, previews e indexação.",
                },
                LLM: {
                    title: "LLM",
                    description: "Exatamente uma LLM pode ficar habilitada para enriquecimento e raciocínio.",
                },
            },
            steps: {
                select: "1. Selecionar provider",
                fill: "2. Preencher credenciais",
                saved: "3. Opções salvas",
            },
            selectTitleStorage: "Escolha qual storage deseja configurar",
            selectTitleLLM: "Escolha qual IA deseja configurar",
            selectDescription: "Selecione uma opção abaixo. Depois preencha apenas as credenciais desse provider no painel à direita.",
            savedTitleStorage: "Providers de storage já configurados",
            savedTitleLLM: "Providers de IA já configurados",
            savedDescription: "Você pode manter várias opções salvas, mas apenas uma fica ativa nessa família por vez.",
            emptyStorage: "Nenhum provider de storage salvo ainda. Selecione um acima e salve as credenciais primeiro.",
            emptyLLM: "Nenhum provider de IA salvo ainda. Selecione um acima e salve as credenciais primeiro.",
            behaviorTitle: "Comportamento",
            behaviorDescription:
                "Salvar este provider armazena as credenciais e o habilita automaticamente. Qualquer outro provider da mesma família é desabilitado.",
            loading: "Carregando configurações de providers...",
        },
    },
    en: {
        common: {
            appName: "ThingIndex",
            loading: "Loading...",
            refresh: "Refresh status",
            oneActive: "One active at a time",
            saved: "Saved",
            active: "Active",
            inactive: "Inactive",
            enabled: "Enabled",
            disabled: "Disabled",
            optional: "Optional",
            editConfig: "Edit this configuration",
            saveAndEnable: "Save and enable {name}",
            language: "Language",
            skipToContent: "Skip to main content",
            beta: "beta",
        },
        seo: {
            siteTitle: "ThingIndex",
            siteDescription:
                "Document intelligence platform with RAG, semantic indexing, and AI-powered document orchestration.",
            pages: {
                home: {
                    title: "ThingIndex | AI document intelligence",
                    description:
                        "Turn documents into structured knowledge with RAG pipelines, vector search, and context automation.",
                },
                login: {
                    title: "Login | ThingIndex",
                    description: "Sign in to your ThingIndex workspace for documents, vectorization, and AI workflows.",
                },
                register: {
                    title: "Create account | ThingIndex",
                    description: "Create your account and configure document pipelines with dedicated storage and LLM providers.",
                },
                dashboard: {
                    title: "Dashboard | ThingIndex",
                    description: "Manage uploads, pipelines, and document queries in your ThingIndex workspace.",
                },
                documents: {
                    title: "Documents | ThingIndex",
                    description: "Explore previews, outlines, and vectorization results for your processed documents.",
                },
                documentation: {
                    title: "Documentation | ThingIndex",
                    description: "Explore the architecture, pipeline, and technical foundations behind ThingIndex.",
                },
                settings: {
                    title: "Settings | ThingIndex",
                    description: "Manage the active storage, LLM, and provider credentials for your account.",
                },
            },
        },
        shell: {
            workspace: "Workspace",
            nav: {
                dashboard: "Dashboard",
                documents: "Documents",
                documentation: "Documentation",
                settings: "Settings",
            },
            user: {
                role: "Explorer Plan",
                logout: "Sign out",
            },
        },
        landing: {
            nav: {
                docs: "Documentation",
                login: "Login",
                start: "Start free",
            },
            hero: {
                badge: "Document intelligence platform",
                titleStart: "Turn documents into",
                titleHighlight: "structured knowledge",
                description:
                    "Adaptive RAG pipeline with a 3-prompt chain. Index, structure, and query any document with better precision.",
                primaryCta: "Start now",
                secondaryCta: "View documentation",
                terminalTitle: "thingindex pipeline",
            },
            metrics: {
                precision: "more accurate than traditional RAG",
                latency: "average query latency",
                throughput: "pages processed per hour",
                uptime: "managed API uptime",
            },
            features: {
                title: "Why ThingIndex?",
                description: "This is more than another LLM wrapper. It is a full document intelligence pipeline.",
                items: {
                    adaptive: {
                        title: "Adaptive 3-prompt chain",
                        desc: "Each chunk goes through summarization, structuring, and embeddings. This is not generic RAG.",
                    },
                    async: {
                        title: "Asynchronous pipeline",
                        desc: "RabbitMQ and parallel workers process thousands of pages without blocking.",
                    },
                    vector: {
                        title: "Vector retrieval",
                        desc: "Qdrant with high-dimensional embeddings for precise and fast retrieval.",
                    },
                    taxonomy: {
                        title: "Categories and collections",
                        desc: "Organize documents with rich metadata for smarter filtering.",
                    },
                    security: {
                        title: "Security first",
                        desc: "ClamAV scans every upload. Your data stays inside your own infrastructure.",
                    },
                    openSource: {
                        title: "Open source",
                        desc: "MIT licensed. Run it on your VM with your keys or use our managed API.",
                    },
                },
            },
            howItWorks: {
                title: "How it works",
                steps: {
                    upload: {
                        title: "Upload and scan",
                        desc: "Upload PDFs, define a category and collection. ClamAV validates security automatically.",
                    },
                    pipeline: {
                        title: "3-prompt pipeline",
                        desc: "Each chunk goes through contextual summarization, semantic structuring, and vector embedding generation.",
                    },
                    query: {
                        title: "Query and answer",
                        desc: "Ask in natural language. The system retrieves vector context and answers with sources.",
                    },
                },
            },
            cta: {
                title: "Ready to get started?",
                description: "Open source and free for self-hosting. Or use our managed API with a free plan.",
                primary: "Create free account",
            },
            footer: {
                docs: "docs",
            },
        },
        auth: {
            login: {
                strapline: "document orchestration and rag",
                password: "Password",
                submit: "Sign in",
                noAccount: "Don't have an account?",
                createAccount: "Create account",
                fallbackError: "Check your credentials and try again.",
                showPassword: "Show password",
                hidePassword: "Hide password",
                pageDescription: "Sign in to ThingIndex to access document orchestration and RAG workflows.",
            },
            register: {
                strapline: "configure your pipeline in minutes",
                name: "Name",
                password: "Password",
                submit: "Create account",
                hasAccount: "Already have an account?",
                login: "Sign in",
                fallbackError: "Check the submitted data and try again.",
                showPassword: "Show password",
                hidePassword: "Hide password",
                pageDescription: "Create your ThingIndex account to configure document and AI pipelines.",
            },
            shared: {
                email: "Email",
                or: "or",
                city: "sao paulo",
            },
        },
        dashboard: {
            eyebrow: "Relational workspace",
            title: "LegisContext Dashboard",
            description:
                "Centralize your legislation and legal documents. Preserve context through categories, collections, and normalized PostgreSQL structures.",
            account: {
                apiStatus: "API status",
                connected: "Connected",
                monthlyUsage: "Monthly usage",
                session: "Session",
                expiresIn: "Expires in 4h",
            },
            tabs: {
                pipeline: "Pipeline",
                assistant: "AI Assistant",
            },
            upload: {
                category: "Category",
                categoryHelp: "Optional. Leave blank to upload without a category.",
                collection: "Collection",
                collectionHelp: "Required. The collection centralizes relational context across files.",
                setCollectionFirst: "Set a collection before adding files.",
                allNeedCollection: "All files need a collection before upload.",
                sending: "Uploading files...",
                sendDocuments: "Upload {count} document",
                sendDocumentsPlural: "Upload {count} documents",
                defineCollectionForAll: "Set a collection for every file before uploading.",
                dropTitle: "Drop your legal documents",
                dropSubtitle: "PDF, HTML, or Excel (max 20MB)",
                uncategorized: "Uncategorized",
                collectionRequired: "Collection required",
                ready: "Ready",
                error: "Error",
                selectOrCreateCategory: "Select or create a category",
                creatingCategory: "Creating category...",
                loadingCategories: "Loading categories...",
                createCategory: "Create category \"{name}\"",
                noCategories: "No categories found.",
                categoryCreated: "Category created successfully.",
                createCategoryFailed: "Could not create the category.",
                categoryBackendHint: "When creating one, the frontend sends the request to the backend immediately.",
                selectOrTypeCollection: "Select or type a collection",
                loadingCollections: "Loading collections...",
                noCollections: "No collections found. Press Enter to use a new one.",
                collectionHint: "This field accepts a new collection and shows existing relational groupings in the dropdown.",
            },
            chat: {
                emptyTitle: "Start a contextual query",
                emptyDescription: "Our AI will decompose your question and rebuild collection context from relational tables.",
                answer: "I found the requested information. In the initial filing, the argument is based on repeated delays. In the response, the defense focuses on force majeure. I recommend reviewing clause 7.2 of the contract.",
                placeholder: "Ask about the indexed cases...",
                hallucination: "AI may hallucinate. Verify the sources.",
                newLineHint: "Shift + Enter for a new line",
                queryDecomposition: "Query decomposition",
                verifiedSource: "Verified source",
                page: "Page",
            },
            cost: {
                title: "Cost estimate",
                llmProvider: "LLM provider",
                contextStore: "Context store",
                indexedFiles: "Indexed files",
                estimatedTokens: "Estimated tokens",
                estimatedCost: "Estimated cost",
                hint: "* Values are based on standard Google Cloud Vertex AI pricing.",
            },
            guide: {
                title: "Quick guide",
                categoriesTitle: "Categories",
                categoriesDesc: "Organize by document type such as laws and cases.",
                collectionsTitle: "Collections",
                collectionsDesc: "Logical groupings that unify context across multiple files.",
                contextTitle: "Context query",
                contextDesc: "Query a collection and aggregate tables, paragraphs, and metadata in one flow.",
                accuracy: "Estimated accuracy",
            },
        },
        settings: {
            heroEyebrow: "Provider control",
            heroTitle: "Storage and LLM settings",
            heroDescription:
                "Add your environment values and choose which storage backend and LLM remain active. The platform allows one active storage and one active LLM at a time.",
            family: {
                STORAGE: {
                    title: "Storage provider",
                    description: "Exactly one storage backend can stay enabled for uploads, previews, and indexing.",
                },
                LLM: {
                    title: "LLM provider",
                    description: "Exactly one LLM can stay enabled for enrichment and reasoning workloads.",
                },
            },
            steps: {
                select: "1. Select provider",
                fill: "2. Fill credentials",
                saved: "3. Saved options",
            },
            selectTitleStorage: "Choose which storage you want to configure",
            selectTitleLLM: "Choose which AI you want to configure",
            selectDescription: "Select one option below. Then fill only the credentials for that provider on the right.",
            savedTitleStorage: "Providers already configured for storage",
            savedTitleLLM: "Providers already configured for AI",
            savedDescription: "You can keep several saved options, but only one stays active in this family at a time.",
            emptyStorage: "No saved storage providers yet. Select one above and save its credentials first.",
            emptyLLM: "No saved AI providers yet. Select one above and save its credentials first.",
            behaviorTitle: "Behavior",
            behaviorDescription:
                "Saving this provider stores its credentials and enables it automatically. Any other provider in the same family is disabled.",
            loading: "Loading provider settings...",
        },
    },
    es: {
        common: {
            appName: "ThingIndex",
            loading: "Cargando...",
            refresh: "Actualizar estado",
            oneActive: "Uno activo a la vez",
            saved: "Guardado",
            active: "Activo",
            inactive: "Inactivo",
            enabled: "Habilitado",
            disabled: "Deshabilitado",
            optional: "Opcional",
            editConfig: "Editar esta configuración",
            saveAndEnable: "Guardar y habilitar {name}",
            language: "Idioma",
            skipToContent: "Saltar al contenido principal",
            beta: "beta",
        },
        seo: {
            siteTitle: "ThingIndex",
            siteDescription:
                "Plataforma de inteligencia documental con RAG, indexación semántica y orquestación de documentos con IA.",
            pages: {
                home: {
                    title: "ThingIndex | Inteligencia documental con IA",
                    description:
                        "Convierte documentos en conocimiento estructurado con pipelines RAG, búsqueda vectorial y automatización de contexto.",
                },
                login: {
                    title: "Iniciar sesión | ThingIndex",
                    description: "Accede a tu workspace de documentos, vectorización e IA en ThingIndex.",
                },
                register: {
                    title: "Crear cuenta | ThingIndex",
                    description: "Crea tu cuenta y configura pipelines de documentos con storage y LLM dedicados.",
                },
                dashboard: {
                    title: "Dashboard | ThingIndex",
                    description: "Gestiona cargas, pipelines y consultas de documentos en tu workspace de ThingIndex.",
                },
                documents: {
                    title: "Documentos | ThingIndex",
                    description: "Explora previews, outlines y resultados de vectorización de tus documentos procesados.",
                },
                documentation: {
                    title: "Documentación | ThingIndex",
                    description: "Conoce la arquitectura, el pipeline y los fundamentos técnicos de ThingIndex.",
                },
                settings: {
                    title: "Configuración | ThingIndex",
                    description: "Administra el storage activo, la LLM y las credenciales de providers de tu cuenta.",
                },
            },
        },
        shell: {
            workspace: "Workspace",
            nav: {
                dashboard: "Dashboard",
                documents: "Documentos",
                documentation: "Documentación",
                settings: "Configuración",
            },
            user: {
                role: "Plan Explorer",
                logout: "Cerrar sesión",
            },
        },
        landing: {
            nav: {
                docs: "Documentación",
                login: "Entrar",
                start: "Empezar gratis",
            },
            hero: {
                badge: "Plataforma de inteligencia documental",
                titleStart: "Convierte documentos en",
                titleHighlight: "conocimiento estructurado",
                description:
                    "Pipeline RAG adaptativo con una cadena de 3 prompts. Indexa, estructura y consulta cualquier documento con mejor precisión.",
                primaryCta: "Empezar ahora",
                secondaryCta: "Ver documentación",
                terminalTitle: "thingindex pipeline",
            },
            metrics: {
                precision: "más preciso que el RAG tradicional",
                latency: "latencia media de consulta",
                throughput: "páginas procesadas por hora",
                uptime: "uptime de la API gestionada",
            },
            features: {
                title: "¿Por qué ThingIndex?",
                description: "No es solo otro wrapper de LLM. Es un pipeline completo de inteligencia documental.",
                items: {
                    adaptive: {
                        title: "Cadena adaptativa de 3 prompts",
                        desc: "Cada chunk pasa por resumen, estructuración y embeddings. No es un RAG genérico.",
                    },
                    async: {
                        title: "Pipeline asíncrono",
                        desc: "RabbitMQ y workers paralelos procesan miles de páginas sin bloquear.",
                    },
                    vector: {
                        title: "Búsqueda vectorial",
                        desc: "Qdrant con embeddings de alta dimensión para recuperación precisa y rápida.",
                    },
                    taxonomy: {
                        title: "Categorías y collections",
                        desc: "Organiza documentos con metadatos ricos para un filtrado más inteligente.",
                    },
                    security: {
                        title: "Seguridad primero",
                        desc: "ClamAV escanea cada carga. Tus datos permanecen dentro de tu infraestructura.",
                    },
                    openSource: {
                        title: "Open source",
                        desc: "Licencia MIT. Ejecútalo en tu VM con tus claves o usa nuestra API gestionada.",
                    },
                },
            },
            howItWorks: {
                title: "Cómo funciona",
                steps: {
                    upload: {
                        title: "Carga y escaneo",
                        desc: "Sube PDFs, define categoría y collection. ClamAV valida la seguridad automáticamente.",
                    },
                    pipeline: {
                        title: "Pipeline de 3 prompts",
                        desc: "Cada chunk pasa por resumen contextual, estructuración semántica y generación de embeddings vectoriales.",
                    },
                    query: {
                        title: "Consulta y respuesta",
                        desc: "Pregunta en lenguaje natural. El sistema recupera contexto vectorial y responde con fuentes.",
                    },
                },
            },
            cta: {
                title: "¿Listo para empezar?",
                description: "Open source y gratis para self-hosting. O usa nuestra API gestionada con plan free.",
                primary: "Crear cuenta gratis",
            },
            footer: {
                docs: "docs",
            },
        },
        auth: {
            login: {
                strapline: "orquestación documental y rag",
                password: "Contraseña",
                submit: "Entrar",
                noAccount: "¿No tienes cuenta?",
                createAccount: "Crear cuenta",
                fallbackError: "Revisa tus credenciales e inténtalo de nuevo.",
                showPassword: "Mostrar contraseña",
                hidePassword: "Ocultar contraseña",
                pageDescription: "Inicia sesión en ThingIndex para acceder a la orquestación documental y los flujos RAG.",
            },
            register: {
                strapline: "configura tu pipeline en minutos",
                name: "Nombre",
                password: "Contraseña",
                submit: "Crear cuenta",
                hasAccount: "¿Ya tienes cuenta?",
                login: "Iniciar sesión",
                fallbackError: "Revisa los datos enviados e inténtalo de nuevo.",
                showPassword: "Mostrar contraseña",
                hidePassword: "Ocultar contraseña",
                pageDescription: "Crea tu cuenta ThingIndex para configurar pipelines documentales y de IA.",
            },
            shared: {
                email: "Email",
                or: "o",
                city: "sao paulo",
            },
        },
        dashboard: {
            eyebrow: "Workspace relacional",
            title: "Dashboard LegisContext",
            description:
                "Centraliza tu legislación y documentos jurídicos. Conserva el contexto mediante categorías, collections y estructuras normalizadas en PostgreSQL.",
            account: {
                apiStatus: "Estado de la API",
                connected: "Conectado",
                monthlyUsage: "Uso mensual",
                session: "Sesión",
                expiresIn: "Expira en 4h",
            },
            tabs: {
                pipeline: "Pipeline",
                assistant: "Asistente IA",
            },
            upload: {
                category: "Categoría",
                categoryHelp: "Opcional. Déjalo vacío para subir sin categoría.",
                collection: "Collection",
                collectionHelp: "Obligatoria. La collection centraliza el contexto relacional entre archivos.",
                setCollectionFirst: "Define una collection antes de agregar archivos.",
                allNeedCollection: "Todos los archivos necesitan una collection antes del envío.",
                sending: "Subiendo archivos...",
                sendDocuments: "Enviar {count} documento",
                sendDocumentsPlural: "Enviar {count} documentos",
                defineCollectionForAll: "Define una collection para todos los archivos antes de enviar.",
                dropTitle: "Arrastra tus documentos legales",
                dropSubtitle: "PDF, HTML o Excel (máx. 20MB)",
                uncategorized: "Sin categoría",
                collectionRequired: "Collection obligatoria",
                ready: "Listo",
                error: "Error",
                selectOrCreateCategory: "Selecciona o crea una categoría",
                creatingCategory: "Creando categoría...",
                loadingCategories: "Cargando categorías...",
                createCategory: "Crear categoría \"{name}\"",
                noCategories: "No se encontraron categorías.",
                categoryCreated: "Categoría creada con éxito.",
                createCategoryFailed: "No fue posible crear la categoría.",
                categoryBackendHint: "Al crearla, el frontend envía la solicitud al backend de inmediato.",
                selectOrTypeCollection: "Selecciona o escribe una collection",
                loadingCollections: "Cargando collections...",
                noCollections: "No se encontraron collections. Presiona Enter para usar una nueva.",
                collectionHint: "El campo acepta una collection nueva y el dropdown muestra agrupaciones relacionales ya catalogadas.",
            },
            chat: {
                emptyTitle: "Inicia una consulta contextual",
                emptyDescription: "Nuestra IA descompondrá tu pregunta y reconstruirá el contexto de la collection a partir de tablas relacionales.",
                answer: "Encontré la información solicitada. En la demanda inicial, el fundamento se basa en retrasos sistemáticos. En la contestación, la defensa se enfoca en fuerza mayor. Recomiendo revisar la cláusula 7.2 del contrato.",
                placeholder: "Pregunta sobre los procesos indexados...",
                hallucination: "La IA puede alucinar. Verifica las fuentes.",
                newLineHint: "Shift + Enter para nueva línea",
                queryDecomposition: "Descomposición de la consulta",
                verifiedSource: "Fuente verificada",
                page: "Pág.",
            },
            cost: {
                title: "Estimación de costos",
                llmProvider: "Proveedor LLM",
                contextStore: "Context store",
                indexedFiles: "Archivos indexados",
                estimatedTokens: "Tokens estimados",
                estimatedCost: "Costo estimado",
                hint: "* Valores basados en tarifas estándar de Google Cloud Vertex AI.",
            },
            guide: {
                title: "Guía rápida",
                categoriesTitle: "Categorías",
                categoriesDesc: "Organiza por tipo de documento, como leyes y procesos.",
                collectionsTitle: "Collections",
                collectionsDesc: "Agrupaciones lógicas que unifican contexto entre varios archivos.",
                contextTitle: "Consulta de contexto",
                contextDesc: "Consulta una collection y agrega tablas, párrafos y metadatos en el mismo flujo.",
                accuracy: "Precisión estimada",
            },
        },
        settings: {
            heroEyebrow: "Control de providers",
            heroTitle: "Configuración de storage y LLM",
            heroDescription:
                "Agrega los valores de tu entorno y elige qué backend de storage y qué LLM permanecen activos. La plataforma permite un storage y una LLM activos al mismo tiempo.",
            family: {
                STORAGE: {
                    title: "Provider de storage",
                    description: "Exactamente un backend de storage puede permanecer habilitado para cargas, previews e indexación.",
                },
                LLM: {
                    title: "Provider LLM",
                    description: "Exactamente una LLM puede permanecer habilitada para enriquecimiento y razonamiento.",
                },
            },
            steps: {
                select: "1. Seleccionar provider",
                fill: "2. Completar credenciales",
                saved: "3. Opciones guardadas",
            },
            selectTitleStorage: "Elige qué storage quieres configurar",
            selectTitleLLM: "Elige qué IA quieres configurar",
            selectDescription: "Selecciona una opción abajo. Luego completa solo las credenciales de ese provider en el panel derecho.",
            savedTitleStorage: "Providers de storage ya configurados",
            savedTitleLLM: "Providers de IA ya configurados",
            savedDescription: "Puedes mantener varias opciones guardadas, pero solo una permanece activa en esta familia a la vez.",
            emptyStorage: "Todavía no hay providers de storage guardados. Selecciona uno arriba y guarda primero sus credenciales.",
            emptyLLM: "Todavía no hay providers de IA guardados. Selecciona uno arriba y guarda primero sus credenciales.",
            behaviorTitle: "Comportamiento",
            behaviorDescription:
                "Guardar este provider almacena sus credenciales y lo habilita automáticamente. Cualquier otro provider de la misma familia se deshabilita.",
            loading: "Cargando configuración de providers...",
        },
    },
};

export function getMessages(locale: AppLocale): TranslationTree {
    return messages[locale];
}
