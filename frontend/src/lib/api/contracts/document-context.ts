export type CollectionMetadata = Record<string, unknown>;

export type CategorySummary = {
    id: number;
    userId?: number;
    name: string;
    description?: string;
    createdAt?: string;
    updatedAt?: string;
};

export type CollectionSummary = {
    id: number | string;
    name: string;
    categoryId?: number | null;
    documentCount: number;
    metadata: CollectionMetadata;
    createdAt?: string;
    updatedAt?: string;
};

export type DocumentSummary = {
    id: number;
    collectionId: number;
    title: string;
    filePath: string;
    mimeType: string;
    status: string;
    createdAt?: string;
    updatedAt?: string;
    /** Quando o indice atual deste documento foi gerado; compara com a fonte de uma resposta antiga. */
    indexedAt?: string;
};

export type ParagraphNode = {
    id: number | string;
    documentId: number;
    sequenceOrder: number;
    content: string;
    pageNumber?: number;
    metadata?: Record<string, unknown>;
};

export type TableSection = {
    id: number | string;
    documentId: number;
    sequenceOrder: number;
    /** Pagina fisica da tabela (PDF); null em HTML e planilha. */
    pageNumber: number | null;
    title?: string;
    headers: string[];
    rows: string[][];
    metadata?: Record<string, unknown>;
};

export type MetadataPair = {
    id: number | string;
    documentId: number;
    sequenceOrder: number;
    key: string;
    value: string;
};

export type OutlineNode = {
    id: number | string;
    /** Id estavel do no; necessario para pedir conexoes. */
    nodeId: string;
    documentId: number;
    sequenceOrder: number;
    level: number;
    title: string;
    slug?: string;
    summary?: string;
    /** Pagina em que a secao comeca (1-based), quando a indexacao soube. */
    startPage?: number;
    metadata?: Record<string, unknown>;
};

/**
* Caixa em pontos PDF, origem no topo esquerdo. So vem do pdftotext -bbox-layout
* ou do OCR com layout (tesseract, HTML e XLSX nao tem).
*/
export type ParagraphBox = {
    xMin: number;
    xMax: number;
    yMin: number;
    yMax: number;
    pageWidth: number;
    pageHeight: number;
};

function finito(value: unknown): value is number {
    return typeof value === "number" && Number.isFinite(value);
}

/** Le `metadata.box` de um paragrafo; `null` quando ausente ou malformado. */
/** Linha visual de um paragrafo, com a caixa no mesmo espaco de readBox. */
export type ParagraphLine = { text: string; box: ParagraphBox };

/**
 * Le metadata.lines ([{t, b: [xMin, yMin, xMax, yMax]}]) usando as dimensoes
 * da pagina de metadata.box. Sem caixa ou sem linhas, lista vazia: o grifo
 * cai no paragrafo inteiro.
 */
export function readLines(node: Pick<ParagraphNode, "metadata">): ParagraphLine[] {
    const box = readBox(node);
    const raw = node.metadata?.lines;
    if (!box || !Array.isArray(raw)) {
        return [];
    }
    const out: ParagraphLine[] = [];
    for (const item of raw) {
        if (typeof item !== "object" || item === null) {
            continue;
        }
        const r = item as Record<string, unknown>;
        const b = r.b;
        if (!Array.isArray(b) || b.length !== 4 || !b.every(finito)) {
            continue;
        }
        out.push({
            text: typeof r.t === "string" ? r.t : "",
            box: { xMin: b[0] as number, yMin: b[1] as number, xMax: b[2] as number, yMax: b[3] as number, pageWidth: box.pageWidth, pageHeight: box.pageHeight },
        });
    }
    return out;
}

