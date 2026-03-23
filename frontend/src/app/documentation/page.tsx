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
import { DocumentationSection } from "@/components/documentation/DocumentationSection";
import { TableOfContents } from "@/components/documentation/TableOfContents";
import {
    DOCUMENTATION_ARCHITECTURE_GOALS,
    DOCUMENTATION_ARCHITECTURE_LAYERS,
    DOCUMENTATION_ARCHITECTURE_SNAPSHOT,
    DOCUMENTATION_HERO_SNAPSHOT,
    DOCUMENTATION_PRINCIPLES,
    DOCUMENTATION_PROCESSING_STEPS,
    DOCUMENTATION_REFERENCE_TRACKS,
    DOCUMENTATION_ROADMAP_ITEMS,
} from "@/components/documentation/page-content";

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
                        {DOCUMENTATION_HERO_SNAPSHOT.map((item) => (
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
                    <DocumentationSection icon={Target} title="Visão Geral" id="overview" delay={0.1}>
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
                    </DocumentationSection>

                    <DocumentationSection icon={Target} title="Foco do Produto" id="product-focus" delay={0.12}>
                        <ProductFocusSection />
                    </DocumentationSection>

                    <DocumentationSection icon={BookOpen} title="Princípios de Projeto" id="principles" delay={0.15}>
                        <div className="grid gap-4 md:grid-cols-2">
                            {DOCUMENTATION_PRINCIPLES.map((item) => (
                                <div key={item.title} className="rounded-xl border border-border bg-background/50 p-5">
                                    <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                                    <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{item.body}</p>
                                </div>
                            ))}
                        </div>
                    </DocumentationSection>

                    <DocumentationSection icon={Network} title="Arquitetura do Sistema" id="architecture" delay={0.2}>
                        <p className="mb-6">
                            A arquitetura separa interface, API, domínio, persistência e processamento documental. Essa divisão
                            reduz acoplamento e facilita evolução incremental.
                        </p>

                        <div className="grid gap-4 lg:grid-cols-2">
                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">Camadas</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    {DOCUMENTATION_ARCHITECTURE_LAYERS.map((item) => (
                                        <li key={item.label}>
                                            <strong className="text-foreground">{item.label}:</strong> {item.description}
                                        </li>
                                    ))}
                                </ul>
                            </div>

                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">Objetivos arquiteturais</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    {DOCUMENTATION_ARCHITECTURE_GOALS.map((item) => (
                                        <li key={item}>{item}</li>
                                    ))}
                                </ul>
                            </div>
                        </div>

                        <div className="mt-6 rounded-xl border border-border bg-zinc-950 p-6 font-mono text-[10px] leading-relaxed text-zinc-400 overflow-x-auto whitespace-pre custom-scrollbar">
                            {DOCUMENTATION_ARCHITECTURE_SNAPSHOT}
                        </div>
                    </DocumentationSection>

                    <DocumentationSection icon={Workflow} title="Processamento Atual" id="processing" delay={0.25}>
                        <p className="mb-6">
                            O repositório já fecha o ciclo principal de upload, processamento assíncrono, indexação estrutural
                            e recuperação do contexto salvo. A tela documental já expõe leitura executiva, mapa de anomalias
                            e próximos passos operacionais, enquanto o dashboard já cruza pricing sincronizado com telemetria
                            real para leitura honesta de custos. O backlog principal agora está fechado; o que sobra é
                            manutenção incremental e pesquisa derivada, não provar que o pipeline existe.
                        </p>

                        <div className="space-y-4">
                            {DOCUMENTATION_PROCESSING_STEPS.map((item) => (
                                <div key={item.step} className="rounded-xl border border-border bg-background/50 p-5">
                                    <div className="flex items-center gap-3">
                                        <span className="rounded-md bg-primary/10 px-2 py-1 text-[10px] font-bold uppercase tracking-[0.2em] text-primary">
                                            {item.step}
                                        </span>
                                        <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                                    </div>
                                    <p className="mt-3 text-sm leading-relaxed text-muted-foreground">{item.description}</p>
                                </div>
                            ))}
                        </div>
                    </DocumentationSection>

                    <DocumentationSection icon={Database} title="Modelo de Dados" id="data-model" delay={0.3}>
                        <p>
                            O banco organiza o domínio em categoria, collection, documento, parágrafo, tabela e metadado
                            tipado. A ordem do conteúdo é mantida no banco e a agregação da resposta ocorre em tempo linear
                            sobre o conjunto carregado.
                        </p>
                    </DocumentationSection>

                    <DocumentationSection icon={Database} title="Recuperação de Contexto" id="retrieval" delay={0.35}>
                        <p>
                            O diferencial implementado hoje é a recuperação de contexto relacional. A API carrega a collection,
                            busca os documentos relacionados, agrega estruturas tipadas por lote e devolve uma visão consolidada
                            sem depender de similaridade vetorial para a navegação principal.
                        </p>
                    </DocumentationSection>

                    <DocumentationSection icon={Target} title="Benchmark de Indexação" id="indexing-benchmark" delay={0.38}>
                        <IndexingBenchmarkSection />
                    </DocumentationSection>

                    <DocumentationSection icon={ShieldCheck} title="Segurança e Governança" id="security" delay={0.4}>
                        <p>
                            O projeto já inclui autenticação JWT, escopo por usuário, respostas padronizadas, validação de upload
                            e criptografia de credenciais de providers. Medidas adicionais podem ser incorporadas depois, mas não
                            são vendidas aqui como se já estivessem no runtime atual.
                        </p>
                    </DocumentationSection>

                    <DocumentationSection icon={ServerCog} title="Deploy e Operação" id="deployment" delay={0.45}>
                        <p>
                            O ambiente local usa Docker Compose com frontend, backend, PostgreSQL, MinIO e RabbitMQ. O RabbitMQ
                            já existe no repositório, mas ainda não é requisito obrigatório de todo o caminho principal.
                        </p>
                    </DocumentationSection>

                    <DocumentationSection icon={Code2} title="Providers e Portabilidade" id="providers" delay={0.5}>
                        <ProviderSupportSection />
                    </DocumentationSection>

                    <DocumentationSection icon={BookOpen} title="Onboarding Open Source" id="open-source" delay={0.55}>
                        <OpenSourceOnboardingSection apiBaseUrl={process.env.NEXT_PUBLIC_API_URL} />
                    </DocumentationSection>

                    <DocumentationSection icon={GitBranch} title="Roadmap" id="roadmap" delay={0.6}>
                        <ul className="space-y-3">
                            {DOCUMENTATION_ROADMAP_ITEMS.map((item) => (
                                <li key={item}>{item}</li>
                            ))}
                        </ul>
                    </DocumentationSection>

                    <DocumentationSection icon={ScrollText} title="TCC e Artigo" id="academic" delay={0.65}>
                        <p>
                            O repositório também serve como artefato de pesquisa. A principal exigência aqui é clareza: separar
                            rigorosamente implementação atual, hipótese de pesquisa e evolução futura. A proposta de continuação
                            e os roadmaps em <code>docs/</code> devem ser lidos como pesquisa derivada, não como descrição do runtime.
                        </p>
                    </DocumentationSection>

                    <DocumentationSection icon={GitBranch} title="Contribuição" id="contributing" delay={0.7}>
                        <p>
                            As contribuições mais úteis agora são pequenas e verificáveis: testes, fixtures, melhorias no
                            dashboard documental e documentação alinhada ao runtime atual.
                        </p>
                    </DocumentationSection>

                    <DocumentationSection icon={ScrollText} title="Referências e Materiais" id="references" delay={0.75}>
                        <div className="space-y-4 text-muted-foreground">
                            <p>
                                O documento principal para visão completa do projeto continua sendo o <code>README.md</code> na
                                raiz do monorepo. A leitura recomendada do repositório agora fica separada em camadas claras.
                            </p>
                            <ul className="space-y-3">
                                {DOCUMENTATION_REFERENCE_TRACKS.map((item) => (
                                    <li key={item.label}>
                                        <strong className="text-foreground">{item.label}:</strong>{" "}
                                        {item.references.map((reference, index) => (
                                            <span key={`${item.label}-${reference}`}>
                                                {index > 0 ? ", " : null}
                                                {reference === "esta página" ? (
                                                    reference
                                                ) : (
                                                    <code>{reference}</code>
                                                )}
                                            </span>
                                        ))}
                                        {item.label === "Pesquisa e roadmap"
                                            ? ", sempre tratados como trilhas acadêmicas."
                                            : "."}
                                    </li>
                                ))}
                            </ul>
                        </div>
                    </DocumentationSection>
                </div>
            </div>
        </AppShell>
    );
}
