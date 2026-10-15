"use client";

import { useEffect, useRef, useState } from "react";
import { ChevronDown, ChevronUp, Clock3, FileSearch, Filter, Loader2, RefreshCw, Search } from "lucide-react";

import AuditMetadataList from "@/components/audit/AuditMetadataList";
import {
    AuditAppliedFilters,
    AuditQuickFilterButton,
    AuditScopeButton,
    AuditSummaryCard,
} from "@/components/audit/AuditExplorerShared";
import AuditStatusBadge from "@/components/audit/AuditStatusBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    fetchDocumentAuditEvents,
    type DocumentAuditEvent,
    type DocumentAuditQueryResult,
} from "@/lib/api/audit";
import {
    formatAuditActor,
    formatAuditActorType,
    formatAuditDateTime,
    formatAuditStatus,
    formatDocumentAuditAction,
    getAuditBucketCount,
    normalizeAuditFilterValue,
    parseAuditActorId,
    toRFC3339FromDateTimeLocal,
} from "@/lib/audit";
import type { AppLocale } from "@/lib/i18n/config";
import { translate } from "@/lib/i18n/translate";

type Scope = "ALL" | "DOCUMENT" | "COLLECTION";

type DocumentAuditExplorerProps = {
    locale: AppLocale;
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
    actorType: string;
    actorId: string;
    query: string;
    correlationId: string;
    createdFrom: string;
    createdTo: string;
};