export function readBox(node: Pick<ParagraphNode, "metadata">): ParagraphBox | null {
    const raw = node.metadata?.box;
    if (typeof raw !== "object" || raw === null || Array.isArray(raw)) {
        return null;
    }
    const r = raw as Record<string, unknown>;
    const xMin = r.x_min, xMax = r.x_max, yMin = r.y_min, yMax = r.y_max;
    const pageWidth = r.page_width, pageHeight = r.page_height;
    if (![xMin, xMax, yMin, yMax, pageWidth, pageHeight].every(finito)) {
        return null;
    }
    if ((pageWidth as number) <= 0 || (pageHeight as number) <= 0 || (xMax as number) < (xMin as number) || (yMax as number) < (yMin as number)) {
        return null;
    }
    return {
        xMin: xMin as number,
        xMax: xMax as number,
        yMin: yMin as number,
        yMax: yMax as number,
        pageWidth: pageWidth as number,
        pageHeight: pageHeight as number,
    };
}

/** Resumo e topicos de uma pagina, gravados pelo enriquecimento por LLM. */
export type DocumentPageSummary = {
    documentId: number;
    pageNumber: number;
    summary: string;
    topics: string[];
};

export type CollectionContext = {
    collection: CollectionSummary;
    documents: DocumentSummary[];
    paragraphNodes: ParagraphNode[];
    tableSections: TableSection[];
    metadataPairs: MetadataPair[];
    outlineNodes: OutlineNode[];
    /** Opcional: fixtures antigas nao trazem; a tela trata como vazio. */
    documentPages?: DocumentPageSummary[];
};

type CollectionApiRecord = {
    id?: number | string;
    name?: string;
    metadata?: CollectionMetadata | null;
    document_count?: number;
    documentCount?: number;
    category_id?: number | null;
    categoryId?: number | null;
    created_at?: string;
    createdAt?: string;
    updated_at?: string;
    updatedAt?: string;
};

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isCollectionApiRecord(value: unknown): value is CollectionApiRecord {
    return isRecord(value) && typeof value.name === "string";
}

export function getCollectionName(value: Pick<CollectionSummary, "name"> | string): string {
    return typeof value === "string" ? value : value.name;
}

export function normalizeCollectionSummary(value: unknown): CollectionSummary | null {
    if (typeof value === "string") {
        const name = value.trim();
        if (!name) {
            return null;
        }

        return {
            id: `legacy:${name.toLowerCase()}`,
            name,
            documentCount: 0,
            metadata: {},
        };
    }

    if (!isCollectionApiRecord(value)) {
        return null;
    }

    const name = value.name?.trim();
    if (!name) {
        return null;
    }

    return {
        id: value.id ?? `legacy:${name.toLowerCase()}`,
        name,
        categoryId: value.categoryId ?? value.category_id ?? null,
        documentCount: value.documentCount ?? value.document_count ?? 0,
        metadata: isRecord(value.metadata) ? value.metadata : {},
        createdAt: value.createdAt ?? value.created_at,
        updatedAt: value.updatedAt ?? value.updated_at,
    };
}

export function normalizeCollectionSummaries(value: unknown): CollectionSummary[] {
    if (!Array.isArray(value)) {
        return [];
    }

    return value.reduce<CollectionSummary[]>((items, entry) => {
        const normalized = normalizeCollectionSummary(entry);
        if (normalized) {
            items.push(normalized);
        }
        return items;
    }, []);
}

type DocumentApiRecord = {
    id?: number;
    collection_id?: number;
    collectionId?: number;
    title?: string;
    file_path?: string;
    filePath?: string;
    file_url?: string;
    fileUrl?: string;
    mime_type?: string;
    mimeType?: string;
    status?: string;
    created_at?: string;
    createdAt?: string;
    updated_at?: string;
    updatedAt?: string;
    indexed_at?: string;
    indexedAt?: string;
};

type ParagraphNodeApiRecord = {
    id?: number | string;
    document_id?: number;
    documentId?: number;
    sequence_order?: number;
    sequenceOrder?: number;
    content?: string;
    page_number?: number;
    pageNumber?: number;
    metadata?: Record<string, unknown>;
};

type TableSectionApiRecord = {
    id?: number | string;
    document_id?: number;
    documentId?: number;
    sequence_order?: number;
    sequenceOrder?: number;
    page_number?: number | null;
    pageNumber?: number | null;
    title?: string;
    headers?: string[];
    rows?: string[][];
    metadata?: Record<string, unknown>;
};

