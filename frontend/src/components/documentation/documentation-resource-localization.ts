import type { IndexingBenchmarkReport } from "@/lib/api/indexing-benchmark";
import type { ProviderSupport } from "@/lib/api/providers";
import type { RuntimeProfile } from "@/lib/api/runtime-profile";
import type { AppLocale } from "@/lib/i18n/config";

const ptRuntimeProfileAudienceMap: Record<string, { name: string; description: string }> = {
    "Operations and compliance teams": {
        name: "Equipes de operações e compliance",
        description: "Precisam de workspaces pesquisáveis, proveniência explícita e tratamento documental repetível entre coleções.",
    },
    "Legal and policy workflows": {
        name: "Fluxos jurídicos e de políticas",
        description: "Se beneficiam de contexto por coleção, extração estruturada e revisão grounded de acompanhamento.",
    },
    "Academic and research environments": {
        name: "Ambientes acadêmicos e de pesquisa",
        description: "Precisam de um runtime inspecionável, reproduzível e fácil de defender tecnicamente em um TCC ou artigo.",
    },
};

const ptRuntimeProfileCurrentScopeMap: Record<string, string> = {
    "JWT-authenticated document workspace with categories, collections, and uploads":
        "Workspace documental com autenticação JWT, categorias, coleções e uploads",
    "Structured indexing with paragraphs, tables, outline nodes, metadata pairs, and monitoring telemetry":
        "Indexação estruturada com parágrafos, tabelas, nós de estrutura, pares de metadados e telemetria de monitoramento",
    "Relational collection-context recovery in PostgreSQL":
        "Recuperação relacional de contexto por coleção em PostgreSQL",
    "Grounded chat by collection with provider runtime separation for chat and indexing":
        "Chat grounded por coleção com separação de runtime entre provedores de chat e indexação",
    "Self-hosted stack with Go API, Next.js frontend, PostgreSQL, MinIO, RabbitMQ, and versioned API docs":
        "Stack auto-hospedada com API Go, frontend Next.js, PostgreSQL, MinIO, RabbitMQ e docs versionadas da API",
};

const ptRuntimeProfileDifferentiatorsMap: Record<string, string> = {
    "Relational-first context reconstruction instead of vector-first navigation":
        "Reconstrução de contexto relacional em vez de navegação centrada em vetores",
    "Explicit separation between current runtime, experimental provider paths, and research-only extensions":
        "Separação explícita entre o runtime atual, caminhos experimentais de provedores e extensões apenas de pesquisa",
    "Operational visibility with audit trails, readiness checks, monitoring, and documented support levels":
        "Visibilidade operacional com trilhas de auditoria, checks de readiness, monitoramento e níveis de suporte documentados",
};

const ptRuntimeProfileNonGoalsMap: Record<string, string> = {
    "Do not present the main runtime as a generic AI copilot detached from document provenance":
        "Não apresentar o runtime principal como um copiloto genérico de IA desconectado da proveniência documental",
    "Do not claim GraphRAG, graph databases, or native vector search as required parts of the current production path":
        "Não tratar GraphRAG, bancos de grafos ou busca vetorial nativa como partes obrigatórias do caminho atual de produção",
    "Do not collapse research experiments into the mainline architecture without isolated evaluation":
        "Não misturar experimentos de pesquisa com a arquitetura principal sem avaliação isolada",
};

const ptRuntimeProfileResearchTracksMap: Record<string, string> = {
    "Expand the controlled indexing benchmark to compare the relational baseline against vector retrieval":
        "Expandir o benchmark controlado de indexação para comparar a baseline relacional com recuperação vetorial",
    "Evaluate GraphRAG as an isolated experimental extension instead of a core dependency":
        "Avaliar GraphRAG como uma extensão experimental isolada, e não como dependência central",
    "Strengthen grounded chat with measurable citation quality, accuracy, and latency metrics":
        "Fortalecer o chat grounded com métricas mensuráveis de qualidade de citação, acurácia e latência",
};

