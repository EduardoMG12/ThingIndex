"use client";

import { motion, type Transition } from "framer-motion";
import {
    BookOpen,
    Code2,
    Database,
    GitBranch,
    Network,
    ScrollText,
    ServerCog,
    ShieldCheck,
    Target,
    Workflow,
} from "lucide-react";

import { OpenSourceOnboardingSection } from "@/components/documentation/OpenSourceOnboardingSection";
import { IndexingBenchmarkSection } from "@/components/documentation/IndexingBenchmarkSection";
import { ProductFocusSection } from "@/components/documentation/ProductFocusSection";
import { ProviderSupportSection } from "@/components/documentation/ProviderSupportSection";
import AppShell from "@/components/layout/AppShell";
import { Section } from "@/components/documentation/Section";
import { TableOfContents } from "@/components/documentation/TableOfContents";

const fadeUp = {
    initial: { opacity: 0, y: 16 },
    animate: { opacity: 1, y: 0 },
    transition: {
        duration: 0.5,
        ease: [0.22, 1, 0.36, 1],
    } as Transition,
};

export default function DocumentationPage() {
    return (
        <AppShell>
            <motion.section
                {...fadeUp}
                className="relative mb-12 overflow-hidden rounded-[28px] border border-primary/15 bg-card/85 px-8 py-10 shadow-card"
            >
                <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(245,166,35,0.16),transparent_22%),radial-gradient(circle_at_bottom_right,rgba(82,168,255,0.1),transparent_28%)]" />
                <div className="relative">
                    <div className="flex flex-wrap items-center gap-2">
                        <span className="rounded-full border border-primary/20 bg-primary/10 px-3 py-1 font-mono text-[10px] uppercase tracking-[0.24em] text-primary">
                            official docs
                        </span>
                        <span className="rounded-full border border-border bg-background/70 px-3 py-1 font-mono text-[10px] uppercase tracking-[0.24em] text-muted-foreground">
                            current runtime
                        </span>
                        <span className="rounded-full border border-border bg-background/70 px-3 py-1 font-mono text-[10px] uppercase tracking-[0.24em] text-muted-foreground">
                            research clearly marked
                        </span>
                    </div>

                    <h1 className="mt-6 max-w-4xl font-display text-4xl font-bold tracking-tight text-foreground sm:text-5xl">
                        Documentação de engenharia alinhada ao
                        <span className="text-primary"> estado atual do projeto</span>
                    </h1>

                    <p className="mt-5 max-w-3xl text-base leading-relaxed text-muted-foreground">
                        O ThingIndex hoje é uma plataforma relacional de inteligência documental. O foco do código atual está
                        em autenticação, upload, organização por categoria e collection, indexação estrutural e recuperação de
                        contexto em PostgreSQL. Os materiais secundários de pesquisa continuam existindo, mas agora ficam
                        separados explicitamente do runtime principal.
                    </p>

                    <div className="mt-8 grid gap-4 lg:grid-cols-4">
                        {[
                            { label: "Escopo", value: "Catálogo documental relacional" },
                            { label: "Stack", value: "Next.js + Go API + PostgreSQL + MinIO" },
                            { label: "Processamento", value: "Indexação estrutural com evolução assíncrona" },
                            { label: "Meta central", value: "Contexto auditável e reproduzível" },
                        ].map((item) => (
                            <div key={item.label} className="rounded-2xl border border-border bg-background/60 p-4">
                                <p className="text-[10px] font-bold uppercase tracking-[0.22em] text-muted-foreground">{item.label}</p>
                                <p className="mt-2 text-sm font-medium leading-relaxed text-foreground/90">{item.value}</p>
                            </div>
                        ))}
                    </div>
                </div>
            </motion.section>

            <div className="grid gap-8 lg:grid-cols-[250px_1fr]">
                <motion.nav
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: 0.15, duration: 0.5 }}
                    className="hidden lg:block"
                >
                    <TableOfContents />
                </motion.nav>

                <div className="space-y-10">
                    <Section icon={Target} title="Visão Geral" id="overview" delay={0.1}>
                        <p>
                            O <strong className="text-foreground">ThingIndex</strong> preserva o contexto documental por meio de
                            modelagem relacional explícita. Em vez de depender de um grafo operacional ou de busca vetorial
                            como núcleo do produto, o projeto atual organiza o domínio com categorias, collections, documentos e
                            estruturas extraídas persistidas em tabelas tipadas.
                        </p>
                        <p className="mt-4">
                            Isso torna a recuperação mais previsível, a auditoria mais simples e a defesa técnica do TCC mais
                            consistente com o que o repositório realmente entrega hoje.
                        </p>
                    </Section>

                    <Section icon={Target} title="Foco do Produto" id="product-focus" delay={0.12}>
                        <ProductFocusSection />
                    </Section>

                    <Section icon={BookOpen} title="Princípios de Projeto" id="principles" delay={0.15}>
                        <div className="grid gap-4 md:grid-cols-2">
                            {[
                                {
                                    title: "Relational first",
                                    body: "Collections, categorias, documentos e estruturas tipadas são a base do sistema.",
                                },
                                {
                                    title: "Thin HTTP boundary",
                                    body: "Handlers lidam com validação, auth e serialização; regras ficam em use cases e repositórios.",
                                },
                                {
                                    title: "Provider agnostic",
                                    body: "Storage e LLM podem ser trocados sem reescrever a lógica principal.",
                                },
                                {
                                    title: "Academic rigor",
                                    body: "A implementação precisa refletir com honestidade o que é código atual e o que ainda é roadmap.",
                                },
                            ].map((item) => (
                                <div key={item.title} className="rounded-xl border border-border bg-background/50 p-5">
                                    <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                                    <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{item.body}</p>
                                </div>
                            ))}
                        </div>
                    </Section>

                    <Section icon={Network} title="Arquitetura do Sistema" id="architecture" delay={0.2}>
                        <p className="mb-6">
                            A arquitetura separa interface, API, domínio, persistência e processamento documental. Essa divisão
                            reduz acoplamento e facilita evolução incremental.
                        </p>

                        <div className="grid gap-4 lg:grid-cols-2">
                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">Camadas</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    <li><strong className="text-foreground">Frontend:</strong> autenticação, dashboard, preview e documentação.</li>
                                    <li><strong className="text-foreground">API:</strong> upload, providers, documentos e recuperação de contexto.</li>
                                    <li><strong className="text-foreground">Processamento:</strong> indexação estrutural e enriquecimento opcional.</li>
                                    <li><strong className="text-foreground">Persistência:</strong> PostgreSQL e object storage compatível com S3.</li>
                                </ul>
                            </div>

                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">Objetivos arquiteturais</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    <li>Preservar contexto com joins previsíveis.</li>
                                    <li>Evitar N+1 e montagem quadrática na API.</li>
                                    <li>Permitir portabilidade de providers e ambientes.</li>
                                    <li>Manter consistência entre preview, indexação e contexto consolidado.</li>
                                </ul>
                            </div>
                        </div>

                        <div className="mt-6 rounded-xl border border-border bg-zinc-950 p-6 font-mono text-[10px] leading-relaxed text-zinc-400 overflow-x-auto whitespace-pre custom-scrollbar">
{`[Client Workspace]
  -> auth / dashboard / preview / settings

[Go API]
  -> auth
  -> upload + metadata
  -> collection context
  -> provider configuration

[Processing]
  -> outbox dispatch
  -> document parsing
  -> structured indexing

[Persistence]
  -> object storage
  -> relational store`}
                        </div>
                    </Section>

                    <Section icon={Workflow} title="Processamento Atual" id="processing" delay={0.25}>
                        <p className="mb-6">
                            O repositório já fecha o ciclo principal de upload, processamento assíncrono, indexação estrutural
                            e recuperação do contexto salvo. A tela documental já expõe leitura executiva, mapa de anomalias
                            e próximos passos operacionais; o trabalho remanescente agora é expandir portabilidade e lapidar
                            pricing, não provar que o pipeline existe.
                        </p>

                        <div className="space-y-4">
                            {[
                                {
                                    step: "01",
                                    title: "Upload e persistência",
                                    desc: "O documento é validado, armazenado e vinculado a uma collection obrigatória.",
                                },
                                {
                                    step: "02",
                                    title: "Despacho opcional",
                                    desc: "O backend pode registrar eventos em outbox e despachar indexação sem bloquear a interface.",
                                },
                                {
                                    step: "03",
                                    title: "Indexação estrutural",
                                    desc: "Parágrafos, tabelas, outline e metadados são persistidos como estruturas tipadas.",
                                },
                                {
                                    step: "04",
                                    title: "Consulta do contexto",
                                    desc: "O frontend consome o contexto consolidado por collection e documento para navegação e análise.",
                                },
                            ].map((item) => (
                                <div key={item.step} className="rounded-xl border border-border bg-background/50 p-5">
                                    <div className="flex items-center gap-3">
                                        <span className="rounded-md bg-primary/10 px-2 py-1 text-[10px] font-bold uppercase tracking-[0.2em] text-primary">
                                            {item.step}
                                        </span>
                                        <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                                    </div>
                                    <p className="mt-3 text-sm leading-relaxed text-muted-foreground">{item.desc}</p>
                                </div>
                            ))}
                        </div>
                    </Section>

                    <Section icon={Database} title="Modelo de Dados" id="data-model" delay={0.3}>
                        <p>
                            O banco organiza o domínio em categoria, collection, documento, parágrafo, tabela e metadado
                            tipado. A ordem do conteúdo é mantida no banco e a agregação da resposta ocorre em tempo linear
                            sobre o conjunto carregado.
                        </p>
                    </Section>

                    <Section icon={Database} title="Recuperação de Contexto" id="retrieval" delay={0.35}>
                        <p>
                            O diferencial implementado hoje é a recuperação de contexto relacional. A API carrega a collection,
                            busca os documentos relacionados, agrega estruturas tipadas por lote e devolve uma visão consolidada
                            sem depender de similaridade vetorial para a navegação principal.
                        </p>
                    </Section>

                    <Section icon={Target} title="Benchmark de Indexação" id="indexing-benchmark" delay={0.38}>
                        <IndexingBenchmarkSection />
                    </Section>

                    <Section icon={ShieldCheck} title="Segurança e Governança" id="security" delay={0.4}>
                        <p>
                            O projeto já inclui autenticação JWT, escopo por usuário, respostas padronizadas, validação de upload
                            e criptografia de credenciais de providers. Medidas adicionais podem ser incorporadas depois, mas não
                            são vendidas aqui como se já estivessem no runtime atual.
                        </p>
                    </Section>

                    <Section icon={ServerCog} title="Deploy e Operação" id="deployment" delay={0.45}>
                        <p>
                            O ambiente local usa Docker Compose com frontend, backend, PostgreSQL, MinIO e RabbitMQ. O RabbitMQ
                            já existe no repositório, mas ainda não é requisito obrigatório de todo o caminho principal.
                        </p>
                    </Section>

                    <Section icon={Code2} title="Providers e Portabilidade" id="providers" delay={0.5}>
                        <ProviderSupportSection />
                    </Section>

                    <Section icon={BookOpen} title="Onboarding Open Source" id="open-source" delay={0.55}>
                        <OpenSourceOnboardingSection apiBaseUrl={process.env.NEXT_PUBLIC_API_URL} />
                    </Section>

                    <Section icon={GitBranch} title="Roadmap" id="roadmap" delay={0.6}>
                        <ul className="space-y-3">
                            <li>Expandir storages realmente exercitados e publicar a CLI para importacao de indices externos.</li>
                            <li>Refinar a leitura de custos e manter a UX alinhada ao nivel real de maturidade do runtime.</li>
                            <li>Reduzir boilerplate, naming excessivo e polimentos visuais menores sem inflar o escopo principal.</li>
                            <li>Manter comparacoes com GraphRAG e abordagens vetoriais como trilhas academicas separadas do produto principal.</li>
                        </ul>
                    </Section>

                    <Section icon={ScrollText} title="TCC e Artigo" id="academic" delay={0.65}>
                        <p>
                            O repositório também serve como artefato de pesquisa. A principal exigência aqui é clareza: separar
                            rigorosamente implementação atual, hipótese de pesquisa e evolução futura. A proposta de continuação
                            e os roadmaps em <code>docs/</code> devem ser lidos como pesquisa derivada, não como descrição do runtime.
                        </p>
                    </Section>

                    <Section icon={GitBranch} title="Contribuição" id="contributing" delay={0.7}>
                        <p>
                            As contribuições mais úteis agora são pequenas e verificáveis: testes, fixtures, melhorias no
                            dashboard documental e documentação alinhada ao runtime atual.
                        </p>
                    </Section>

                    <Section icon={ScrollText} title="Referências e Materiais" id="references" delay={0.75}>
                        <div className="space-y-4 text-muted-foreground">
                            <p>
                                O documento principal para visão completa do projeto continua sendo o <code>README.md</code> na
                                raiz do monorepo. A leitura recomendada do repositório agora fica separada em camadas claras.
                            </p>
                            <ul className="space-y-3">
                                <li><strong className="text-foreground">Runtime atual:</strong> README raiz, <code>api/README.md</code>, <code>frontend/README.md</code> e esta página.</li>
                                <li><strong className="text-foreground">Prontidão atual:</strong> <code>docs/functional-readiness-spec.md</code>.</li>
                                <li><strong className="text-foreground">Pesquisa e roadmap:</strong> <code>docs/proposta-continuacao-tcc.md</code> e <code>docs/roadmap/</code>, sempre tratados como trilhas acadêmicas.</li>
                                <li><strong className="text-foreground">Índice das docs secundárias:</strong> <code>docs/README.md</code>.</li>
                            </ul>
                        </div>
                    </Section>
                </div>
            </div>
        </AppShell>
    );
}