type MetadataPairApiRecord = {
    id?: number | string;
    document_id?: number;
    documentId?: number;
    sequence_order?: number;
    sequenceOrder?: number;
    key?: string;
    value?: string;
};

type OutlineNodeApiRecord = {
    id?: number | string;
    node_id?: string;
    nodeId?: string;
    document_id?: number;
    documentId?: number;
    sequence_order?: number;
    sequenceOrder?: number;
    level?: number;
    title?: string;
    slug?: string;
    summary?: string;
    start_page?: number;
    startPage?: number;
    metadata?: Record<string, unknown>;
};

function normalizeDocumentSummary(value: unknown): DocumentSummary | null {
    if (!isRecord(value) || typeof value.id !== "number" || typeof value.title !== "string") {
        return null;
    }

    const record = value as DocumentApiRecord;

    return {
        id: record.id!,
        collectionId: record.collectionId ?? record.collection_id ?? 0,
        title: record.title!,
        filePath: record.filePath ?? record.file_path ?? record.fileUrl ?? record.file_url ?? "",
        mimeType: record.mimeType ?? record.mime_type ?? "",
        status: record.status ?? "",
        createdAt: record.createdAt ?? record.created_at,
        updatedAt: record.updatedAt ?? record.updated_at,
        ...(typeof (record.indexedAt ?? record.indexed_at) === "string" ? { indexedAt: (record.indexedAt ?? record.indexed_at) as string } : {}),
    };
}

function normalizeParagraphNode(value: unknown): ParagraphNode | null {
    if (!isRecord(value) || typeof value.content !== "string") {
        return null;
    }

    const record = value as ParagraphNodeApiRecord;
    const documentId = record.documentId ?? record.document_id;
    const sequenceOrder = record.sequenceOrder ?? record.sequence_order;

    if (typeof documentId !== "number" || typeof sequenceOrder !== "number") {
        return null;
    }

    return {
        id: record.id ?? `paragraph:${documentId}:${sequenceOrder}`,
        documentId,
        sequenceOrder,
        content: record.content!,
        pageNumber: record.pageNumber ?? record.page_number,
        metadata: isRecord(record.metadata) ? record.metadata : {},
    };
}

function normalizeTableSection(value: unknown): TableSection | null {
    if (!isRecord(value)) {
        return null;
    }

    const record = value as TableSectionApiRecord;
    const documentId = record.documentId ?? record.document_id;
    const sequenceOrder = record.sequenceOrder ?? record.sequence_order;

    if (typeof documentId !== "number" || typeof sequenceOrder !== "number") {
        return null;
    }

    return {
        id: record.id ?? `table:${documentId}:${sequenceOrder}`,
        documentId,
        sequenceOrder,
        pageNumber: typeof (record.pageNumber ?? record.page_number) === "number" ? (record.pageNumber ?? record.page_number)! : null,
        title: record.title,
        headers: Array.isArray(record.headers) ? record.headers.filter((item): item is string => typeof item === "string") : [],
        rows: Array.isArray(record.rows)
            ? record.rows.map((row) => (Array.isArray(row) ? row.filter((item): item is string => typeof item === "string") : []))
            : [],
        metadata: isRecord(record.metadata) ? record.metadata : {},
    };
}

function normalizeMetadataPair(value: unknown): MetadataPair | null {
    if (!isRecord(value) || typeof value.key !== "string" || typeof value.value !== "string") {
        return null;
    }

    const record = value as MetadataPairApiRecord;
    const documentId = record.documentId ?? record.document_id;
    const sequenceOrder = record.sequenceOrder ?? record.sequence_order;

    if (typeof documentId !== "number" || typeof sequenceOrder !== "number") {
        return null;
    }

    return {
        id: record.id ?? `metadata:${documentId}:${sequenceOrder}:${record.key}`,
        documentId,
        sequenceOrder,
        key: record.key!,
        value: record.value!,
    };
}