const ptRuntimeProfileArtifactsByPath: Record<string, { label: string; summary: string }> = {
    "rfc/0001-product-focus.md": {
        label: "RFC de foco do produto",
        summary: "Define o caso de uso principal, usuários-alvo, não objetivos explícitos e implicações de backlog.",
    },
    "docs/article/sbc_template (1).doc": {
        label: "Referência do template SBC",
        summary: "Template Word original da SBC mantido como referência de formatação para o arquivo final da submissão.",
    },
    "docs/article/README.md": {
        label: "Workspace acadêmico",
        summary: "Notas do workspace e fontes de figuras usadas para preparar o artefato final da submissão SBC fora do histórico principal do runtime.",
    },
    "api/internal/usecase/testdata/indexing-benchmark/corpus.json": {
        label: "Corpus do benchmark de indexação",
        summary: "Corpus controlado e expectativas de correspondência exata usados pelo endpoint público do benchmark de indexação.",
    },
    "docs/proposta-continuacao-tcc.md": {
        label: "Proposta de continuidade acadêmica",
        summary: "Notas de continuidade da pesquisa que enquadram vetores e GraphRAG como experimentos futuros explícitos.",
    },
};

const ptIndexingBenchmarkNarrativeMap: Record<string, string> = {
    "Controlled exact-match benchmark for the current relational indexing baseline. The corpus is embedded, reproducible, and evaluated with the real deterministic indexer used by the main runtime.":
        "Benchmark controlado de correspondência exata para a baseline relacional atual de indexação. O corpus é embutido, reproduzível e avaliado com o indexador determinístico real usado pelo runtime principal.",
    "Run the real deterministic indexer over the embedded benchmark corpus.":
        "Executar o indexador determinístico real sobre o corpus de benchmark embutido.",
    "Check expected outline titles and table counts for each benchmark document.":
        "Verificar títulos esperados da estrutura e contagem de tabelas para cada documento do benchmark.",
    "Run document-tree retrieval cases and validate exact top-1 node and page matches.":
        "Executar casos de recuperação na árvore documental e validar correspondência exata de nó e página em top-1.",
    "Run collection retrieval cases and validate exact top-1 document matches.":
        "Executar casos de recuperação por coleção e validar correspondência exata do documento em top-1.",
    "Keep the monitoring heuristic visible as an operational signal, but separate it from benchmark accuracy.":
        "Manter a heurística de monitoramento visível como sinal operacional, mas separada da acurácia do benchmark.",
    "Compare the current relational baseline against flat and hybrid experimental retrieval controls over the same corpus.":
        "Comparar a baseline relacional atual com controles experimentais planos e híbridos de retrieval sobre o mesmo corpus.",
    "Document structure passes only when all expected outline titles are present and the table count matches the corpus expectation.":
        "A estrutura documental só é aprovada quando todos os títulos esperados da estrutura estão presentes e a contagem de tabelas corresponde ao esperado no corpus.",
    "Document-tree cases pass only when the expected node and page are both ranked first.":
        "Casos da árvore documental só são aprovados quando o nó esperado e a página esperada aparecem ambos em primeiro lugar.",
    "Collection cases pass only when the expected document is ranked first.":
        "Casos por coleção só são aprovados quando o documento esperado aparece em primeiro lugar.",
    "This corpus is intentionally small and controlled; it validates the baseline, not arbitrary production corpora.":
        "Este corpus é intencionalmente pequeno e controlado; ele valida a baseline, não corpora arbitrários de produção.",
    "The benchmark measures structural indexing and lexical retrieval on the current runtime, not grounded chat answer faithfulness.":
        "O benchmark mede indexação estrutural e recuperação lexical no runtime atual, não a fidelidade das respostas do chat grounded.",
    "The public comparison currently covers flat and hybrid experimental controls; native vector retrieval and GraphRAG remain separate research tracks.":
        "A comparação pública atualmente cobre controles experimentais planos e híbridos; recuperação vetorial nativa e GraphRAG continuam como trilhas de pesquisa separadas.",
    "The quality score exposed in document monitoring remains a structural heuristic for operators. It is not ground truth and should be interpreted together with this benchmark instead of replacing it.":
        "A pontuação de qualidade exposta no monitoramento documental continua sendo uma heurística estrutural para operadores. Ela não é verdade de referência e deve ser interpretada junto com este benchmark, e não no lugar dele.",
    "The relational baseline remains ahead of the strongest experimental control (Flat hybrid control), especially on exact node and page grounding.":
        "A baseline relacional continua à frente do controle experimental mais forte (controle híbrido plano), especialmente no grounding exato de nó e página.",
    "Current production path combining outline-aware retrieval, page grounding, and explicit relational provenance.":
        "Caminho atual de produção combinando retrieval orientado por outline, grounding por página e proveniência relacional explícita.",
    "Experimental control that ranks nodes and pages independently without iterative tree traversal or path reinforcement.":
        "Controle experimental que ranqueia nós e páginas de forma independente, sem travessia iterativa da árvore nem reforço de caminho.",
    "Experimental control that flattens retrieval to page text and maps page hits back to overlapping nodes only.":
        "Controle experimental que achata o retrieval para texto de página e mapeia hits de página apenas para nós sobrepostos.",
    "This repository does not ship a native vector index in the current runtime.":
        "Este repositório não entrega um índice vetorial nativo no runtime atual.",
    "The variants below are explicit experimental controls over the same benchmark corpus, not hidden production paths.":
        "As variantes abaixo são controles experimentais explícitos sobre o mesmo corpus do benchmark, e não caminhos ocultos de produção.",
    "A future vector or GraphRAG baseline should be added here as a separate experimental variant instead of replacing the relational path.":
        "Uma baseline futura de vetores ou GraphRAG deve ser adicionada aqui como variante experimental separada, em vez de substituir o caminho relacional.",
};

