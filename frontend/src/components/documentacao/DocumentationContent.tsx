"use client";

import { motion } from "framer-motion";
import {
    FileText,
    Database,
    Layers,
    Code2,
    BookOpen,
    Target,
    Lightbulb,
    Workflow,
} from "lucide-react";

const fadeUp = {
    initial: { opacity: 0, y: 20 },
    animate: { opacity: 1, y: 0 },
};

// Dados separados para fácil manutenção (podem vir de uma API no futuro)
export const sections = [
    {
        icon: Target,
        title: "Objetivo do Projeto",
        content:
            "Este projeto de TCC tem como objetivo desenvolver um sistema baseado em grafos de conhecimento (Knowledge Graph) para análise e recuperação de informações de documentos legais brasileiros. Utilizando técnicas de Retrieval Augmented Generation (RAG).",
    },
    {
        icon: Workflow,
        title: "Arquitetura do Sistema",
        content:
            "O sistema segue uma arquitetura em camadas:\n\n• **Frontend** — Interface interativa (React)\n• **API Gateway** — Edge functions\n• **Crawling Engine** — Motor de crawling\n• **Graph Database** — Banco de grafos\n• **LLM Integration** — Integração com modelos para RAG\n\n*(Conteúdo detalhado a ser adicionado...)*",
    },
    {
        icon: Database,
        title: "Banco de Grafos",
        content:
            "O banco armazena as relações entre Documentos, Seções, Artigos e Conceitos jurídicos extraídos. As conexões representam referências cruzadas e hierarquias.\n\n*(Conteúdo detalhado a ser adicionado...)*",
    },
    {
        icon: Layers,
        title: "Pipeline de Processamento",
        content:
            "As etapas incluem:\n1. Crawling\n2. Parsing Estrutural\n3. Extração de Entidades (NER)\n4. Construção do Grafo\n5. Indexação Vetorial\n6. RAG (Retrieval Augmented Generation)",
    },
    {
        icon: Code2,
        title: "Stack Tecnológico",
        content:
            "• Frontend: React, Tailwind CSS\n• Backend: Edge Functions\n• Banco: PostgreSQL + pgvector\n• Extratores: Firecrawl API",
    },
    {
        icon: BookOpen,
        title: "Referências Bibliográficas",
        content:
            "• Lewis, P. et al. (2020). Retrieval-Augmented Generation...\n• Ji, S. et al. (2021). A Survey on Knowledge Graphs...\n\n*(Mais referências em breve...)*",
    },
    {
        icon: Lightbulb,
        title: "Resultados Esperados",
        content:
            "Construir grafos representativos da estrutura hierárquica e fornecer respostas jurídicas embasadas e com referências diretas.",
    },
];

export default function DocumentationContent() {
    return (
        <main className="container max-w-3xl py-8">
            <motion.div {...fadeUp} transition={{ duration: 0.5 }}>
                <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/15">
                        <FileText className="h-5 w-5 text-primary" />
                    </div>
                    <div>
                        <h1 className="text-3xl font-bold">
                            <span className="text-gradient-gold">Documentação</span>
                        </h1>
                        <p className="text-sm text-muted-foreground">
                            Estrutura do projeto, arquitetura e referências
                        </p>
                    </div>
                </div>
            </motion.div>

            <div className="mt-10 space-y-8">
                {sections.map((section, i) => {
                    const Icon = section.icon;
                    return (
                        <motion.article
                            key={section.title}
                            {...fadeUp}
                            transition={{ delay: 0.1 + i * 0.08, duration: 0.5 }}
                            className="surface-card rounded-lg p-6"
                        >
                            <div className="mb-3 flex items-center gap-2.5">
                                <Icon className="h-4.5 w-4.5 text-primary" />
                                <h2 className="text-base font-bold">{section.title}</h2>
                            </div>
                            <div className="whitespace-pre-line text-sm leading-relaxed text-secondary-foreground">
                                {section.content.split(/(\*\*.*?\*\*)/).map((part, j) => {
                                    if (part.startsWith("**") && part.endsWith("**")) {
                                        return (
                                            <strong key={j} className="font-semibold text-foreground">
                                                {part.slice(2, -2)}
                                            </strong>
                                        );
                                    }
                                    return part;
                                })}
                            </div>
                        </motion.article>
                    );
                })}
            </div>
        </main>
    );
}