function normalizeOutlineNode(value: unknown): OutlineNode | null {
    if (!isRecord(value) || typeof value.title !== "string") {
        return null;
    }

    const record = value as OutlineNodeApiRecord;
    const documentId = record.documentId ?? record.document_id;
    const sequenceOrder = record.sequenceOrder ?? record.sequence_order;
    const level = record.level;

    if (typeof documentId !== "number" || typeof sequenceOrder !== "number" || typeof level !== "number") {
        return null;
    }

    return {
        id: record.id ?? `outline:${documentId}:${sequenceOrder}`,
        nodeId: record.nodeId ?? record.node_id ?? "",
        documentId,
        sequenceOrder,
        level,
        title: record.title!,
        slug: record.slug,
        summary: record.summary,
        startPage: finito(record.startPage ?? record.start_page) ? (record.startPage ?? record.start_page) : undefined,
        metadata: isRecord(record.metadata) ? record.metadata : {},
    };
}

export function normalizeOutlineNodes(value: unknown): OutlineNode[] {
    if (!Array.isArray(value)) {
        return [];
    }

    return value.reduce<OutlineNode[]>((items, entry) => {
        const normalized = normalizeOutlineNode(entry);
        if (normalized) {
            items.push(normalized);
        }
        return items;
    }, []);
}

export function normalizeParagraphNodes(value: unknown): ParagraphNode[] {
    if (!Array.isArray(value)) {
        return [];
    }

    return value.reduce<ParagraphNode[]>((items, entry) => {
        const normalized = normalizeParagraphNode(entry);
        if (normalized) {
            items.push(normalized);
        }
        return items;
    }, []);
}

export function normalizeTableSections(value: unknown): TableSection[] {
    if (!Array.isArray(value)) {
        return [];
    }

    return value.reduce<TableSection[]>((items, entry) => {
        const normalized = normalizeTableSection(entry);
        if (normalized) {
            items.push(normalized);
        }
        return items;
    }, []);
}

export function normalizeMetadataPairs(value: unknown): MetadataPair[] {
    if (!Array.isArray(value)) {
        return [];
    }

    return value.reduce<MetadataPair[]>((items, entry) => {
        const normalized = normalizeMetadataPair(entry);
        if (normalized) {
            items.push(normalized);
        }
        return items;
    }, []);
}

export function normalizeCollectionContext(value: unknown): CollectionContext | null {
    if (!isRecord(value)) {
        return null;
    }

    const collection = normalizeCollectionSummary(value.collection);
    if (!collection) {
        return null;
    }

    return {
        collection,
        documents: Array.isArray(value.documents)
            ? value.documents.reduce<DocumentSummary[]>((items, entry) => {
                  const normalized = normalizeDocumentSummary(entry);
                  if (normalized) {
                      items.push(normalized);
                  }
                  return items;
              }, [])
            : [],
        paragraphNodes: normalizeParagraphNodes(value.paragraph_nodes),
        tableSections: normalizeTableSections(value.table_sections),
        metadataPairs: normalizeMetadataPairs(value.metadata_pairs),
        outlineNodes: normalizeOutlineNodes(value.outline_nodes),
        documentPages: normalizeDocumentPages(value.document_pages),
    };
}

/** So o que a tela usa: numero, resumo e topicos. O conteudo da pagina fica de fora. */
export function normalizeDocumentPages(value: unknown): DocumentPageSummary[] {
    if (!Array.isArray(value)) {
        return [];
    }
    const out: DocumentPageSummary[] = [];
    for (const entry of value) {
        if (!isRecord(entry)) {
            continue;
        }
        const documentId = Number(entry.document_id ?? entry.documentId);
        const pageNumber = Number(entry.page_number ?? entry.pageNumber);
        if (!Number.isFinite(documentId) || !Number.isFinite(pageNumber)) {
            continue;
        }
        const summary = typeof entry.summary === "string" ? entry.summary.trim() : "";
        const meta = isRecord(entry.metadata) ? entry.metadata : {};
        const topics = typeof meta.topics === "string" ? meta.topics.split(",").map((s) => s.trim()).filter(Boolean).slice(0, 5) : [];
        if (summary || topics.length > 0) {
            out.push({ documentId, pageNumber, summary, topics });
        }
    }
    return out;
}