const ptIndexingBenchmarkFocusByDocumentID: Record<string, string> = {
    "service-agreement": "Prazos de pagamento, escopo de onboarding e compromissos de nível de serviço.",
    "incident-playbook": "Fluxo de escalonamento, ownership da bridge e retenção de evidências de resposta.",
    "retention-policy": "Agenda de retenção e fluxo de exclusão para registros operacionais.",
};

const ptIndexingBenchmarkDocumentTitleMap: Record<string, string> = {
    "Master Service Agreement": "Contrato Mestre de Serviços",
    "Incident Response Playbook": "Playbook de Resposta a Incidentes",
    "Data Retention Policy": "Política de Retenção de Dados",
};

const ptIndexingBenchmarkRetrievalVariantLabelMap: Record<string, string> = {
    "Current relational baseline": "Baseline relacional atual",
    "Flat hybrid control": "Controle híbrido plano",
    "Flat page control": "Controle plano por páginas",
};

const ptIndexingBenchmarkNodeTitleMap: Record<string, string> = {
    "MASTER SERVICE AGREEMENT": "CONTRATO MESTRE DE SERVIÇOS",
    "1 Scope": "1 Escopo",
    "2 Payment Terms": "2 Condições de pagamento",
    "3 Service Levels": "3 Níveis de serviço",
    "INCIDENT RESPONSE PLAYBOOK": "PLAYBOOK DE RESPOSTA A INCIDENTES",
    "1 Severity Model": "1 Modelo de severidade",
    "2 Escalation": "2 Escalonamento",
    "3 Evidence Retention": "3 Retenção de evidências",
    "DATA RETENTION POLICY": "POLÍTICA DE RETENÇÃO DE DADOS",
    "Retention Schedule": "Agenda de retenção",
    "Deletion Workflow": "Fluxo de exclusão",
};

const ptIndexingBenchmarkQueryByCaseID: Record<string, string> = {
    "service-payment-deadline": "prazo de pagamento recebimento da fatura",
    "incident-bridge-commander": "bridge commander severidade um",
    "retention-security-incidents": "retenção de incidentes de segurança",
    "retention-deletion-workflow": "solicitações de exclusão dupla aprovação",
    "collection-payment-pause": "pagamento em atraso aviso de onboarding",
    "collection-bridge-notification": "bridge commander notificação executiva",
    "collection-security-retention": "incidentes de segurança sete anos",
};

