"use client";

import { motion, Transition } from "framer-motion";
import AppShell from "@/components/layout/AppShell";
import {
    Target, Brain, Workflow, Layers, Code2,
    DollarSign, BookOpen, Lightbulb, CheckCircle2,
    XCircle, Minus, Zap, ArrowRight, ExternalLink
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Section } from "@/components/documentation/Section";
import { ComparisonTable } from "@/components/documentation/ComparisonTable";
import { PricingTable } from "@/components/documentation/PricingTable";
import { TableOfContents } from "@/components/documentation/TableOfContents";

const fadeUp = {
    initial: { opacity: 0, y: 16 },
    animate: { opacity: 1, y: 0 },
    transition: {
        duration: 0.5,
        ease: [0.22, 1, 0.36, 1]
    } as Transition
};

export default function DocumentationPage() {
    return (
        <AppShell>
            <motion.div {...fadeUp} className="mb-12">
                <p className="inline-block px-3 py-1 rounded-full bg-primary/10 text-primary font-mono text-[10px] uppercase tracking-widest mb-4">
                    documentação oficial
                </p>
                <h1 className="font-display text-4xl font-bold tracking-tight text-foreground">
                    <span className="text-primary">ThingIndex</span> — Documentação Técnica
                </h1>
                <p className="mt-4 max-w-2xl text-base text-muted-foreground leading-relaxed">
                    Bem-vindo à central técnica do ThingIndex. Aqui você encontrará detalhes sobre nossa arquitetura RAG avançada,
                    cadeias de prompts especializados e infraestrutura distribuída voltada para inteligência documental.
                </p>
            </motion.div>

            <div className="grid gap-8 lg:grid-cols-[240px_1fr]">
                {/* Sidebar TOC */}
                <motion.nav
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: 0.2, duration: 0.5 }}
                    className="hidden lg:block"
                >
                    <TableOfContents />
                </motion.nav>

                {/* Main Content Sections */}
                <div className="space-y-10">
                    <Section icon={Target} title="Sobre o Projeto" id="about" delay={0.1}>
                        <p>
                            O <strong className="text-foreground">ThingIndex</strong> é um sistema de Retrieval-Augmented Generation (RAG) de última geração,
                            desenvolvido como Trabalho de Conclusão de Curso (TCC). Ele transcende as implementações básicas ao focar em
                            <strong className="text-foreground">enriquecimento semântico</strong> e <strong className="text-foreground">extração estruturada</strong>.
                        </p>
                        <p className="mt-4">
                            Diferente de sistemas que tratam o documento como um bloco de texto plano, o ThingIndex reconhece a
                            natureza hierárquica e multidimensional dos dados, permitindo que a IA não apenas "leia", mas
                            <strong className="text-foreground">compreenda contextualmente</strong> a organização de cada arquivo.
                        </p>
                        <div className="mt-6 rounded-xl border border-primary/20 bg-primary/5 p-5">
                            <div className="flex items-center gap-2 text-primary mb-2">
                                <BookOpen className="h-4 w-4" />
                                <span className="text-xs font-bold uppercase tracking-wider">Contexto Acadêmico</span>
                            </div>
                            <p className="text-xs text-muted-foreground/90 leading-relaxed">
                                Este projeto é o ápice de uma pesquisa acadêmica focada na otimização de fluxos RAG para o
                                setor jurídico e corporativo brasileiro, priorizando precisão técnica e soberania de dados.
                            </p>
                        </div>
                    </Section>

                    <Section icon={Brain} title="RAG Tradicional vs ThingIndex" id="rag-vs-thingindex" delay={0.15}>
                        <p className="mb-6">
                            A maioria das soluções de mercado foca em "quantidade" de dados. O ThingIndex foca em
                            <strong className="text-foreground">qualidade de indexação</strong>. Abaixo comparamos
                            nossa abordagem com o padrão industrial:
                        </p>

                        <div className="grid gap-5 sm:grid-cols-2 mb-8">
                            <div className="rounded-xl border border-border bg-background/50 p-5 shadow-sm">
                                <h4 className="text-xs font-bold text-foreground mb-4 flex items-center gap-2">
                                    <XCircle className="h-4 w-4 text-destructive" />
                                    RAG Comum (Standard)
                                </h4>
                                <ul className="space-y-3 text-[11px] text-muted-foreground">
                                    <li className="flex gap-2"><Minus className="mt-0.5 h-3 w-3 shrink-0 text-muted-foreground/40" /> Chunks de tamanho fixo ignoram o contexto da página</li>
                                    <li className="flex gap-2"><Minus className="mt-0.5 h-3 w-3 shrink-0 text-muted-foreground/40" /> Metadados pobres dificultam filtros avançados</li>
                                    <li className="flex gap-2"><Minus className="mt-0.5 h-3 w-3 shrink-0 text-muted-foreground/40" /> Busca cega por similaridade gera "alucinações" de contexto</li>
                                </ul>
                            </div>
                            <div className="rounded-xl border border-primary/20 bg-primary/5 p-5 shadow-sm">
                                <h4 className="text-xs font-bold text-foreground mb-4 flex items-center gap-2">
                                    <CheckCircle2 className="h-4 w-4 text-primary" />
                                    ThingIndex (Advanced)
                                </h4>
                                <ul className="space-y-3 text-[11px] text-muted-foreground">
                                    <li className="flex gap-2"><Zap className="mt-0.5 h-3 w-3 shrink-0 text-primary" /> Chunks dinâmicos respeitam parágrafos e seções</li>
                                    <li className="flex gap-2"><Zap className="mt-0.5 h-3 w-3 shrink-0 text-primary" /> Multi-tags e grafos de entidades por nó de conhecimento</li>
                                    <li className="flex gap-2"><Zap className="mt-0.5 h-3 w-3 shrink-0 text-primary" /> Rastreamento bidirecional página-original ↔ resumo-IA</li>
                                </ul>
                            </div>
                        </div>

                        <ComparisonTable />
                    </Section>

                    <Section icon={Workflow} title="Como Funciona" id="how-it-works" delay={0.2}>
                        <p className="mb-6">O pipeline do ThingIndex é uma orquestra assíncrona feita para escalar:</p>

                        <div className="space-y-4">
                            {[
                                { step: "01", title: "Ingestão e Segurança", desc: "Upload multi-form validado por scan de malware ClamAV. Os arquivos são armazenados com segurança em MinIO (compatível com S3)." },
                                { step: "02", title: "Orquestração Assíncrona", desc: "Trabalhos de processamento são enviados para o RabbitMQ, permitindo que a API responda instantaneamente enquanto o worker processa em background." },
                                { step: "03", title: "Inteligência Documental", desc: "O ThingIndex worker fragmenta o PDF e aplica nossa cadeia de 3 prompts especializados para cada página individual." },
                                { step: "04", title: "Memória Vetorial & Relacional", desc: "Vetores de alta dimensão são salvos no Qdrant, enquanto relações estruturadas alimentam nosso grafo de conhecimento." },
                            ].map((item) => (
                                <div key={item.step} className="group relative pl-12 pb-2">
                                    <div className="absolute left-0 top-0 flex h-8 w-8 items-center justify-center rounded-lg bg-secondary border border-border font-mono text-[10px] font-bold text-muted-foreground group-hover:border-primary group-hover:text-primary transition-colors">
                                        {item.step}
                                    </div>
                                    <h4 className="text-xs font-bold text-foreground group-hover:text-primary transition-colors">{item.title}</h4>
                                    <p className="mt-1 text-[11px] text-muted-foreground/80 leading-relaxed">{item.desc}</p>
                                </div>
                            ))}
                        </div>
                    </Section>

                    <Section icon={Layers} title="Pipeline Detalhado" id="pipeline" delay={0.25}>
                        <div className="relative group">
                            <div className="absolute -inset-1 bg-gradient-to-r from-primary/20 to-transparent rounded-xl blur opacity-20 group-hover:opacity-40 transition-opacity" />
                            <div className="relative rounded-xl border border-border bg-zinc-950 p-6 font-mono text-[10px] leading-relaxed text-zinc-400 overflow-x-auto whitespace-pre custom-scrollbar">
                                {`[Web Interface] ──> (POST: /api/v1/sync)
    │
[API Gateway]
    ├─ Auth: verifySession()
    ├─ Security: clamav_scan()
    └─ Storage: storeRaw("minio://bucket/raw")
        │
[Message Queue (RabbitMQ)]
    └─ job: { doc_id, user_id, type: "PROCESS_DOCUMENT" }
        │
[ThingIndex Worker]
    ├─ Loader: loadPDF()
    ├─ Splitter: splitToPages()
    │   └─ For Each Page:
    │       ├─ P1: extractEntites(page)
    │       ├─ P2: structureSchema(resultsP1)
    │       └─ P3: generateEmbedding(resultsP2)
    │
[Vector DB (Qdrant)] ──> index_vector(1536_dims)
[Graph DB (Neo4j)]   ──> create_relationships()`}
                            </div>
                        </div>
                    </Section>

                    <Section icon={Code2} title="Cadeia de Prompts" id="prompt-chain" delay={0.3}>
                        <p className="mb-6">
                            O coração da inteligência do sistema reside na nossa <strong className="text-primary">Prompt Chain</strong>.
                            Ao invés de um único prompt genérico, usamos uma sequência lógica para garantir precisão cirúrgica:
                        </p>

                        <div className="grid gap-4 md:grid-cols-3">
                            {[
                                {
                                    title: "Enriquecimento",
                                    color: "border-blue-500/20 bg-blue-500/5",
                                    icon: Zap,
                                    desc: "Extração de entidades nomeadas e mapeamento de conceitos-chave escondidos no texto bruto."
                                },
                                {
                                    title: "Estruturação",
                                    color: "border-primary/20 bg-primary/5",
                                    icon: Layers,
                                    desc: "Conversão de dados soltos em um objeto JSON tipado seguindo o padrão industrial ThingIndex."
                                },
                                {
                                    title: "Vetorização",
                                    color: "border-emerald-500/20 bg-emerald-500/5",
                                    icon: Target,
                                    desc: "Transformação do contexto estruturado em coordenadas espaciais para busca matemática ultra-veloz."
                                },
                            ].map((step, i) => (
                                <div key={i} className={`p-4 rounded-xl border ${step.color} relative overflow-hidden group`}>
                                    <step.icon className="h-4 w-4 mb-3 opacity-50 group-hover:scale-110 transition-transform" />
                                    <h5 className="text-[11px] font-bold uppercase tracking-wider mb-2">{step.title}</h5>
                                    <p className="text-[10px] text-muted-foreground leading-relaxed">{step.desc}</p>
                                </div>
                            ))}
                        </div>
                    </Section>

                    <Section icon={DollarSign} title="Comparativo de Preços (AI)" id="pricing" delay={0.35}>
                        <p className="mb-6">
                            Manter um sistema de IA requer transparência de custos. O ThingIndex permite orquestrar
                            múltiplos modelos para balancear custo e precisão técnica:
                        </p>
                        <PricingTable />
                        <div className="mt-8 p-4 rounded-xl border border-border bg-muted/50 text-[10px] italic text-muted-foreground flex items-center gap-3">
                            <Lightbulb className="h-4 w-4 text-primary shrink-0" />
                            <span>
                                <strong>Dica:</strong> Em produção, recomendamos o uso do <strong>Gemini 1.5 Flash</strong> para a etapa 1 e
                                o <strong>Gemini 1.5 Pro</strong> para as etapas finais de estruturação, visando economia de até 70% sem perda de qualidade.
                            </span>
                        </div>
                    </Section>

                    <Section icon={ExternalLink} title="Referências & Artigo" id="references" delay={0.4}>
                        <div className="rounded-xl bg-gradient-to-br from-primary/10 to-transparent border border-primary/20 p-8 text-center">
                            <BookOpen className="mx-auto h-10 w-10 text-primary mb-4" />
                            <h3 className="text-xl font-bold mb-2">Artigo Científico Completo</h3>
                            <p className="text-sm text-muted-foreground mb-6 max-w-md mx-auto">
                                Leia a fundamentação teórica completa por trás do ThingIndex, incluindo análise de HNSW,
                                embeddings multidimensionais e o futuro da inteligência documental no Brasil.
                            </p>
                            <Button className="gap-2 px-8 font-bold">
                                <ExternalLink className="h-4 w-4" />
                                Baixar TCC (PDF)
                            </Button>
                        </div>
                    </Section>
                </div>
            </div>

            <motion.footer
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: 0.5 }}
                className="mt-20 border-t border-border pt-8 pb-12 flex flex-col md:flex-row justify-between items-center gap-4"
            >
                <div className="flex items-center gap-2">
                    <div className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
                    <span className="font-mono text-[9px] uppercase tracking-[0.2em] text-muted-foreground">docs status: running · version 0.1.0</span>
                </div>
                <div className="flex gap-6 text-[10px] font-medium text-muted-foreground">
                    <a href="#" className="hover:text-primary transition-colors">Github</a>
                    <a href="#" className="hover:text-primary transition-colors">API Reference</a>
                    <a href="#" className="hover:text-primary transition-colors">Privacy</a>
                </div>
            </motion.footer>
        </AppShell>
    );
}
