"use client";

import { motion, type Transition } from "framer-motion";
import {
    ArrowRight,
    BookOpen,
    Brain,
    Code2,
    Database,
    DollarSign,
    ExternalLink,
    GitBranch,
    Lightbulb,
    Network,
    Rocket,
    ScrollText,
    ServerCog,
    ShieldCheck,
    Target,
    Workflow,
} from "lucide-react";

import AppShell from "@/components/layout/AppShell";
import { Button } from "@/components/ui/button";
import { ComparisonTable } from "@/components/documentation/ComparisonTable";
import { PricingTable } from "@/components/documentation/PricingTable";
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
                            open source mindset
                        </span>
                        <span className="rounded-full border border-border bg-background/70 px-3 py-1 font-mono text-[10px] uppercase tracking-[0.24em] text-muted-foreground">
                            tcc / article / research
                        </span>
                    </div>

                    <h1 className="mt-6 max-w-4xl font-display text-4xl font-bold tracking-tight text-foreground sm:text-5xl">
                        Documentação de engenharia para um projeto universitário com ambição de
                        <span className="text-primary"> plataforma open source</span>
                    </h1>

                    <p className="mt-5 max-w-3xl text-base leading-relaxed text-muted-foreground">
                        O ThingIndex foi concebido como um sistema de inteligência documental com RAG, ingestão estruturada,
                        vetorização, rastreabilidade e flexibilidade operacional. Esta documentação descreve o projeto como
                        software, como artefato acadêmico e como base de evolução contínua.
                    </p>

                    <div className="mt-8 grid gap-4 lg:grid-cols-4">
                        {[
                            { label: "Escopo", value: "RAG documental com fluxo estruturado" },
                            { label: "Domínio", value: "Jurídico, pesquisa, compliance e operações" },
                            { label: "Arquitetura", value: "API + workers + storage + vetores + grafo" },
                            { label: "Meta central", value: "Contexto auditável e portável" },
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
                            O <strong className="text-foreground">ThingIndex</strong> é uma plataforma de inteligência documental
                            baseada em Retrieval-Augmented Generation, desenvolvida para preservar a estrutura do documento e
                            transformar conteúdo bruto em contexto utilizável por sistemas de IA.
                        </p>
                        <p className="mt-4">
                            Em vez de tratar um PDF como texto contínuo, o sistema modela o documento como uma unidade
                            composta por páginas, parágrafos, tabelas, metadados, agrupamentos e referências persistentes.
                            Essa decisão afeta diretamente a qualidade da indexação, a legibilidade da recuperação e a defesa
                            metodológica do projeto no contexto do TCC.
                        </p>

                        <div className="mt-6 grid gap-4 md:grid-cols-3">
                            {[
                                {
                                    title: "Problema",
                                    body: "Soluções RAG genéricas perdem contexto estrutural e oferecem pouca rastreabilidade por trecho ou etapa do pipeline.",
                                },
                                {
                                    title: "Hipótese",
                                    body: "Um pipeline multiestágio com extração estruturada melhora a qualidade do contexto entregue à IA.",
                                },
                                {
                                    title: "Objetivo",
                                    body: "Construir um sistema academicamente defensável, tecnicamente consistente e pronto para evolução open source.",
                                },
                            ].map((item) => (
                                <div key={item.title} className="rounded-xl border border-border bg-background/50 p-4">
                                    <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-primary">{item.title}</p>
                                    <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{item.body}</p>
                                </div>
                            ))}
                        </div>
                    </Section>

                    <Section icon={Brain} title="Princípios de Projeto" id="principles" delay={0.15}>
                        <div className="grid gap-4 md:grid-cols-2">
                            {[
                                {
                                    title: "Structure first",
                                    body: "A estrutura documental é tratada como sinal de alto valor semântico, não como detalhe descartável.",
                                },
                                {
                                    title: "Provider agnostic",
                                    body: "Storage e LLMs devem poder ser substituídos sem reescrever a lógica principal da aplicação.",
                                },
                                {
                                    title: "Operational traceability",
                                    body: "Cada etapa do fluxo precisa ser observável e explicável, do upload ao documento indexado.",
                                },
                                {
                                    title: "Academic rigor",
                                    body: "As decisões precisam ser reproduzíveis, justificáveis e compatíveis com defesa técnica e produção de artigo.",
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
                            A arquitetura do ThingIndex foi organizada para separar ingestão, orquestração, processamento,
                            persistência e consumo de contexto. Essa divisão reduz acoplamento e melhora a capacidade de evolução do projeto.
                        </p>

                        <div className="grid gap-4 lg:grid-cols-2">
                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">Camadas</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    <li><strong className="text-foreground">Frontend:</strong> workspace, preview, providers e documentação.</li>
                                    <li><strong className="text-foreground">API:</strong> autenticação, upload, preview, metadados e acionamento de jobs.</li>
                                    <li><strong className="text-foreground">Workers:</strong> leitura, estruturação, sumarização e vetorização.</li>
                                    <li><strong className="text-foreground">Persistência:</strong> object storage, banco relacional, store vetorial e base relacional/grafo.</li>
                                </ul>
                            </div>

                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">Objetivos arquiteturais</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    <li>Escalar processamento sem degradar a API interativa.</li>
                                    <li>Manter fronteiras claras entre domínio e infraestrutura.</li>
                                    <li>Permitir portabilidade de providers e ambientes.</li>
                                    <li>Preservar consistência entre preview, indexação e consulta.</li>
                                </ul>
                            </div>
                        </div>

                        <div className="mt-6 rounded-xl border border-border bg-zinc-950 p-6 font-mono text-[10px] leading-relaxed text-zinc-400 overflow-x-auto whitespace-pre custom-scrollbar">
{`[Client Workspace]
  -> navigation / auth / provider settings / document preview

[Next App Routes]
  -> internal authenticated proxies
  -> locale-aware routes and SEO surface

[Core API]
  -> auth
  -> upload + metadata
  -> content preview
  -> provider configuration

[Async Processing]
  -> queue dispatch
  -> document parsing
  -> summary / outline / metadata / embeddings

[Persistence]
  -> object storage
  -> relational store
  -> vector index
  -> future graph-oriented expansion`}
                        </div>
                    </Section>

                    <Section icon={Workflow} title="Pipeline de Processamento" id="pipeline" delay={0.25}>
                        <p className="mb-6">
                            O pipeline é assíncrono por definição. Isso permite baixa latência na API e processamento mais
                            pesado em background, com etapas bem definidas para análise e auditoria.
                        </p>

                        <div className="space-y-4">
                            {[
                                {
                                    step: "01",
                                    title: "Ingestão e persistência inicial",
                                    desc: "O arquivo é recebido, validado, classificado e armazenado de forma consistente com o contexto do usuário.",
                                },
                                {
                                    step: "02",
                                    title: "Despacho para processamento",
                                    desc: "A API registra o documento e entrega o trabalho a um worker sem bloquear a interface principal.",
                                },
                                {
                                    step: "03",
                                    title: "Reconstrução estrutural",
                                    desc: "O worker identifica páginas, trechos, tabelas, outline e metadados relevantes para navegação e indexação.",
                                },
                                {
                                    step: "04",
                                    title: "Enriquecimento semântico",
                                    desc: "Prompts especializados geram artefatos intermediários reutilizáveis, como resumo, estrutura e contexto enriquecido.",
                                },
                                {
                                    step: "05",
                                    title: "Indexação e disponibilização",
                                    desc: "Os resultados finais são persistidos e passam a alimentar preview, vetorização e futuros fluxos de query.",
                                },
                            ].map((item) => (
                                <div key={item.step} className="group relative pl-12">
                                    <div className="absolute left-0 top-0 flex h-8 w-8 items-center justify-center rounded-lg bg-secondary border border-border font-mono text-[10px] font-bold text-muted-foreground group-hover:border-primary group-hover:text-primary transition-colors">
                                        {item.step}
                                    </div>
                                    <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                                    <p className="mt-1 text-sm leading-relaxed text-muted-foreground">{item.desc}</p>
                                </div>
                            ))}
                        </div>
                    </Section>

                    <Section icon={Database} title="Modelo de Dados e Organização" id="data-model" delay={0.3}>
                        <p className="mb-6">
                            O domínio foi estruturado para refletir o modo como operadores humanos e sistemas automáticos
                            precisam enxergar o acervo: agrupado, contextualizado e com navegação consistente.
                        </p>

                        <div className="grid gap-4 lg:grid-cols-[1fr_0.9fr]">
                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">Hierarquia canônica</h4>
                                <div className="mt-4 rounded-lg border border-border bg-card/70 p-4 font-mono text-[11px] text-muted-foreground">
                                    Categoria → Collection → Documento → Página → Seção/Chunk/Tabela → Metadados → Embedding
                                </div>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    <li><strong className="text-foreground">Categoria:</strong> classificação macro do documento.</li>
                                    <li><strong className="text-foreground">Collection:</strong> agrupamento relacional e contextual entre documentos.</li>
                                    <li><strong className="text-foreground">Documento:</strong> unidade principal de ingestão, preview e indexação.</li>
                                    <li><strong className="text-foreground">Nós derivados:</strong> páginas, tabelas, outline, resumo e metadados persistidos.</li>
                                </ul>
                            </div>

                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">Vantagens operacionais</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    <li>Recuperação com filtros por contexto de negócio.</li>
                                    <li>Preview e auditoria humana coerentes com a origem do dado.</li>
                                    <li>Separação entre binário bruto e conhecimento derivado.</li>
                                    <li>Base sólida para evolução de grafos e benchmarks futuros.</li>
                                </ul>
                            </div>
                        </div>
                    </Section>

                    <Section icon={Code2} title="Retrieval, Indexação e Cadeia de Prompts" id="retrieval" delay={0.35}>
                        <p className="mb-6">
                            O ThingIndex não reduz recuperação a similaridade vetorial. A qualidade do retrieval depende do
                            preparo do contexto, da estrutura dos metadados e da forma como o documento foi enriquecido.
                        </p>

                        <div className="grid gap-4 md:grid-cols-3">
                            {[
                                {
                                    title: "Resumo",
                                    color: "border-blue-500/20 bg-blue-500/5",
                                    icon: ScrollText,
                                    desc: "Sintetiza conteúdo relevante e melhora legibilidade operacional do documento indexado.",
                                },
                                {
                                    title: "Estruturação",
                                    color: "border-primary/20 bg-primary/5",
                                    icon: Code2,
                                    desc: "Transforma saídas em objetos coerentes para navegação, inspeção e indexação posterior.",
                                },
                                {
                                    title: "Embeddings",
                                    color: "border-emerald-500/20 bg-emerald-500/5",
                                    icon: Brain,
                                    desc: "Permitem retrieval semântico sobre conteúdo enriquecido, não apenas sobre texto bruto recortado.",
                                },
                            ].map((item) => (
                                <div key={item.title} className={`rounded-xl border p-4 ${item.color}`}>
                                    <item.icon className="h-4 w-4 text-primary" />
                                    <h4 className="mt-3 text-sm font-semibold text-foreground">{item.title}</h4>
                                    <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{item.desc}</p>
                                </div>
                            ))}
                        </div>

                        <div className="mt-8">
                            <ComparisonTable />
                        </div>
                    </Section>

                    <Section icon={ShieldCheck} title="Segurança, Governança e Limites" id="security" delay={0.4}>
                        <div className="grid gap-4 md:grid-cols-2">
                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">Controles presentes</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    <li>Autenticação e fluxo protegido por sessão/token.</li>
                                    <li>Preview desacoplado do provider e isolado por documento.</li>
                                    <li>Gestão de um provider ativo por família para reduzir inconsistência operacional.</li>
                                    <li>Separação entre frontend, proxies autenticados e backend principal.</li>
                                </ul>
                            </div>

                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">Direções futuras</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    <li>Auditoria formal de jobs, retries e mudanças de provider.</li>
                                    <li>Observabilidade por etapa do pipeline.</li>
                                    <li>Políticas explícitas de retenção e versionamento de artefatos.</li>
                                    <li>Testes comparativos para qualidade de retrieval em cenário acadêmico.</li>
                                </ul>
                            </div>
                        </div>
                    </Section>

                    <Section icon={Rocket} title="Deploy, Operação e Ambientes" id="deployment" delay={0.45}>
                        <p className="mb-6">
                            O desenho atual favorece execução local, ambiente universitário controlado e self-hosting. A proposta
                            é manter o projeto simples o suficiente para reproduzir e robusto o bastante para evoluir.
                        </p>

                        <div className="grid gap-4 md:grid-cols-3">
                            {[
                                {
                                    title: "Desenvolvimento",
                                    body: "Volumes locais, iteração rápida e inspeção contínua das saídas intermediárias.",
                                },
                                {
                                    title: "Homologação",
                                    body: "Providers reais, custos controlados, testes de fluxo e inspeção semântica do enriquecimento.",
                                },
                                {
                                    title: "Produção",
                                    body: "Storage resiliente, filas estáveis, métricas, retries e política clara de secrets.",
                                },
                            ].map((item) => (
                                <div key={item.title} className="rounded-xl border border-border bg-background/50 p-5">
                                    <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                                    <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{item.body}</p>
                                </div>
                            ))}
                        </div>
                    </Section>

                    <Section icon={ServerCog} title="Providers, Portabilidade e Evolução" id="providers" delay={0.5}>
                        <p className="mb-6">
                            Uma das linhas mestras do projeto é evitar lock-in desnecessário. Isso aparece principalmente nas
                            abstrações de storage e LLM, que permitem alternância operacional com impacto reduzido no resto da aplicação.
                        </p>

                        <div className="grid gap-4 md:grid-cols-2">
                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">Storage</h4>
                                <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
                                    O sistema suporta uma camada de storage abstrata para acomodar MinIO, S3 e serviços compatíveis.
                                    Isso é importante para pesquisa, ambientes universitários e migração entre infraestruturas.
                                </p>
                            </div>
                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">LLMs</h4>
                                <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
                                    O fluxo de configuração por provider facilita experimentação com modelos, custos e latência,
                                    o que é especialmente útil para benchmarks e escrita acadêmica comparativa.
                                </p>
                            </div>
                        </div>
                    </Section>

                    <Section icon={DollarSign} title="Custos, Benchmark e Comparativos" id="pricing" delay={0.55}>
                        <p className="mb-6">
                            Um projeto sério precisa documentar também o custo operacional. Transparência sobre embeddings, modelos
                            e consumo de contexto é parte da disciplina de engenharia e da honestidade metodológica do TCC.
                        </p>

                        <PricingTable />

                        <div className="mt-8 flex items-center gap-3 rounded-xl border border-border bg-muted/50 p-4 text-[10px] italic text-muted-foreground">
                            <Lightbulb className="h-4 w-4 shrink-0 text-primary" />
                            <span>
                                <strong>Observação:</strong> o indicador mais relevante não é custo isolado, mas custo por contexto útil recuperado
                                com rastreabilidade suficiente para justificar a resposta final.
                            </span>
                        </div>
                    </Section>

                    <Section icon={BookOpen} title="Contexto Acadêmico, TCC e Artigo" id="academic" delay={0.6}>
                        <div className="rounded-xl border border-primary/20 bg-primary/5 p-6">
                            <div className="flex items-center gap-2 text-primary">
                                <BookOpen className="h-4 w-4" />
                                <span className="text-xs font-bold uppercase tracking-[0.2em]">Pesquisa aplicada</span>
                            </div>
                            <p className="mt-4 text-sm leading-relaxed text-muted-foreground">
                                O ThingIndex deve ser entendido tanto como sistema funcional quanto como artefato acadêmico.
                                Isso implica documentar hipóteses, trade-offs, limitações, arquitetura e critérios de avaliação
                                de forma clara o suficiente para banca, artigo e reprodução futura.
                            </p>
                        </div>

                        <div className="mt-6 grid gap-4 md:grid-cols-3">
                            {[
                                "Formalizar o problema e o diferencial frente a pipelines RAG tradicionais.",
                                "Registrar a cadeia de processamento com granularidade compatível com reprodutibilidade.",
                                "Conectar escolhas de engenharia com impacto metodológico e resultados observáveis.",
                            ].map((item) => (
                                <div key={item} className="rounded-xl border border-border bg-background/50 p-4 text-sm leading-relaxed text-muted-foreground">
                                    {item}
                                </div>
                            ))}
                        </div>
                    </Section>

                    <Section icon={GitBranch} title="Contribuição, Roadmap e Próximos Passos" id="contributing" delay={0.65}>
                        <div className="grid gap-4 lg:grid-cols-2">
                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-sm font-semibold text-foreground">Roadmap técnico</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    <li>Expandir suporte a providers com testes integrados e documentação operacional.</li>
                                    <li>Formalizar observabilidade e métricas de qualidade de pipeline.</li>
                                    <li>Adicionar avaliação comparativa de retrieval e grounding.</li>
                                    <li>Publicar guias de deploy, troubleshooting e operação contínua.</li>
                                </ul>
                            </div>

                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-sm font-semibold text-foreground">Padrão de contribuição</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    <li>Commits coesos, mudanças bem justificadas e validação mínima sempre que possível.</li>
                                    <li>Alterações de arquitetura devem registrar trade-offs e implicações de manutenção.</li>
                                    <li>Novos providers precisam respeitar as abstrações existentes.</li>
                                    <li>Fluxos novos devem ser acompanhados por documentação e clareza operacional.</li>
                                </ul>
                            </div>
                        </div>
                    </Section>

                    <Section icon={ExternalLink} title="Referências e Materiais" id="references" delay={0.7}>
                        <div className="rounded-xl border border-primary/20 bg-gradient-to-br from-primary/10 to-transparent p-8">
                            <BookOpen className="mb-4 h-10 w-10 text-primary" />
                            <h3 className="mb-2 text-xl font-bold text-foreground">Pacote documental do projeto</h3>
                            <p className="mb-6 max-w-2xl text-sm text-muted-foreground">
                                Esta seção deve concentrar o TCC em PDF, o artigo derivado da pesquisa, benchmarks,
                                apresentações e referências bibliográficas usadas para sustentar o desenho do sistema.
                            </p>

                            <div className="grid gap-3 md:grid-cols-2">
                                {[
                                    "TCC completo em PDF",
                                    "Artigo técnico e versões submetidas",
                                    "Benchmarks de retrieval e avaliação",
                                    "Referências bibliográficas e materiais de apoio",
                                ].map((item) => (
                                    <div key={item} className="rounded-xl border border-border bg-background/60 p-4 text-sm text-muted-foreground">
                                        {item}
                                    </div>
                                ))}
                            </div>

                            <div className="mt-6 flex flex-wrap gap-3">
                                <Button className="gap-2 px-6 font-bold">
                                    <ExternalLink className="h-4 w-4" />
                                    Baixar TCC
                                </Button>
                                <Button variant="outline" className="gap-2 px-6">
                                    <ArrowRight className="h-4 w-4" />
                                    Abrir artigo técnico
                                </Button>
                            </div>
                        </div>
                    </Section>
                </div>
            </div>

            <motion.footer
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: 0.5 }}
                className="mt-20 flex flex-col items-center justify-between gap-4 border-t border-border pb-12 pt-8 md:flex-row"
            >
                <div className="flex items-center gap-2">
                    <div className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
                    <span className="font-mono text-[9px] uppercase tracking-[0.2em] text-muted-foreground">
                        docs status: maintained · version 0.1.0
                    </span>
                </div>
                <div className="flex gap-6 text-[10px] font-medium text-muted-foreground">
                    <a href="#" className="transition-colors hover:text-primary">GitHub</a>
                    <a href="#" className="transition-colors hover:text-primary">Architecture</a>
                    <a href="#" className="transition-colors hover:text-primary">Research</a>
                </div>
            </motion.footer>
        </AppShell>
    );
}