const ptProviderSetupValueMap: Record<string, string> = {
    "Bring your bridge URL": "Informe a URL da sua bridge",
};

const ptProviderSupportByType: Partial<
    Record<
        ProviderSupport["provider_type"],
        {
            display_name?: string;
            summary: string;
            notes: string[];
        }
    >
> = {
    MINIO: {
        display_name: "MinIO / compatível com S3",
        summary: "Pronto para o runtime principal com upload, visualização, download, indexação e cobertura de integração ao vivo contra a stack local.",
        notes: [
            "Usa o caminho compartilhado compatível com S3 apoiado pelo cliente Go do MinIO.",
            "Adequado para o runtime local padrão e para implantações auto-hospedadas que querem um caminho direto de armazenamento de objetos.",
        ],
    },
    AWS_S3: {
        summary: "Pronto via o adaptador compatível com S3 compartilhado, com padrões sensíveis à região, verificações de prontidão e o mesmo ciclo documental usado pelo MinIO.",
        notes: [
            "O bootstrap do runtime pode continuar em MinIO enquanto provedores por usuário apontam para buckets AWS S3.",
            "Endpoints customizados continuam opcionais para gateways compatíveis com S3 colocados à frente do S3.",
        ],
    },
    CLOUDFLARE_R2: {
        summary: "Suportado pelo adaptador compatível com S3 compartilhado, com defaults de endpoint sensíveis à conta e paridade no ciclo documental.",
        notes: [
            "Usa o mesmo adaptador de armazenamento de objetos do caminho MinIO/S3, e não uma SDK específica da Cloudflare.",
            "Valide comportamentos específicos do fornecedor na sua própria implantação antes de assumir paridade com todos os recursos do R2.",
        ],
    },
    DIGITALOCEAN_SPACES: {
        summary: "Suportado pelo adaptador compatível com S3 compartilhado quando você informa endpoint regional, bucket e credenciais.",
        notes: [
            "O comportamento operacional segue o mesmo caminho compatível com S3 já usado pelo pipeline documental.",
            "Trate CDN, ACL ou políticas de ciclo de vida específicas do provider como preocupações de implantação fora do contrato principal do runtime.",
        ],
    },
    WASABI: {
        summary: "Suportado pelo adaptador compatível com S3 compartilhado, com endpoints regionais explícitos e validação de bucket.",
        notes: [
            "Usa o mesmo fluxo de upload, preview e exclusão exercitado pelo caminho de armazenamento compatível com S3.",
            "Mantenha a validação de implantação focada na sua configuração concreta de região e política de bucket na Wasabi.",
        ],
    },
    BACKBLAZE_B2: {
        summary: "Suportado pelo adaptador compatível com S3 compartilhado com endpoint, região e bucket configurados explicitamente.",
        notes: [
            "Backblaze B2 trafega pelo caminho compatível com S3 comum, em vez de uma SDK dedicada do B2.",
            "Valide políticas operacionais específicas do B2 separadamente quando garantias rígidas dependerem delas.",
        ],
    },
    GCP_STORAGE: {
        summary: "Pronto para o runtime principal com caminho nativo de Google Cloud Storage, armazenamento criptografado de credenciais, teste de conexão e cobertura de integração via emulator.",
        notes: [
            "Suporta JSON de service account para buckets reais e emulator_host para desenvolvimento e integrações locais.",
            "Reduz a dependência prática da stack compatível com S3 mantendo o mesmo contrato de ciclo documental.",
        ],
    },
    OPENAI: {
        summary: "Pronto para o runtime principal com configuração padrão direta da OpenAI, teste de conexão, descoberta de modelos e chat/indexação grounded.",
        notes: [
            "Usa o caminho direto da API OpenAI com substituições opcionais de URL base personalizada.",
            "O modelo pode permanecer implícito porque o runtime faz fallback para gpt-4o-mini quando o campo está vazio.",
        ],
    },
    OLLAMA: {
        summary: "Pronto para o runtime principal em implantações auto-hospedadas, com chat nativo via Ollama, padrões locais e descoberta de modelos ao vivo.",
        notes: [
            "Ollama local funciona sem api_key; Ollama Cloud exige api_key.",
            "O modelo precisa ser preenchido explicitamente nas configurações salvas, mesmo que o runtime sugira llama3.1:8b.",
        ],
    },
    DEEP_SEEK: {
        summary: "Suportado pelo adaptador compartilhado compatível com OpenAI, com padrões da DeepSeek, verificações de prontidão e validação ao vivo opcional.",
        notes: [
            "O comportamento específico do provider ainda trafega pelo cliente compatível com OpenAI compartilhado, e não por um transporte dedicado da DeepSeek.",
            "Adequado para uso controlado em indexação e chat, mas não é apresentado como caminho de runtime de primeira classe acima de OpenAI/Ollama.",
        ],
    },
    GROK: {
        summary: "Suportado pelo adaptador compatível com OpenAI compartilhado, com padrões da xAI, verificações de prontidão e validação ao vivo opcional.",
        notes: [
            "Usa o mesmo caminho de transporte compatível com OpenAI dos demais providers em estilo bridge.",
            "Mantenha a validação restrita a implantações concretas em vez de assumir paridade com todos os recursos da xAI.",
        ],
    },
    OPENROUTER: {
        summary: "Suportado pelo caminho de gateway compatível com OpenAI, com verificações de prontidão, descoberta de modelos e validação ao vivo opcional.",
        notes: [
            "Exige um id de modelo explícito nas configurações salvas, mesmo que o runtime sugira openai/gpt-4o-mini.",
            "É melhor tratado como uma camada de gateway cujo comportamento upstream depende do modelo roteado selecionado.",
        ],
    },
    GOOGLE_GEMINI: {
        summary: "Suporte experimental apenas via bridge. O runtime pode funcionar quando Gemini é exposto atrás de um endpoint compatível com OpenAI, com modelo e base_url explícitos.",
        notes: [
            "A integração nativa com a API Gemini não está implementada no runtime principal.",
            "Trate isso como validação apoiada em bridge, e não como transporte Gemini de primeira classe.",
        ],
    },
    CLAUDE: {
        summary: "Suporte experimental apenas via bridge. O runtime pode funcionar quando Claude é exposto por um endpoint compatível com OpenAI, com modelo e base_url explícitos.",
        notes: [
            "A integração nativa com a API da Anthropic não está implementada no runtime principal.",
            "Use apenas quando você controlar o comportamento da bridge e puder validar diretamente o endpoint implantado.",
        ],
    },
};