const initialFilters: AuditFilters = {
    scope: "ALL",
    action: "",
    status: "",
    actorType: "",
    actorId: "",
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

const selectClassName =
    "flex h-10 w-full rounded-md border border-input bg-secondary/30 px-3 py-2 text-sm ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2";

function DocumentAuditEventCard({
    event,
    locale,
}: {
    event: DocumentAuditEvent;
    locale: AppLocale;
}) {
    const t = (key: string, variables?: Record<string, string | number | boolean>) => translate(locale, key, variables);

    return (
        <div className="rounded-xl border border-border bg-background/70 p-4">
            <div className="flex flex-wrap items-center gap-2">
                <p className="text-sm font-semibold">{event.title || t("audit.documents.eventTitleFallback", { id: event.document_id ?? event.id })}</p>
                <span className="rounded-full border border-border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {formatDocumentAuditAction(event.action, locale)}
                </span>
                <AuditStatusBadge status={event.status} locale={locale} />
            </div>

            <p className="mt-2 text-sm text-foreground/90">{event.detail || t("audit.common.noDetail")}</p>

            <AuditMetadataList metadata={event.metadata} />

            <div className="mt-3 grid gap-2 text-[11px] text-muted-foreground md:grid-cols-2 xl:grid-cols-3">
                <p>{t("audit.common.when")}: <span className="font-mono text-foreground">{formatAuditDateTime(event.created_at, locale)}</span></p>
                <p>{t("audit.common.origin")}: <span className="font-mono text-foreground">{event.origin || "-"}</span></p>
                <p>{t("audit.common.request")}: <span className="font-mono text-foreground">{event.correlation_id || "-"}</span></p>
                <p>{t("audit.common.actor")}: <span className="font-mono text-foreground">{formatAuditActor(event.actor_type, event.actor_id, locale)}</span></p>
                <p>{t("audit.common.document")}: <span className="font-mono text-foreground">{event.document_id ?? "-"}</span></p>
                <p>{t("audit.common.collection")}: <span className="font-mono text-foreground">{event.collection_id ?? "-"}</span></p>
            </div>
        </div>
    );
}

export default function DocumentAuditExplorer({
    locale,
    reloadToken = 0,
    selectedDocument,
}: DocumentAuditExplorerProps) {
    const t = (key: string, variables?: Record<string, string | number | boolean>) => translate(locale, key, variables);
    const [filters, setFilters] = useState<AuditFilters>(initialFilters);
    const [appliedFilters, setAppliedFilters] = useState<AuditFilters>(initialFilters);
    const [result, setResult] = useState<DocumentAuditQueryResult>(emptyResult);
    // Id da consulta ativa: descarta pagina em voo de um filtro ja trocado.
    // Ref porque e lida depois do await.
    const consultaAtivaRef = useRef(0);
    const [error, setError] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [isLoadingMore, setIsLoadingMore] = useState(false);
    const [showAdvancedFilters, setShowAdvancedFilters] = useState(false);

    // Parte de `filters`, nao de appliedFilters: senao apaga o que foi digitado
    // e ainda nao submetido.
    function applyImmediateFilters(partial: Partial<AuditFilters>) {
        const nextFilters = {
            ...filters,
            ...partial,
        };
        setFilters(nextFilters);
        setAppliedFilters(nextFilters);
    }

    useEffect(() => {
        if (filters.scope === "DOCUMENT" && !selectedDocument) {
            setFilters((current) => ({ ...current, scope: "ALL" }));
        }
        if (filters.scope === "COLLECTION" && !selectedDocument?.collection_id) {
            setFilters((current) => ({ ...current, scope: "ALL" }));
        }
    }, [filters.scope, selectedDocument, selectedDocument?.collection_id]);

    const hasAdvancedFilters =
        filters.action !== "" ||
        filters.status !== "" ||
        filters.actorType !== "" ||
        filters.actorId.trim() !== "" ||
        filters.correlationId.trim() !== "" ||
        filters.createdFrom !== "" ||
        filters.createdTo !== "";

    useEffect(() => {
        if (hasAdvancedFilters) {
            setShowAdvancedFilters(true);
        }
    }, [hasAdvancedFilters]);

    useEffect(() => {
        let cancelled = false;
        consultaAtivaRef.current += 1;

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
                    action: appliedFilters.action || undefined,
                    status: appliedFilters.status || undefined,
                    actorType: appliedFilters.actorType || undefined,
                    actorId: parseAuditActorId(appliedFilters.actorId),
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
                    setError(nextError instanceof Error ? nextError.message : translate(locale, "audit.documents.errors.load"));
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
    }, [appliedFilters, locale, reloadToken, selectedDocument?.collection_id, selectedDocument?.id]);

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

        const consultaDoClique = consultaAtivaRef.current;

        setIsLoadingMore(true);
        setError(null);

        try {
            const nextPage = await fetchDocumentAuditEvents({
                limit: 12,
                cursor: result.page_info.next_cursor,
                query: appliedFilters.query || undefined,
                action: appliedFilters.action || undefined,
                status: appliedFilters.status || undefined,
                actorType: appliedFilters.actorType || undefined,
                actorId: parseAuditActorId(appliedFilters.actorId),
                correlationId: appliedFilters.correlationId || undefined,
                documentId: resolvedScope === "DOCUMENT" ? selectedDocument?.id : undefined,
                collectionId: resolvedScope === "COLLECTION" ? selectedDocument?.collection_id ?? undefined : undefined,
                createdFrom: toRFC3339FromDateTimeLocal(appliedFilters.createdFrom),
                createdTo: toRFC3339FromDateTimeLocal(appliedFilters.createdTo),
            });

            if (consultaDoClique !== consultaAtivaRef.current) {
                return;
            }

            setResult((current) => ({
                ...nextPage,
                items: [...current.items, ...nextPage.items],
            }));
        } catch (nextError) {
            if (consultaDoClique !== consultaAtivaRef.current) {
                return;
            }
            setError(nextError instanceof Error ? nextError.message : translate(locale, "audit.documents.errors.loadMore"));
        } finally {
            // Desliga o spinner mesmo se a resposta foi descartada.
            setIsLoadingMore(false);
        }
    }

    const failedEvents = getAuditBucketCount(result.summary.status_breakdown, "FAILED");
    const uploadEvents = getAuditBucketCount(result.summary.action_breakdown, "DOCUMENT_UPLOAD");
    const retryEvents = getAuditBucketCount(result.summary.action_breakdown, "DOCUMENT_RETRY_INDEX");
    const deleteEvents = getAuditBucketCount(result.summary.action_breakdown, "DOCUMENT_DELETE");
    const appliedActorId = parseAuditActorId(appliedFilters.actorId);
    const advancedFilterCount = [
        filters.action,
        filters.status,
        filters.actorType,
        filters.actorId.trim(),
        filters.correlationId.trim(),
        filters.createdFrom,
        filters.createdTo,
    ].filter(Boolean).length;
    const activeFilterItems = [
        appliedFilters.scope !== "ALL"
            ? {
                  key: "scope",
                  label: t("audit.common.scope"),
                  value: appliedFilters.scope === "DOCUMENT"
                      ? t("audit.documents.scopeSelectedDocument")
                      : t("audit.documents.scopeCurrentCollection"),
              }
            : null,
        appliedFilters.query
            ? {
                  key: "query",
                  label: t("audit.common.search"),
                  value: appliedFilters.query,
              }
            : null,
        appliedFilters.action
            ? {
                  key: "action",
                  label: t("audit.common.action"),
                  value: formatDocumentAuditAction(appliedFilters.action, locale),
              }
            : null,
        appliedFilters.status
            ? {
                  key: "status",
                  label: t("audit.common.status"),
                  value: formatAuditStatus(appliedFilters.status, locale),
              }
            : null,
        appliedFilters.actorType
            ? {
                  key: "actor-type",
                  label: t("audit.common.actor"),
                  value: formatAuditActorType(appliedFilters.actorType, locale),
              }
            : null,
        typeof appliedActorId === "number"
            ? {
                  key: "actor-id",
                  label: t("audit.common.actorId"),
                  value: String(appliedActorId),
              }
            : null,
        appliedFilters.correlationId
            ? {
                  key: "correlation",
                  label: t("audit.common.request"),
                  value: appliedFilters.correlationId,
              }
            : null,
        appliedFilters.createdFrom
            ? {
                  key: "created-from",
                  label: t("audit.common.createdFrom"),
                  value: appliedFilters.createdFrom,
              }
            : null,
        appliedFilters.createdTo
            ? {
                  key: "created-to",
                  label: t("audit.common.createdTo"),
                  value: appliedFilters.createdTo,
              }
            : null,
    ].filter((item): item is { key: string; label: string; value: string } => item !== null);

    return (
        <Card className="mb-6 border-border/80 bg-card/70">
            <CardHeader className="space-y-4">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                        <p className="text-[10px] font-bold uppercase tracking-[0.28em] text-primary">{t("audit.common.eyebrow")}</p>
                        <h2 className="mt-2 text-lg font-semibold">{t("audit.documents.title")}</h2>
                        <p className="mt-1 text-sm text-muted-foreground">
                            {t("audit.documents.description")}
                        </p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                        <AuditScopeButton active={filters.scope === "ALL"} onClick={() => setFilters((current) => ({ ...current, scope: "ALL" }))}>
                            {t("audit.documents.scopeAccount")}
                        </AuditScopeButton>
                        {selectedDocument ? (
                            <AuditScopeButton active={filters.scope === "DOCUMENT"} onClick={() => setFilters((current) => ({ ...current, scope: "DOCUMENT" }))}>
                                {t("audit.documents.scopeSelectedDocument")}
                            </AuditScopeButton>
                        ) : null}
                        {selectedDocument?.collection_id ? (
                            <AuditScopeButton active={filters.scope === "COLLECTION"} onClick={() => setFilters((current) => ({ ...current, scope: "COLLECTION" }))}>
                                {t("audit.documents.scopeCurrentCollection")}
                            </AuditScopeButton>
                        ) : null}
                    </div>
                </div>

                {selectedDocument ? (
                    <div className="rounded-xl border border-primary/15 bg-primary/5 px-4 py-3 text-sm text-foreground/85">
                        {t("audit.documents.contextReady", { title: selectedDocument.title })}
                        {selectedDocument.collection
                            ? ` ${t("audit.documents.contextReadyCollection", { collection: selectedDocument.collection })}`
                            : "."}
                    </div>
                ) : null}
            </CardHeader>

            <CardContent className="space-y-6">
                <form
                    className="space-y-4 rounded-2xl border border-border bg-background/55 p-4"
                    onSubmit={(event) => {
                        event.preventDefault();
                        setAppliedFilters(filters);
                    }}
                >
                    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto_auto] lg:items-end">
                        <div className="space-y-2">
                            <Label htmlFor="document-audit-query">{t("audit.documents.quickSearchLabel")}</Label>
                            <div className="relative">
                                <Search className="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                                <Input
                                    id="document-audit-query"
                                    value={filters.query}
                                    onChange={(event) => setFilters((current) => ({ ...current, query: event.target.value }))}
                                    className="pl-9"
                                    placeholder={t("audit.documents.searchPlaceholder")}
                                />
                            </div>
                            <p className="text-xs text-muted-foreground">{t("audit.documents.searchHint")}</p>
                        </div>

                        <Button type="submit" className="gap-2">
                            <Search className="h-4 w-4" />
                            {t("audit.documents.searchAction")}
                        </Button>
                        <Button
                            type="button"
                            variant="outline"
                            className="gap-2"
                            onClick={() => {
                                setFilters(initialFilters);
                                setAppliedFilters(initialFilters);
                                setShowAdvancedFilters(false);
                            }}
                        >
                            <RefreshCw className="h-4 w-4" />
                            {t("audit.documents.clearAll")}
                        </Button>
                    </div>

                    <div className="rounded-xl border border-border bg-background/45 p-4">
                        <p className="text-[10px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">
                            {t("audit.documents.quickFiltersTitle")}
                        </p>
                        <p className="mt-2 text-xs text-muted-foreground">{t("audit.documents.quickFiltersDescription")}</p>
                        <div className="mt-3 flex flex-wrap gap-2">
                            <AuditQuickFilterButton
                                label={t("audit.common.onlyFailures")}
                                count={failedEvents}
                                active={appliedFilters.status === "FAILED"}
                                tone="danger"
                                onClick={() =>
                                    applyImmediateFilters({
                                        status: appliedFilters.status === "FAILED" ? "" : "FAILED",
                                    })
                                }
                            />
                            <AuditQuickFilterButton
                                label={t("audit.documents.quickFilters.uploads")}
                                count={uploadEvents}
                                active={appliedFilters.action === "DOCUMENT_UPLOAD"}
                                onClick={() =>
                                    applyImmediateFilters({
                                        action: appliedFilters.action === "DOCUMENT_UPLOAD" ? "" : "DOCUMENT_UPLOAD",
                                    })
                                }
                            />
                            <AuditQuickFilterButton
                                label={t("audit.documents.quickFilters.retries")}
                                count={retryEvents}
                                active={appliedFilters.action === "DOCUMENT_RETRY_INDEX"}
                                onClick={() =>
                                    applyImmediateFilters({
                                        action: appliedFilters.action === "DOCUMENT_RETRY_INDEX" ? "" : "DOCUMENT_RETRY_INDEX",
                                    })
                                }
                            />
                            <AuditQuickFilterButton
                                label={t("audit.documents.quickFilters.deletes")}
                                count={deleteEvents}
                                active={appliedFilters.action === "DOCUMENT_DELETE"}
                                onClick={() =>
                                    applyImmediateFilters({
                                        action: appliedFilters.action === "DOCUMENT_DELETE" ? "" : "DOCUMENT_DELETE",
                                    })
                                }
                            />
                        </div>
                    </div>

                    <div className="rounded-xl border border-border bg-background/45 p-4">
                        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                            <div>
                                <p className="text-[10px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">
                                    {t("audit.documents.advancedFiltersTitle")}
                                </p>
                                <p className="mt-2 text-xs text-muted-foreground">
                                    {t("audit.documents.advancedFiltersDescription")}
                                </p>
                            </div>
                            <button
                                type="button"
                                onClick={() => setShowAdvancedFilters((current) => !current)}
                                className="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-xs font-semibold text-foreground transition hover:bg-muted"
                                aria-expanded={showAdvancedFilters}
                            >
                                <Filter className="h-4 w-4" />
                                {showAdvancedFilters
                                    ? t("audit.documents.advancedFiltersHide")
                                    : t("audit.documents.advancedFiltersShow", { count: advancedFilterCount })}
                                {showAdvancedFilters ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                            </button>
                        </div>

                        {showAdvancedFilters ? (
                            <div className="mt-4 grid gap-4 lg:grid-cols-2 xl:grid-cols-4">
                                <div className="space-y-2">
                                    <Label htmlFor="document-audit-action">{t("audit.common.action")}</Label>
                                    <select
                                        id="document-audit-action"
                                        value={filters.action}
                                        onChange={(event) => setFilters((current) => ({ ...current, action: normalizeAuditFilterValue(event.target.value) }))}
                                        className={selectClassName}
                                    >
                                        <option value="">{t("audit.documents.anyOption")}</option>
                                        <option value="DOCUMENT_UPLOAD">{t("audit.documents.quickFilters.uploads")}</option>
                                        <option value="DOCUMENT_RETRY_INDEX">{t("audit.documents.quickFilters.retries")}</option>
                                        <option value="DOCUMENT_DELETE">{t("audit.documents.quickFilters.deletes")}</option>
                                    </select>
                                </div>

                                <div className="space-y-2">
                                    <Label htmlFor="document-audit-status">{t("audit.common.status")}</Label>
                                    <select
                                        id="document-audit-status"
                                        value={filters.status}
                                        onChange={(event) => setFilters((current) => ({ ...current, status: normalizeAuditFilterValue(event.target.value) }))}
                                        className={selectClassName}
                                    >
                                        <option value="">{t("audit.documents.anyOption")}</option>
                                        <option value="SUCCESS">{t("audit.status.success")}</option>
                                        <option value="FAILED">{t("audit.status.failed")}</option>
                                    </select>
                                </div>

                                <div className="space-y-2">
                                    <Label htmlFor="document-audit-created-from">{t("audit.common.createdFrom")}</Label>
                                    <Input
                                        id="document-audit-created-from"
                                        type="datetime-local"
                                        value={filters.createdFrom}
                                        onChange={(event) => setFilters((current) => ({ ...current, createdFrom: event.target.value }))}
                                    />
                                </div>

                                <div className="space-y-2">
                                    <Label htmlFor="document-audit-created-to">{t("audit.common.createdTo")}</Label>
                                    <Input
                                        id="document-audit-created-to"
                                        type="datetime-local"
                                        value={filters.createdTo}
                                        onChange={(event) => setFilters((current) => ({ ...current, createdTo: event.target.value }))}
                                    />
                                </div>

                                <div className="space-y-2">
                                    <Label htmlFor="document-audit-correlation">{t("audit.common.requestId")}</Label>
                                    <Input
                                        id="document-audit-correlation"
                                        value={filters.correlationId}
                                        onChange={(event) => setFilters((current) => ({ ...current, correlationId: event.target.value }))}
                                        placeholder="req-..."
                                    />
                                </div>

                                <div className="space-y-2">
                                    <Label htmlFor="document-audit-actor-type">{t("audit.common.actor")}</Label>
                                    <select
                                        id="document-audit-actor-type"
                                        value={filters.actorType}
                                        onChange={(event) => setFilters((current) => ({ ...current, actorType: normalizeAuditFilterValue(event.target.value) }))}
                                        className={selectClassName}
                                    >
                                        <option value="">{t("audit.documents.anyOption")}</option>
                                        <option value="USER">{t("audit.actorType.user")}</option>
                                        <option value="SYSTEM">{t("audit.actorType.system")}</option>
                                        <option value="SERVICE">{t("audit.actorType.service")}</option>
                                        <option value="WORKER">{t("audit.actorType.worker")}</option>
                                    </select>
                                </div>

                                <div className="space-y-2">
                                    <Label htmlFor="document-audit-actor-id">{t("audit.common.actorId")}</Label>
                                    <Input
                                        id="document-audit-actor-id"
                                        type="number"
                                        min="1"
                                        step="1"
                                        value={filters.actorId}
                                        onChange={(event) => setFilters((current) => ({ ...current, actorId: event.target.value }))}
                                        placeholder="9"
                                    />
                                </div>
                            </div>
                        ) : null}
                    </div>
                </form>

                <AuditAppliedFilters items={activeFilterItems} title={t("audit.common.activeFilters")} />

                <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                    <AuditSummaryCard
                        label={t("audit.documents.summary.eventsLabel")}
                        value={String(result.summary.total_events)}
                        detail={t("audit.documents.summary.eventsDetail", { count: failedEvents })}
                    />
                    <AuditSummaryCard
                        label={t("audit.documents.summary.coverageLabel")}
                        value={String(result.summary.unique_documents)}
                        detail={t("audit.documents.summary.coverageDetail", { count: result.summary.unique_collections })}
                    />
                    <AuditSummaryCard
                        label={t("audit.documents.summary.requestsLabel")}
                        value={String(result.summary.unique_requests)}
                        detail={t("audit.documents.summary.requestsDetail")}
                    />
                    <AuditSummaryCard
                        label={t("audit.documents.summary.lastEventLabel")}
                        value={formatAuditDateTime(result.summary.last_event_at, locale)}
                        detail={t("audit.documents.summary.lastEventDetail", {
                            uploads: uploadEvents,
                            retries: retryEvents,
                            deletes: deleteEvents,
                        })}
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
                        {t("audit.documents.loading")}
                    </div>
                ) : null}

                {!isLoading && !error && result.items.length === 0 ? (
                    <div className="rounded-xl border border-dashed border-border bg-background/45 p-5 text-sm text-muted-foreground">
                        {t("audit.documents.empty")}
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
                            {t("audit.common.loadMore")}
                        </Button>
                    </div>
                ) : null}

                {!isLoading && result.items.length > 0 ? (
                    <p className="flex items-center gap-2 text-xs text-muted-foreground">
                        <FileSearch className="h-4 w-4" />
                        {t("audit.documents.feedDetail", { count: result.items.length })}
                    </p>
                ) : null}
            </CardContent>
        </Card>
    );
}
