"use client";

import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { Clock3, FileSearch, Filter, Loader2, RefreshCw, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import AuditMetadataList from "@/components/audit/AuditMetadataList";
import AuditStatusBadge from "@/components/audit/AuditStatusBadge";
import {
    fetchDocumentAuditEvents,
    type DocumentAuditEvent,
    type DocumentAuditQueryResult,
} from "@/lib/api/audit";
import {
    formatAuditActor,
    formatAuditDateTime,
    formatDocumentAuditAction,
    getAuditBucketCount,
    toRFC3339FromDateTimeLocal,
} from "@/lib/audit";
import { cn } from "@/lib/utils";

type Scope = "ALL" | "DOCUMENT" | "COLLECTION";

type DocumentAuditExplorerProps = {
    locale: string;
    reloadToken?: number;
    selectedDocument?: {
        id: number;
        title: string;
        collection_id?: number | null;
        collection?: string;
    } | null;
};

type AuditFilters = {
    scope: Scope;
    action: string;
    status: string;
    query: string;
    correlationId: string;
    createdFrom: string;
    createdTo: string;
};

const initialFilters: AuditFilters = {
    scope: "ALL",
    action: "ALL",
    status: "ALL",
    query: "",
    correlationId: "",
    createdFrom: "",
    createdTo: "",
};

const emptyResult: DocumentAuditQueryResult = {
    items: [],
    summary: {
        total_events: 0,
        unique_documents: 0,
        unique_collections: 0,
        unique_requests: 0,
        action_breakdown: [],
        status_breakdown: [],
    },
    page_info: {
        limit: 0,
        returned: 0,
        has_next: false,
    },
};

function SummaryCard({
    label,
    value,
    detail,
}: {
    label: string;
    value: string;
    detail: string;
}) {
    return (
        <div className="rounded-xl border border-border bg-background/60 p-4">
            <p className="text-[10px] uppercase tracking-[0.24em] text-muted-foreground">{label}</p>
            <p className="mt-3 text-xl font-semibold">{value}</p>
            <p className="mt-1 text-xs text-muted-foreground">{detail}</p>
        </div>
    );
}

function ScopeButton({
    active,
    children,
    onClick,
}: {
    active: boolean;
    children: ReactNode;
    onClick: () => void;
}) {
    return (
        <button
            type="button"
            onClick={onClick}
            className={cn(
                "rounded-full border px-3 py-1 text-[11px] font-semibold transition-colors",
                active
                    ? "border-primary/20 bg-primary/10 text-primary"
                    : "border-border bg-background text-muted-foreground"
            )}
        >
            {children}
        </button>
    );
}

function DocumentAuditEventCard({
    event,
    locale,
}: {
    event: DocumentAuditEvent;
    locale: string;
}) {
    return (
        <div className="rounded-xl border border-border bg-background/70 p-4">
            <div className="flex flex-wrap items-center gap-2">
                <p className="text-sm font-semibold">{event.title || `Documento #${event.document_id ?? event.id}`}</p>
                <span className="rounded-full border border-border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {formatDocumentAuditAction(event.action)}
                </span>
                <AuditStatusBadge status={event.status} />
            </div>

            <p className="mt-2 text-sm text-foreground/90">{event.detail || "Sem detalhe registrado."}</p>

            <AuditMetadataList metadata={event.metadata} />

            <div className="mt-3 grid gap-2 text-[11px] text-muted-foreground md:grid-cols-2 xl:grid-cols-3">
                <p>Quando: <span className="font-mono text-foreground">{formatAuditDateTime(event.created_at, locale)}</span></p>
                <p>Origem: <span className="font-mono text-foreground">{event.origin || "-"}</span></p>
                <p>Request: <span className="font-mono text-foreground">{event.correlation_id || "-"}</span></p>
                <p>Actor: <span className="font-mono text-foreground">{formatAuditActor(event.actor_type, event.actor_id)}</span></p>
                <p>Documento: <span className="font-mono text-foreground">{event.document_id ?? "-"}</span></p>
                <p>Collection: <span className="font-mono text-foreground">{event.collection_id ?? "-"}</span></p>
            </div>
        </div>
    );
}

export default function DocumentAuditExplorer({
    locale,
    reloadToken = 0,
    selectedDocument,
}: DocumentAuditExplorerProps) {
    const [filters, setFilters] = useState<AuditFilters>(initialFilters);
    const [appliedFilters, setAppliedFilters] = useState<AuditFilters>(initialFilters);
    const [result, setResult] = useState<DocumentAuditQueryResult>(emptyResult);
    const [error, setError] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [isLoadingMore, setIsLoadingMore] = useState(false);

    useEffect(() => {
        if (filters.scope === "DOCUMENT" && !selectedDocument) {
            setFilters((current) => ({ ...current, scope: "ALL" }));
        }
        if (filters.scope === "COLLECTION" && !selectedDocument?.collection_id) {
            setFilters((current) => ({ ...current, scope: "ALL" }));
        }
    }, [filters.scope, selectedDocument, selectedDocument?.collection_id]);

    useEffect(() => {
        let cancelled = false;

        const resolvedScope =
            appliedFilters.scope === "DOCUMENT" && selectedDocument?.id
                ? "DOCUMENT"
                : appliedFilters.scope === "COLLECTION" && selectedDocument?.collection_id
                  ? "COLLECTION"
                  : "ALL";

        async function load() {
            setIsLoading(true);
            setError(null);

            try {
                const nextResult = await fetchDocumentAuditEvents({
                    limit: 12,
                    query: appliedFilters.query || undefined,
                    action: appliedFilters.action === "ALL" ? undefined : appliedFilters.action,
                    status: appliedFilters.status === "ALL" ? undefined : appliedFilters.status,
                    correlationId: appliedFilters.correlationId || undefined,
                    documentId: resolvedScope === "DOCUMENT" ? selectedDocument?.id : undefined,
                    collectionId: resolvedScope === "COLLECTION" ? selectedDocument?.collection_id ?? undefined : undefined,
                    createdFrom: toRFC3339FromDateTimeLocal(appliedFilters.createdFrom),
                    createdTo: toRFC3339FromDateTimeLocal(appliedFilters.createdTo),
                });

                if (!cancelled) {
                    setResult(nextResult);
                }
            } catch (nextError) {
                if (!cancelled) {
                    setError(nextError instanceof Error ? nextError.message : "Nao foi possivel carregar a auditoria documental.");
                    setResult(emptyResult);
                }
            } finally {
                if (!cancelled) {
                    setIsLoading(false);
                }
            }
        }

        void load();

        return () => {
            cancelled = true;
        };
    }, [appliedFilters, reloadToken, selectedDocument?.collection_id, selectedDocument?.id]);

    async function handleLoadMore() {
        if (!result.page_info.has_next || !result.page_info.next_cursor) {
            return;
        }

        const resolvedScope =
            appliedFilters.scope === "DOCUMENT" && selectedDocument?.id
                ? "DOCUMENT"
                : appliedFilters.scope === "COLLECTION" && selectedDocument?.collection_id
                  ? "COLLECTION"
                  : "ALL";

        setIsLoadingMore(true);
        setError(null);

        try {
            const nextPage = await fetchDocumentAuditEvents({
                limit: 12,
                cursor: result.page_info.next_cursor,
                query: appliedFilters.query || undefined,
                action: appliedFilters.action === "ALL" ? undefined : appliedFilters.action,
                status: appliedFilters.status === "ALL" ? undefined : appliedFilters.status,
                correlationId: appliedFilters.correlationId || undefined,
                documentId: resolvedScope === "DOCUMENT" ? selectedDocument?.id : undefined,
                collectionId: resolvedScope === "COLLECTION" ? selectedDocument?.collection_id ?? undefined : undefined,
                createdFrom: toRFC3339FromDateTimeLocal(appliedFilters.createdFrom),
                createdTo: toRFC3339FromDateTimeLocal(appliedFilters.createdTo),
            });

            setResult((current) => ({
                ...nextPage,
                items: [...current.items, ...nextPage.items],
            }));
        } catch (nextError) {
            setError(nextError instanceof Error ? nextError.message : "Nao foi possivel carregar mais eventos de auditoria.");
        } finally {
            setIsLoadingMore(false);
        }
    }

    const failedEvents = getAuditBucketCount(result.summary.status_breakdown, "FAILED");
    const uploadEvents = getAuditBucketCount(result.summary.action_breakdown, "DOCUMENT_UPLOAD");
    const retryEvents = getAuditBucketCount(result.summary.action_breakdown, "DOCUMENT_RETRY_INDEX");
    const deleteEvents = getAuditBucketCount(result.summary.action_breakdown, "DOCUMENT_DELETE");

    return (
        <Card className="mb-6 border-border/80 bg-card/70">
            <CardHeader className="space-y-4">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                        <p className="text-[10px] font-bold uppercase tracking-[0.28em] text-primary">Audit Trail</p>
                        <h2 className="mt-2 text-lg font-semibold">Historico de auditoria documental</h2>
                        <p className="mt-1 text-sm text-muted-foreground">
                            Consulta operacional dos uploads, retries e exclusoes com request id, origem, recorte por documento e leitura agregada.
                        </p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                        <ScopeButton active={filters.scope === "ALL"} onClick={() => setFilters((current) => ({ ...current, scope: "ALL" }))}>
                            Conta inteira
                        </ScopeButton>
                        {selectedDocument ? (
                            <ScopeButton active={filters.scope === "DOCUMENT"} onClick={() => setFilters((current) => ({ ...current, scope: "DOCUMENT" }))}>
                                Documento selecionado
                            </ScopeButton>
                        ) : null}
                        {selectedDocument?.collection_id ? (
                            <ScopeButton active={filters.scope === "COLLECTION"} onClick={() => setFilters((current) => ({ ...current, scope: "COLLECTION" }))}>
                                Collection atual
                            </ScopeButton>
                        ) : null}
                    </div>
                </div>

                {selectedDocument ? (
                    <div className="rounded-xl border border-primary/15 bg-primary/5 px-4 py-3 text-sm text-foreground/85">
                        Painel contextual pronto para filtrar por <span className="font-semibold">{selectedDocument.title}</span>
                        {selectedDocument.collection ? ` na collection ${selectedDocument.collection}.` : "."}
                    </div>
                ) : null}
            </CardHeader>

            <CardContent className="space-y-6">
                <form
                    className="grid gap-4 rounded-2xl border border-border bg-background/55 p-4 lg:grid-cols-2 xl:grid-cols-4"
                    onSubmit={(event) => {
                        event.preventDefault();
                        setAppliedFilters(filters);
                    }}
                >
                    <div className="space-y-2 xl:col-span-2">
                        <Label htmlFor="document-audit-query">Busca operacional</Label>
                        <div className="relative">
                            <Search className="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                            <Input
                                id="document-audit-query"
                                value={filters.query}
                                onChange={(event) => setFilters((current) => ({ ...current, query: event.target.value }))}
                                className="pl-9"
                                placeholder="Titulo, detalhe, origem ou request id"
                            />
                        </div>
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="document-audit-action">Acao</Label>
                        <Input
                            id="document-audit-action"
                            list="document-audit-actions"
                            value={filters.action}
                            onChange={(event) => setFilters((current) => ({ ...current, action: event.target.value || "ALL" }))}
                            placeholder="ALL"
                        />
                        <datalist id="document-audit-actions">
                            <option value="ALL" />
                            <option value="DOCUMENT_UPLOAD" />
                            <option value="DOCUMENT_RETRY_INDEX" />
                            <option value="DOCUMENT_DELETE" />
                        </datalist>
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="document-audit-status">Status</Label>
                        <Input
                            id="document-audit-status"
                            list="document-audit-statuses"
                            value={filters.status}
                            onChange={(event) => setFilters((current) => ({ ...current, status: event.target.value || "ALL" }))}
                            placeholder="ALL"
                        />
                        <datalist id="document-audit-statuses">
                            <option value="ALL" />
                            <option value="SUCCESS" />
                            <option value="FAILED" />
                        </datalist>
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="document-audit-correlation">Request ID</Label>
                        <Input
                            id="document-audit-correlation"
                            value={filters.correlationId}
                            onChange={(event) => setFilters((current) => ({ ...current, correlationId: event.target.value }))}
                            placeholder="req-..."
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="document-audit-created-from">Criado de</Label>
                        <Input
                            id="document-audit-created-from"
                            type="datetime-local"
                            value={filters.createdFrom}
                            onChange={(event) => setFilters((current) => ({ ...current, createdFrom: event.target.value }))}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="document-audit-created-to">Criado ate</Label>
                        <Input
                            id="document-audit-created-to"
                            type="datetime-local"
                            value={filters.createdTo}
                            onChange={(event) => setFilters((current) => ({ ...current, createdTo: event.target.value }))}
                        />
                    </div>

                    <div className="flex items-end gap-2 xl:col-span-4">
                        <Button type="submit" className="gap-2">
                            <Filter className="h-4 w-4" />
                            Aplicar filtros
                        </Button>
                        <Button
                            type="button"
                            variant="outline"
                            className="gap-2"
                            onClick={() => {
                                setFilters(initialFilters);
                                setAppliedFilters(initialFilters);
                            }}
                        >
                            <RefreshCw className="h-4 w-4" />
                            Limpar
                        </Button>
                    </div>
                </form>

                <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                    <SummaryCard
                        label="Eventos"
                        value={String(result.summary.total_events)}
                        detail={`${failedEvents} falhas no recorte atual`}
                    />
                    <SummaryCard
                        label="Cobertura"
                        value={String(result.summary.unique_documents)}
                        detail={`${result.summary.unique_collections} collections envolvidas`}
                    />
                    <SummaryCard
                        label="Requests"
                        value={String(result.summary.unique_requests)}
                        detail="Correlacoes distintas registradas"
                    />
                    <SummaryCard
                        label="Ultimo evento"
                        value={formatAuditDateTime(result.summary.last_event_at, locale)}
                        detail={`${uploadEvents} uploads · ${retryEvents} retries · ${deleteEvents} exclusoes`}
                    />
                </div>

                {error ? (
                    <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                        {error}
                    </div>
                ) : null}

                {isLoading ? (
                    <div className="flex items-center gap-2 rounded-xl border border-dashed border-border bg-background/45 p-5 text-sm text-muted-foreground">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        Carregando auditoria documental...
                    </div>
                ) : null}

                {!isLoading && !error && result.items.length === 0 ? (
                    <div className="rounded-xl border border-dashed border-border bg-background/45 p-5 text-sm text-muted-foreground">
                        Nenhum evento de auditoria encontrado para esse recorte.
                    </div>
                ) : null}

                {!isLoading && result.items.length > 0 ? (
                    <div className="space-y-3">
                        {result.items.map((event) => (
                            <DocumentAuditEventCard key={event.id} event={event} locale={locale} />
                        ))}
                    </div>
                ) : null}

                {!isLoading && result.page_info.has_next ? (
                    <div className="flex justify-center">
                        <Button
                            type="button"
                            variant="outline"
                            onClick={() => {
                                void handleLoadMore();
                            }}
                            disabled={isLoadingMore}
                            className="gap-2"
                        >
                            {isLoadingMore ? <Loader2 className="h-4 w-4 animate-spin" /> : <Clock3 className="h-4 w-4" />}
                            Carregar mais eventos
                        </Button>
                    </div>
                ) : null}

                {!isLoading && result.items.length > 0 ? (
                    <p className="flex items-center gap-2 text-xs text-muted-foreground">
                        <FileSearch className="h-4 w-4" />
                        Exibindo {result.items.length} evento(s) no feed atual. O resumo acima considera todo o recorte filtrado, nao apenas a pagina atual.
                    </p>
                ) : null}
            </CardContent>
        </Card>
    );
}
