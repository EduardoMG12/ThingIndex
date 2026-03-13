"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import {
    ChevronDown,
    ChevronRight,
    Database,
    FileText,
    FolderOpen,
    Hash,
    Layers3,
    Loader2,
    Table2,
    Tag,
} from "lucide-react";

import AppShell from "@/components/layout/AppShell";
import { useDocuments } from "@/hooks/use-documents";
import { fetchCollectionContext } from "@/lib/api/collections";
import type { Document } from "@/lib/api/documents";
import type { CollectionContext } from "@/lib/api/contracts/document-context";

const fadeUp = { initial: { opacity: 0, y: 16 }, animate: { opacity: 1, y: 0 } };

type DocumentCollectionGroup = {
    name: string;
    documents: Document[];
};

type DocumentCategoryGroup = {
    name: string;
    collections: DocumentCollectionGroup[];
    documentsCount: number;
};

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

function groupDocuments(documents: Document[]): DocumentCategoryGroup[] {
    const categories = new Map<string, Map<string, Document[]>>();

    for (const document of documents) {
        const categoryName = document.category?.name?.trim() || "Sem categoria";
        const collectionName = document.collection?.trim() || "Sem collection";

        if (!categories.has(categoryName)) {
            categories.set(categoryName, new Map());
        }

        const collections = categories.get(categoryName)!;
        if (!collections.has(collectionName)) {
            collections.set(collectionName, []);
        }

        collections.get(collectionName)!.push(document);
    }

    return Array.from(categories.entries())
        .map(([categoryName, collections]) => {
            const mappedCollections = Array.from(collections.entries())
                .map(([collectionName, collectionDocuments]) => ({
                    name: collectionName,
                    documents: [...collectionDocuments].sort((left, right) =>
                        new Date(right.created_at ?? 0).getTime() - new Date(left.created_at ?? 0).getTime()
                    ),
                }))
                .sort((left, right) => left.name.localeCompare(right.name, "pt-BR"));

            return {
                name: categoryName,
                collections: mappedCollections,
                documentsCount: mappedCollections.reduce((total, collection) => total + collection.documents.length, 0),
            };
        })
        .sort((left, right) => left.name.localeCompare(right.name, "pt-BR"));
}

