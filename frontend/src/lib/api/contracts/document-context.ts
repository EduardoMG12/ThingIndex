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

export type CollectionContext = {
    collection: CollectionSummary;
    documents: DocumentSummary[];
    paragraphNodes: ParagraphNode[];
    tableSections: TableSection[];
    metadataPairs: MetadataPair[];
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