function translateList(items: string[], map: Record<string, string>) {
    return items.map((item) => map[item] ?? item);
}

function translateValue(value: string, map: Record<string, string>) {
    return map[value] ?? value;
}

export function localizeRuntimeProfile(profile: RuntimeProfile, locale: AppLocale): RuntimeProfile {
    if (locale !== "pt-BR") {
        return profile;
    }

    return {
        ...profile,
        tagline: "Inteligência documental relacional para conhecimento operacional auditável.",
        primary_use_case:
            "Organizar coleções de documentos operacionais, de compliance, jurídicos e de pesquisa com indexação relacional tipada, contexto recuperável e fluxos grounded de acompanhamento.",
        core_problem:
            "Times com grande volume documental costumam ter storage, metadados e respostas de IA espalhados em ferramentas desconectadas, o que dificulta recuperação de contexto, proveniência e auditoria operacional.",
        value_proposition:
            "ThingIndex mantém organização documental, extração estruturada, recuperação de contexto e portabilidade entre provedores em um único runtime relacional, que pode ser auto-hospedado ou consumido como API sem esconder proveniência atrás de uma stack centrada em vetores.",
        target_users: profile.target_users.map((item) => ptRuntimeProfileAudienceMap[item.name] ?? item),
        current_scope: translateList(profile.current_scope, ptRuntimeProfileCurrentScopeMap),
        differentiators: translateList(profile.differentiators, ptRuntimeProfileDifferentiatorsMap),
        non_goals: translateList(profile.non_goals, ptRuntimeProfileNonGoalsMap),
        research_tracks: translateList(profile.research_tracks, ptRuntimeProfileResearchTracksMap),
        artifact_refs: profile.artifact_refs.map((artifact) => ({
            ...artifact,
            label: ptRuntimeProfileArtifactsByPath[artifact.path]?.label ?? artifact.label,
            summary: ptRuntimeProfileArtifactsByPath[artifact.path]?.summary ?? artifact.summary,
        })),
    };
}