function DocumentCard({ document }: { document: Document }) {
    const [isOpen, setIsOpen] = useState(false);
    const [context, setContext] = useState<CollectionContext | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (!isOpen || context || !document.collection_id) {
            return;
        }

        const controller = new AbortController();

        async function loadContext() {
            try {
                setIsLoading(true);
                setError(null);

                const data = await fetchCollectionContext(document.collection_id!, controller.signal);
                setContext(data);
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }

                setError(err instanceof Error ? err.message : "Nao foi possivel carregar o preview do documento.");
            } finally {
                if (!controller.signal.aborted) {
                    setIsLoading(false);
                }
            }
        }

        void loadContext();

        return () => controller.abort();
    }, [context, document.collection_id, isOpen]);

    const paragraphNodes = context?.paragraphNodes.filter((node) => node.documentId === document.id) ?? [];
    const tableSections = context?.tableSections.filter((section) => section.documentId === document.id) ?? [];
    const metadataPairs = context?.metadataPairs.filter((pair) => pair.documentId === document.id) ?? [];

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
                    onClick={() => setIsOpen((current) => !current)}
                    className="flex w-full items-center justify-between rounded-lg border border-border bg-card/40 px-3 py-2 text-left"
                >
                    <span className="flex items-center gap-2 text-xs font-semibold">
                        <Layers3 className="h-4 w-4 text-primary" />
                        Preview e vetorizacao
                    </span>
                    {isOpen ? <ChevronDown className="h-4 w-4 text-primary" /> : <ChevronRight className="h-4 w-4 text-primary" />}
                </button>

                {isOpen && (
                    <div className="mt-3 space-y-4 rounded-lg border border-border bg-card/20 p-4">
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
                            <>
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

                                <div className="space-y-2">
                                    <h4 className="text-xs font-semibold">Preview textual</h4>
                                    {paragraphNodes.length > 0 ? (
                                        <div className="space-y-2">
                                            {paragraphNodes.slice(0, 3).map((node) => (
                                                <div key={String(node.id)} className="rounded-lg border border-border bg-background/60 p-3">
                                                    <div className="mb-2 flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                                                        <Layers3 className="h-3 w-3" />
                                                        Chunk {node.sequenceOrder}
                                                        {node.pageNumber ? ` · Pagina ${node.pageNumber}` : ""}
                                                    </div>
                                                    <p className="text-xs leading-relaxed text-foreground/90">{node.content}</p>
                                                </div>
                                            ))}
                                        </div>
                                    ) : (
                                        <p className="text-xs text-muted-foreground">
                                            Ainda nao existe preview estruturado para este documento. O arquivo foi salvo, mas os chunks ainda nao foram persistidos.
                                        </p>
                                    )}
                                </div>

                                <div className="space-y-2">
                                    <h4 className="text-xs font-semibold">Como esta vetorizado</h4>
                                    {paragraphNodes.length > 0 ? (
                                        <div className="space-y-2">
                                            {paragraphNodes.map((node) => (
                                                <div key={`vector-${String(node.id)}`} className="rounded-lg border border-border bg-background/60 p-3">
                                                    <div className="flex flex-wrap gap-2 text-[10px] text-muted-foreground">
                                                        <span className="rounded bg-muted px-2 py-1 font-mono">chunk:{node.sequenceOrder}</span>
                                                        {node.pageNumber ? <span className="rounded bg-muted px-2 py-1 font-mono">page:{node.pageNumber}</span> : null}
                                                        <span className="rounded bg-muted px-2 py-1 font-mono">chars:{node.content.length}</span>
                                                    </div>
                                                    <p className="mt-2 line-clamp-3 text-xs text-foreground/90">{node.content}</p>
                                                </div>
                                            ))}
                                        </div>
                                    ) : (
                                        <p className="text-xs text-muted-foreground">
                                            O sistema ainda nao expoe os vetores numericos na API. Aqui estamos mostrando a granularidade dos chunks que alimentam a vetorizacao.
                                        </p>
                                    )}
                                </div>

                                {(tableSections.length > 0 || metadataPairs.length > 0) && (
                                    <div className="grid gap-4 lg:grid-cols-2">
                                        <div className="space-y-2">
                                            <h4 className="flex items-center gap-2 text-xs font-semibold">
                                                <Table2 className="h-4 w-4 text-primary" />
                                                Tabelas estruturadas
                                            </h4>
                                            {tableSections.length > 0 ? (
                                                tableSections.map((section) => (
                                                    <div key={String(section.id)} className="rounded-lg border border-border bg-background/60 p-3">
                                                        <p className="text-xs font-semibold">
                                                            {section.title || `Tabela ${section.sequenceOrder}`}
                                                        </p>
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
                                            <h4 className="text-xs font-semibold">Metadados extraidos</h4>
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
                            </>
                        )}
                    </div>
                )}
            </div>
        </div>
    );
}

export default function DocumentsPage() {
    const { documents, isLoading, error } = useDocuments();
    const [openCategories, setOpenCategories] = useState<Record<string, boolean>>({});
    const [openCollections, setOpenCollections] = useState<Record<string, boolean>>({});
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
                    Visualize os arquivos separados por categoria e collection, com abertura progressiva para inspecionar cada documento.
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
                            {groupedDocuments.map((category) => {
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
                                                    {category.collections.length} collection(s) · {category.documentsCount} documento(s)
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
                                                                        {collection.documents.length} documento(s) nesta collection
                                                                    </p>
                                                                </div>
                                                                <span className="rounded-full border border-border bg-muted px-3 py-1 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
                                                                    Collection
                                                                </span>
                                                            </button>

                                                            {isCollectionOpen && (
                                                                <div className="grid gap-3 border-t border-border p-4">
                                                                    {collection.documents.map((document) => (
                                                                        <DocumentCard key={document.id} document={document} />
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
                        </div>
                    )}
                </div>
            </motion.div>
        </AppShell>
    );
}
