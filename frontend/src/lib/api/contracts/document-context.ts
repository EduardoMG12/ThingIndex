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
    documentId: number;
    sequenceOrder: number;
    level: number;
    title: string;
    slug?: string;
    summary?: string;
    metadata?: Record<string, unknown>;
};

export type CollectionContext = {
    collection: CollectionSummary;
    documents: DocumentSummary[];
    paragraphNodes: ParagraphNode[];
    tableSections: TableSection[];
    metadataPairs: MetadataPair[];
    outlineNodes: OutlineNode[];
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
    document_id?: number;
    documentId?: number;
    sequence_order?: number;
    sequenceOrder?: number;
    level?: number;
    title?: string;
    slug?: string;
    summary?: string;
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
        documentId,
        sequenceOrder,
        level,
        title: record.title!,
        slug: record.slug,
        summary: record.summary,
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
    };
}