export function localizeIndexingBenchmark(report: IndexingBenchmarkReport, locale: AppLocale): IndexingBenchmarkReport {
    if (locale !== "pt-BR") {
        return report;
    }

    return {
        ...report,
        summary: ptIndexingBenchmarkNarrativeMap[report.summary] ?? report.summary,
        methodology: translateList(report.methodology, ptIndexingBenchmarkNarrativeMap),
        pass_criteria: translateList(report.pass_criteria, ptIndexingBenchmarkNarrativeMap),
        limitations: translateList(report.limitations, ptIndexingBenchmarkNarrativeMap),
        heuristic_quality_note:
            ptIndexingBenchmarkNarrativeMap[report.heuristic_quality_note] ?? report.heuristic_quality_note,
        corpus: report.corpus.map((item) => ({
            ...item,
            title: translateValue(item.title, ptIndexingBenchmarkDocumentTitleMap),
            focus: ptIndexingBenchmarkFocusByDocumentID[item.id] ?? item.focus,
            expected_outline_titles: translateList(item.expected_outline_titles, ptIndexingBenchmarkNodeTitleMap),
        })),
        cases: report.cases.map((item) => ({
            ...item,
            query: ptIndexingBenchmarkQueryByCaseID[item.id] ?? item.query,
            document_title: translateValue(item.document_title, ptIndexingBenchmarkDocumentTitleMap),
            expected_document_title: translateValue(item.expected_document_title, ptIndexingBenchmarkDocumentTitleMap),
            actual_document_title: translateValue(item.actual_document_title, ptIndexingBenchmarkDocumentTitleMap),
            expected_node_title: translateValue(item.expected_node_title, ptIndexingBenchmarkNodeTitleMap),
            actual_node_title: translateValue(item.actual_node_title, ptIndexingBenchmarkNodeTitleMap),
        })),
        retrieval_comparison: report.retrieval_comparison
            ? {
                ...report.retrieval_comparison,
                summary:
                    ptIndexingBenchmarkNarrativeMap[report.retrieval_comparison.summary] ?? report.retrieval_comparison.summary,
                notes: translateList(report.retrieval_comparison.notes, ptIndexingBenchmarkNarrativeMap),
                variants: report.retrieval_comparison.variants.map((variant) => ({
                    ...variant,
                    label: ptIndexingBenchmarkRetrievalVariantLabelMap[variant.label] ?? variant.label,
                    summary: ptIndexingBenchmarkNarrativeMap[variant.summary] ?? variant.summary,
                })),
            }
            : undefined,
    };
}

export function localizeBenchmarkScope(scope: string, locale: AppLocale): string {
    if (locale !== "pt-BR") {
        return scope;
    }

    if (scope === "DOCUMENT_TREE") {
        return "Árvore documental";
    }

    if (scope === "COLLECTION") {
        return "Coleção";
    }

    return scope;
}

export function localizeProviderSupportItems(items: ProviderSupport[], locale: AppLocale): ProviderSupport[] {
    if (locale !== "pt-BR") {
        return items;
    }

    return items.map((item) => {
        const copy = ptProviderSupportByType[item.provider_type];
        if (!copy) {
            return item;
        }

        return {
            ...item,
            display_name: copy.display_name ?? item.display_name,
            summary: copy.summary,
            notes: copy.notes,
            suggested_setup: Object.fromEntries(
                Object.entries(item.suggested_setup).map(([key, value]) => [key, ptProviderSetupValueMap[value] ?? value])
            ),
        };
    });
}
