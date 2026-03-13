"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import {
    ChevronDown,
    ChevronLeft,
    ChevronRight,
    Database,
    ExternalLink,
    FileText,
    FolderOpen,
    Hash,
    Loader2,
    PanelRight,
    Table2,
    Tag,
    X,
} from "lucide-react";

import AppShell from "@/components/layout/AppShell";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDocuments } from "@/hooks/use-documents";
import { fetchCollectionContext } from "@/lib/api/collections";
import type { CollectionContext } from "@/lib/api/contracts/document-context";
import { fetchDocumentIndex, type Document, type DocumentIndex } from "@/lib/api/documents";

const fadeUp = { initial: { opacity: 0, y: 16 }, animate: { opacity: 1, y: 0 } };

function formatSize(bytes: number) {
    if (bytes < 1024) {
        return `${bytes} B`;
    }
    if (bytes < 1024 * 1024) {
        return `${(bytes / 1024).toFixed(1)} KB`;
    }
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDate(value?: string) {
    if (!value) {
        return "-";
    }

    return new Intl.DateTimeFormat("pt-BR", {
        dateStyle: "short",
        timeStyle: "short",
    }).format(new Date(value));
}

function isPreviewableDocument(document: Document) {
    return ["application/pdf", "text/html", "text/plain"].includes(document.mime_type);
}

function buildPreviewURL(document: Document) {
    return `/api/documents/${document.id}/preview`;
}

function groupDocuments(documents: Document[]) {
    const categories = new Map<string, Map<string, Document[]>>();
    const uncategorizedCollections = new Map<string, Document[]>();
    const uncategorizedLoose: Document[] = [];

    for (const document of documents) {
        const categoryName = document.category?.name?.trim();
        const collectionName = document.collection?.trim();

        if (!categoryName) {
            if (collectionName) {
                if (!uncategorizedCollections.has(collectionName)) {
                    uncategorizedCollections.set(collectionName, []);
                }

                uncategorizedCollections.get(collectionName)!.push(document);
            } else {
                uncategorizedLoose.push(document);
            }

            continue;
        }

        if (!categories.has(categoryName)) {
            categories.set(categoryName, new Map());
        }

        const collections = categories.get(categoryName)!;
        const normalizedCollection = collectionName || "Sem collection";

        if (!collections.has(normalizedCollection)) {
            collections.set(normalizedCollection, []);
        }

        collections.get(normalizedCollection)!.push(document);
    }

    const categorized = Array.from(categories.entries())
        .map(([categoryName, collections]) => ({
            name: categoryName,
            collections: Array.from(collections.entries())
                .map(([collectionName, collectionDocuments]) => ({
                    name: collectionName,
                    documents: [...collectionDocuments].sort((left, right) =>
                        new Date(right.created_at ?? 0).getTime() - new Date(left.created_at ?? 0).getTime()
                    ),
                }))
                .sort((left, right) => left.name.localeCompare(right.name, "pt-BR")),
        }))
        .sort((left, right) => left.name.localeCompare(right.name, "pt-BR"));

    const uncategorizedByCollection = Array.from(uncategorizedCollections.entries())
        .map(([collectionName, collectionDocuments]) => ({
            name: collectionName,
            documents: [...collectionDocuments].sort((left, right) =>
                new Date(right.created_at ?? 0).getTime() - new Date(left.created_at ?? 0).getTime()
            ),
        }))
        .sort((left, right) => left.name.localeCompare(right.name, "pt-BR"));

    const uncategorizedWithoutCollection = [...uncategorizedLoose].sort((left, right) =>
        new Date(right.created_at ?? 0).getTime() - new Date(left.created_at ?? 0).getTime()
    );

    return {
        categorized,
        uncategorizedByCollection,
        uncategorizedWithoutCollection,
    };
}

function FilePreview({ document }: { document: Document }) {
    if (!isPreviewableDocument(document)) {
        return (
            <div className="rounded-xl border border-border bg-background/60 p-5 text-sm text-muted-foreground">
                Preview visual indisponivel para este tipo de arquivo. Tipos suportados: PDF, HTML e texto plano.
            </div>
        );
    }

    return (
        <iframe
            title={`Preview de ${document.title}`}
            src={buildPreviewURL(document)}
            className="h-[70vh] w-full rounded-xl border border-border bg-white"
            sandbox="allow-same-origin"
        />
    );
}

function VectorizationView({
    document,
    context,
    documentIndex,
}: {
    document: Document;
    context: CollectionContext | null;
    documentIndex: DocumentIndex | null;
}) {
    const paragraphNodes = context?.paragraphNodes.filter((node) => node.documentId === document.id) ?? [];
    const tableSections = context?.tableSections.filter((section) => section.documentId === document.id) ?? [];
    const metadataPairs = context?.metadataPairs.filter((pair) => pair.documentId === document.id) ?? [];
    const outlineNodes = documentIndex?.outline_nodes ?? context?.outlineNodes.filter((node) => node.documentId === document.id) ?? [];
    const documentSummary =
        documentIndex?.document_summary ||
        metadataPairs.find((pair) => pair.key === "document_summary")?.value ||
        document.document_summary ||
        "";

    return (
        <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-3">
                <div className="rounded-lg border border-border bg-background/60 p-3">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">Chunks</p>
                    <p className="mt-2 text-lg font-semibold">{paragraphNodes.length}</p>
                </div>
                <div className="rounded-lg border border-border bg-background/60 p-3">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">Tabelas</p>
                    <p className="mt-2 text-lg font-semibold">{tableSections.length}</p>
                </div>
                <div className="rounded-lg border border-border bg-background/60 p-3">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">Metadados</p>
                    <p className="mt-2 text-lg font-semibold">{metadataPairs.length}</p>
                </div>
            </div>

            <div className="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">Resumo indexado</p>
                    <p className="mt-3 text-sm leading-relaxed text-foreground/90">
                        {documentSummary || "Resumo ainda indisponivel. Quando o worker concluir o enriquecimento, ele aparecera aqui."}
                    </p>
                </div>

                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">Outline</p>
                    {outlineNodes.length > 0 ? (
                        <div className="mt-3 space-y-2">
                            {outlineNodes.map((node) => (
                                <div key={String(node.id)} className="rounded-lg border border-border bg-card/70 px-3 py-2">
                                    <p className="text-[10px] font-mono uppercase tracking-wider text-muted-foreground">
                                        nivel {node.level} · secao {node.sequenceOrder}
                                    </p>
                                    <p className="mt-1 text-sm font-medium">{node.title}</p>
                                    {node.summary ? (
                                        <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{node.summary}</p>
                                    ) : null}
                                </div>
                            ))}
                        </div>
                    ) : (
                        <p className="mt-3 text-xs text-muted-foreground">
                            O outline estrutural ainda nao foi gerado para este documento.
                        </p>
                    )}
                </div>
            </div>

            <div className="space-y-2">
                <h3 className="text-sm font-semibold">Preview da vetorizacao</h3>
                {paragraphNodes.length > 0 ? (
                    <div className="space-y-2">
                        {paragraphNodes.map((node) => (
                            <div key={String(node.id)} className="rounded-lg border border-border bg-background/60 p-3">
                                <div className="flex flex-wrap gap-2 text-[10px] text-muted-foreground">
                                    <span className="rounded bg-muted px-2 py-1 font-mono">chunk:{node.sequenceOrder}</span>
                                    {node.pageNumber ? <span className="rounded bg-muted px-2 py-1 font-mono">page:{node.pageNumber}</span> : null}
                                    <span className="rounded bg-muted px-2 py-1 font-mono">chars:{node.content.length}</span>
                                </div>
                                <p className="mt-2 text-xs leading-relaxed text-foreground/90">{node.content}</p>
                            </div>
                        ))}
                    </div>
                ) : (
                    <p className="text-sm text-muted-foreground">
                        Ainda nao ha estrutura vetorizada persistida para este documento. Quando o processamento estrutural ocorrer, os chunks aparecerao aqui.
                    </p>
                )}
            </div>

            {(tableSections.length > 0 || metadataPairs.length > 0) && (
                <div className="grid gap-4 lg:grid-cols-2">
                    <div className="space-y-2">
                        <h4 className="flex items-center gap-2 text-sm font-semibold">
                            <Table2 className="h-4 w-4 text-primary" />
                            Tabelas estruturadas
                        </h4>
                        {tableSections.length > 0 ? (
                            tableSections.map((section) => (
                                <div key={String(section.id)} className="rounded-lg border border-border bg-background/60 p-3">
                                    <p className="text-xs font-semibold">{section.title || `Tabela ${section.sequenceOrder}`}</p>
                                    <p className="mt-1 text-[11px] text-muted-foreground">
                                        {section.headers.length} coluna(s) · {section.rows.length} linha(s)
                                    </p>
                                </div>
                            ))
                        ) : (
                            <p className="text-xs text-muted-foreground">Nenhuma tabela estruturada para este documento.</p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <h4 className="text-sm font-semibold">Metadados extraidos</h4>
                        {metadataPairs.length > 0 ? (
                            metadataPairs.map((pair) => (
                                <div key={String(pair.id)} className="rounded-lg border border-border bg-background/60 p-3">
                                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{pair.key}</p>
                                    <p className="mt-1 text-xs text-foreground/90">{pair.value}</p>
                                </div>
                            ))
                        ) : (
                            <p className="text-xs text-muted-foreground">Nenhum metadado estruturado para este documento.</p>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

function DocumentPreviewPanel({
    document,
    onClose,
}: {
    document: Document | null;
    onClose: () => void;
}) {
    const [context, setContext] = useState<CollectionContext | null>(null);
    const [documentIndex, setDocumentIndex] = useState<DocumentIndex | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (!document) {
            setContext(null);
            setDocumentIndex(null);
            setIsLoading(false);
            setError(null);
            return;
        }
        const currentDocument = document;

        const controller = new AbortController();

        async function loadContext() {
            try {
                setIsLoading(true);
                setError(null);
                const [indexData, contextData] = await Promise.all([
                    fetchDocumentIndex(currentDocument.id, controller.signal),
                    currentDocument.collection_id
                        ? fetchCollectionContext(currentDocument.collection_id, controller.signal)
                        : Promise.resolve(null),
                ]);
                setDocumentIndex(indexData);
                setContext(contextData);
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }

                setError(err instanceof Error ? err.message : "Nao foi possivel carregar a vetorizacao do documento.");
            } finally {
                if (!controller.signal.aborted) {
                    setIsLoading(false);
                }
            }
        }

        void loadContext();

        return () => controller.abort();
    }, [document]);

    if (!document) {
        return null;
    }

    return (
        <div className="fixed inset-0 z-50 flex">
            <button
                type="button"
                onClick={onClose}
                className="flex-1 bg-black/55 backdrop-blur-sm"
                aria-label="Fechar painel"
            />
            <aside className="relative h-full w-full max-w-[980px] shrink-0 border-l border-border bg-card shadow-2xl">
                <div className="flex h-full flex-col">
                    <div className="flex items-start justify-between gap-4 border-b border-border px-6 py-5">
                        <div className="min-w-0">
                            <div className="mb-2 flex items-center gap-2">
                                <PanelRight className="h-4 w-4 text-primary" />
                                <span className="text-[10px] font-bold uppercase tracking-[0.3em] text-primary">Preview Panel</span>
                            </div>
                            <h2 className="truncate text-lg font-semibold">{document.title}</h2>
                            <p className="mt-2 break-all font-mono text-[11px] text-muted-foreground">
                                {document.file_path || document.file_url}
                            </p>
                        </div>
                        <button
                            type="button"
                            onClick={onClose}
                            className="rounded-lg border border-border p-2 text-muted-foreground transition-colors hover:text-foreground"
                        >
                            <X className="h-4 w-4" />
                        </button>
                    </div>

                    <div className="border-b border-border px-6 py-3">
                        <div className="flex flex-wrap gap-2">
                            <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                <Hash className="h-3 w-3" />
                                {formatSize(document.size)}
                            </span>
                            <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                <FileText className="h-3 w-3" />
                                {document.mime_type}
                            </span>
                            <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                <FolderOpen className="h-3 w-3" />
                                {document.collection || "Sem collection"}
                            </span>
                            <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                <Tag className="h-3 w-3" />
                                {document.category?.name || "Sem categoria"}
                            </span>
                        </div>
                    </div>

                    <div className="flex-1 overflow-y-auto p-6">
                        <Tabs defaultValue="document" className="w-full">
                            <TabsList className="mb-6 h-auto justify-start gap-3 bg-transparent p-0">
                                <TabsTrigger
                                    value="document"
                                    className="border border-transparent px-4 py-2 text-xs font-bold uppercase tracking-widest data-[state=active]:border-primary/20 data-[state=active]:bg-primary/10 data-[state=active]:text-primary"
                                >
                                    Documento
                                </TabsTrigger>
                                <TabsTrigger
                                    value="vectorization"
                                    className="border border-transparent px-4 py-2 text-xs font-bold uppercase tracking-widest data-[state=active]:border-primary/20 data-[state=active]:bg-primary/10 data-[state=active]:text-primary"
                                >
                                    Vetorizacao
                                </TabsTrigger>
                            </TabsList>

                            <TabsContent value="document" className="mt-0 space-y-4">
                                <div className="flex items-center justify-between">
                                    <h3 className="text-sm font-semibold">Preview do documento</h3>
                                    <a
                                        href={buildPreviewURL(document)}
                                        target="_blank"
                                        rel="noreferrer"
                                        className="inline-flex items-center gap-1 text-[11px] font-medium text-primary"
                                    >
                                        Abrir arquivo
                                        <ExternalLink className="h-3.5 w-3.5" />
                                    </a>
                                </div>
                                <FilePreview document={document} />
                                {!isPreviewableDocument(document) && (
                                    <p className="text-xs text-muted-foreground">
                                        `docx`, `xlsx` e formatos similares nao possuem renderizacao nativa no navegador sem conversao adicional.
                                    </p>
                                )}
                            </TabsContent>

                            <TabsContent value="vectorization" className="mt-0 space-y-4">
                                {isLoading && (
                                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                        <Loader2 className="h-4 w-4 animate-spin" />
                                        Carregando estrutura do documento...
                                    </div>
                                )}

                                {!isLoading && error && (
                                    <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive">
                                        {error}
                                    </div>
                                )}

                                {!isLoading && !error && (
                                    <VectorizationView document={document} context={context} documentIndex={documentIndex} />
                                )}
                            </TabsContent>
                        </Tabs>
                    </div>
                </div>
            </aside>
        </div>
    );
}

function DocumentCard({
    document,
    onOpenPreview,
}: {
    document: Document;
    onOpenPreview: (document: Document) => void;
}) {
    return (
        <div className="rounded-xl border border-border bg-background/70 p-4 shadow-card">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="min-w-0">
                    <div className="flex items-center gap-2">
                        <FileText className="h-4 w-4 text-primary" />
                        <h3 className="truncate text-sm font-semibold">{document.title}</h3>
                    </div>
                    <p className="mt-2 break-all font-mono text-[11px] text-muted-foreground">
                        {document.file_path || document.file_url}
                    </p>
                </div>

                <div className="rounded-full border border-primary/20 bg-primary/10 px-3 py-1 text-[10px] font-bold uppercase tracking-widest text-primary">
                    {document.status}
                </div>
            </div>

            <div className="mt-4 flex flex-wrap gap-2">
                <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                    <Hash className="h-3 w-3" />
                    {formatSize(document.size)}
                </span>
                <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                    <FileText className="h-3 w-3" />
                    {document.mime_type}
                </span>
                <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                    <FolderOpen className="h-3 w-3" />
                    {document.collection || "Sem collection"}
                </span>
                <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                    <Tag className="h-3 w-3" />
                    {document.category?.name || "Sem categoria"}
                </span>
            </div>

            <div className="mt-4 grid gap-2 border-t border-border pt-4 text-[11px] text-muted-foreground md:grid-cols-2">
                <p>Criado em: <span className="font-mono text-foreground">{formatDate(document.created_at)}</span></p>
                <p>Atualizado em: <span className="font-mono text-foreground">{formatDate(document.updated_at)}</span></p>
            </div>

            <div className="mt-4 border-t border-border pt-4">
                <button
                    type="button"
                    onClick={() => onOpenPreview(document)}
                    className="inline-flex items-center gap-2 rounded-lg border border-primary/20 bg-primary/10 px-3 py-2 text-xs font-semibold text-primary"
                >
                    <PanelRight className="h-4 w-4" />
                    Abrir preview lateral
                    <ChevronLeft className="h-4 w-4" />
                </button>
            </div>
        </div>
    );
}

export default function DocumentsPage() {
    const { documents, isLoading, error } = useDocuments();
    const [openCategories, setOpenCategories] = useState<Record<string, boolean>>({});
    const [openCollections, setOpenCollections] = useState<Record<string, boolean>>({});
    const [selectedDocument, setSelectedDocument] = useState<Document | null>(null);
    const groupedDocuments = groupDocuments(documents);

    const toggleCategory = (categoryName: string) => {
        setOpenCategories((current) => ({
            ...current,
            [categoryName]: !current[categoryName],
        }));
    };

    const toggleCollection = (key: string) => {
        setOpenCollections((current) => ({
            ...current,
            [key]: !current[key],
        }));
    };

    return (
        <AppShell>
            <motion.div {...fadeUp} transition={{ duration: 0.5 }} className="mb-8">
                <div className="mb-3 flex items-center gap-2">
                    <span className="h-px w-8 bg-primary" />
                    <span className="font-bold uppercase tracking-[0.3em] text-[10px] text-primary">Object Storage Index</span>
                </div>
                <h1 className="font-display text-4xl font-bold tracking-tight text-foreground">
                    Document<span className="text-primary italic">os</span>
                </h1>
                <p className="mt-4 max-w-3xl text-base leading-relaxed text-muted-foreground">
                    Visualize os arquivos separados por categoria e collection. Cada documento pode ser aberto em um painel lateral grande com preview e vetorizacao.
                </p>
            </motion.div>

            <motion.div
                {...fadeUp}
                transition={{ delay: 0.1, duration: 0.4 }}
                className="surface-elevated rounded-xl border border-border"
            >
                <div className="flex items-center justify-between border-b border-border px-6 py-4">
                    <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
                            <Database className="h-5 w-5" />
                        </div>
                        <div>
                            <h2 className="text-sm font-bold tracking-tight">Biblioteca de documentos</h2>
                            <p className="text-xs text-muted-foreground">{documents.length} documento(s) indexado(s)</p>
                        </div>
                    </div>
                </div>

                <div className="p-6">
                    {isLoading && (
                        <div className="flex min-h-64 items-center justify-center gap-3 text-sm text-muted-foreground">
                            <Loader2 className="h-4 w-4 animate-spin" />
                            Carregando documentos...
                        </div>
                    )}

                    {!isLoading && error && (
                        <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                            {error}
                        </div>
                    )}

                    {!isLoading && !error && documents.length === 0 && (
                        <div className="flex min-h-64 flex-col items-center justify-center text-center">
                            <FileText className="mb-4 h-10 w-10 text-muted-foreground" />
                            <h3 className="text-sm font-semibold">Nenhum documento enviado ainda</h3>
                            <p className="mt-2 max-w-md text-xs leading-relaxed text-muted-foreground">
                                Envie arquivos pelo dashboard para que eles aparecam aqui e possam ser auditados por collection, categoria e status.
                            </p>
                        </div>
                    )}

                    {!isLoading && !error && documents.length > 0 && (
                        <div className="space-y-4">
                            {groupedDocuments.categorized.map((category) => {
                                const isCategoryOpen = openCategories[category.name] ?? true;

                                return (
                                    <div key={category.name} className="rounded-xl border border-border bg-background/40">
                                        <button
                                            type="button"
                                            onClick={() => toggleCategory(category.name)}
                                            className="flex w-full items-center justify-between gap-3 px-4 py-4 text-left"
                                        >
                                            <div className="min-w-0">
                                                <div className="flex items-center gap-2">
                                                    {isCategoryOpen ? <ChevronDown className="h-4 w-4 text-primary" /> : <ChevronRight className="h-4 w-4 text-primary" />}
                                                    <span className="text-sm font-semibold">{category.name}</span>
                                                </div>
                                                <p className="mt-1 text-[11px] text-muted-foreground">
                                                    {category.collections.length} collection(s)
                                                </p>
                                            </div>
                                            <span className="rounded-full border border-primary/20 bg-primary/10 px-3 py-1 text-[10px] font-bold uppercase tracking-widest text-primary">
                                                Categoria
                                            </span>
                                        </button>

                                        {isCategoryOpen && (
                                            <div className="space-y-3 border-t border-border p-4">
                                                {category.collections.map((collection) => {
                                                    const collectionKey = `${category.name}::${collection.name}`;
                                                    const isCollectionOpen = openCollections[collectionKey] ?? true;

                                                    return (
                                                        <div key={collectionKey} className="rounded-xl border border-border bg-card/40">
                                                            <button
                                                                type="button"
                                                                onClick={() => toggleCollection(collectionKey)}
                                                                className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left"
                                                            >
                                                                <div className="min-w-0">
                                                                    <div className="flex items-center gap-2">
                                                                        {isCollectionOpen ? <ChevronDown className="h-4 w-4 text-foreground" /> : <ChevronRight className="h-4 w-4 text-foreground" />}
                                                                        <FolderOpen className="h-4 w-4 text-primary" />
                                                                        <span className="truncate text-sm font-medium">{collection.name}</span>
                                                                    </div>
                                                                    <p className="mt-1 text-[11px] text-muted-foreground">
                                                                        {collection.documents.length} documento(s)
                                                                    </p>
                                                                </div>
                                                            </button>

                                                            {isCollectionOpen && (
                                                                <div className="grid gap-3 border-t border-border p-4">
                                                                    {collection.documents.map((document) => (
                                                                        <DocumentCard key={document.id} document={document} onOpenPreview={setSelectedDocument} />
                                                                    ))}
                                                                </div>
                                                            )}
                                                        </div>
                                                    );
                                                })}
                                            </div>
                                        )}
                                    </div>
                                );
                            })}

                            {(groupedDocuments.uncategorizedByCollection.length > 0 ||
                                groupedDocuments.uncategorizedWithoutCollection.length > 0) && (
                                <div className="rounded-xl border border-dashed border-border bg-background/30 p-4">
                                    <div className="mb-4">
                                        <h3 className="text-sm font-semibold">Documentos sem categoria</h3>
                                        <p className="mt-1 text-[11px] text-muted-foreground">
                                            Estes documentos ficam fora da arvore principal por ainda nao estarem vinculados a uma categoria.
                                        </p>
                                    </div>

                                    <div className="space-y-4">
                                        {groupedDocuments.uncategorizedByCollection.map((collection) => (
                                            <div key={`uncategorized-${collection.name}`} className="rounded-xl border border-border bg-card/40">
                                                {(() => {
                                                    const collectionKey = `uncategorized::${collection.name}`;
                                                    const isCollectionOpen = openCollections[collectionKey] ?? true;

                                                    return (
                                                        <>
                                                            <button
                                                                type="button"
                                                                onClick={() => toggleCollection(collectionKey)}
                                                                className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left"
                                                            >
                                                                <div className="min-w-0">
                                                                    <div className="flex items-center gap-2">
                                                                        {isCollectionOpen ? <ChevronDown className="h-4 w-4 text-foreground" /> : <ChevronRight className="h-4 w-4 text-foreground" />}
                                                                        <FolderOpen className="h-4 w-4 text-primary" />
                                                                        <span className="truncate text-sm font-medium">{collection.name}</span>
                                                                    </div>
                                                                    <p className="mt-1 text-[11px] text-muted-foreground">
                                                                        {collection.documents.length} documento(s) sem categoria
                                                                    </p>
                                                                </div>
                                                            </button>

                                                            {isCollectionOpen && (
                                                                <div className="grid gap-3 border-t border-border p-4">
                                                                    {collection.documents.map((document) => (
                                                                        <DocumentCard key={document.id} document={document} onOpenPreview={setSelectedDocument} />
                                                                    ))}
                                                                </div>
                                                            )}
                                                        </>
                                                    );
                                                })()}
                                            </div>
                                        ))}

                                        {groupedDocuments.uncategorizedWithoutCollection.length > 0 && (
                                            <div className="grid gap-3">
                                                {groupedDocuments.uncategorizedWithoutCollection.map((document) => (
                                                    <DocumentCard key={document.id} document={document} onOpenPreview={setSelectedDocument} />
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </motion.div>

            <DocumentPreviewPanel document={selectedDocument} onClose={() => setSelectedDocument(null)} />
        </AppShell>
    );
}
