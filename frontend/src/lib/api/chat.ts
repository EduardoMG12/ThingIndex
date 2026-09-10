import { UnauthorizedRequestError } from "@/lib/api/fetch-with-auth-redirect";
import { apiErrorMessage } from "@/lib/api/error";
import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";

export type ChatHistoryMessage = {
    role: "user" | "assistant";
    content: string;
};

export type ChatConversationSummary = {
    id: number;
    collectionId: number;
    title: string;
    lastMessageAt: string | null;
    createdAt: string;
    updatedAt: string;
};

/** Consumo de uma resposta. `usageReported=false` = estimativa (runas/4), rotulada na tela. */
export type ChatUsage = {
    promptTokens: number;
    completionTokens: number;
    usageReported: boolean;
    providerType?: string;
    model?: string;
    estimatedCostUsd?: number;
};

export type PersistedChatMessage = {
    id: number;
    role: "user" | "assistant";
    content: string;
    sources: ChatSource[];
    usage?: ChatUsage;
    createdAt: string;
    updatedAt: string;
};

export type ChatConversationDetail = {
    conversation: ChatConversationSummary;
    messages: PersistedChatMessage[];
};

export type ChatSource = {
    id: string;
    docName: string;
    page: number;
    snippet: string;
    /** Ausentes em mensagens gravadas antes de o backend envia-los. */
    documentId?: number;
    nodeId?: string;
    /** Paragrafo da pagina com mais termos da pergunta: o que o canvas grifa. */
    paragraphId?: number;
    /** Indices das linhas do paragrafo que casam a pergunta (grifo fino). */
    lines?: number[];
};

export type ChatStreamEvent =
    | { type: "stage"; stage?: string; message?: string }
    | { type: "delta"; delta: string }
    | { type: "sources"; sources: ChatSource[] }
    | { type: "done"; stage?: string; message?: string; usage?: ChatUsage }
    | { type: "error"; message: string };

type StreamInput = {
    conversationId: number;
    collectionId: number;
    prompt: string;
    history: ChatHistoryMessage[];
    signal?: AbortSignal;
    onEvent: (event: ChatStreamEvent) => void;
};

export async function streamCollectionChat(input: StreamInput): Promise<void> {
    const response = await fetch("/api/chat/stream", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            conversation_id: input.conversationId,
            collection_id: input.collectionId,
            prompt: input.prompt,
            history: input.history,
        }),
        signal: input.signal,
    });

    if (response.status === 401) {
        throw new UnauthorizedRequestError();
    }

    if (!response.ok || !response.body) {
        const payload = await response.json().catch(() => null) as { message?: string } | null;
        throw new Error(payload?.message || apiErrorMessage("chat.start"));
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
        const { value, done } = await reader.read();
        if (done) {
            break;
        }

        buffer += decoder.decode(value, { stream: true });
        const segments = buffer.split("\n\n");
        buffer = segments.pop() ?? "";

        for (const segment of segments) {
            const event = parseSSEEvent(segment);
            if (!event) {
                continue;
            }

            input.onEvent(event);
        }
    }

    const trailing = buffer.trim();
    if (trailing) {
        const event = parseSSEEvent(trailing);
        if (event) {
            input.onEvent(event);
        }
    }
}

export async function fetchChatConversations(
    collectionId: number,
    signal?: AbortSignal
): Promise<ChatConversationSummary[]> {
    const response = await fetch(`/api/chat/conversations?collection_id=${collectionId}`, {
        method: "GET",
        cache: "no-store",
        signal,
    });

    if (response.status === 401) {
        throw new UnauthorizedRequestError();
    }

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("chat.list"));
    }

    const data = hasResponseData(payload) ? payload.data : null;
    if (!Array.isArray(data)) {
        return [];
    }

    return data
        .map(normalizeConversationSummary)
        .filter((conversation): conversation is ChatConversationSummary => conversation !== null);
}

export async function createChatConversation(
    collectionId: number,
    title: string,
    signal?: AbortSignal
): Promise<ChatConversationSummary> {
    const response = await fetch("/api/chat/conversations", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            collection_id: collectionId,
            title,
        }),
        cache: "no-store",
        signal,
    });

    if (response.status === 401) {
        throw new UnauthorizedRequestError();
    }

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("chat.create"));
    }

    const normalized = normalizeConversationSummary(hasResponseData(payload) ? payload.data : null);
    if (!normalized) {
        throw new Error(apiErrorMessage("chat.createInvalid"));
    }

    return normalized;
}

export async function fetchChatConversation(
    conversationId: number,
    signal?: AbortSignal
): Promise<ChatConversationDetail> {
    const response = await fetch(`/api/chat/conversations/${conversationId}`, {
        method: "GET",
        cache: "no-store",
        signal,
    });

    if (response.status === 401) {
        throw new UnauthorizedRequestError();
    }

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("chat.fetch"));
    }

    const normalized = normalizeConversationDetail(hasResponseData(payload) ? payload.data : null);
    if (!normalized) {
        throw new Error(apiErrorMessage("chat.fetchInvalid"));
    }

    return normalized;
}

export async function deleteChatConversation(
    conversationId: number,
    signal?: AbortSignal
): Promise<void> {
    const response = await fetch(`/api/chat/conversations/${conversationId}`, {
        method: "DELETE",
        cache: "no-store",
        signal,
    });

    if (response.status === 401) {
        throw new UnauthorizedRequestError();
    }

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("chat.delete"));
    }
}

export async function updateChatConversationTitle(
    conversationId: number,
    title: string,
    signal?: AbortSignal
): Promise<ChatConversationSummary> {
    const response = await fetch(`/api/chat/conversations/${conversationId}`, {
        method: "PATCH",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ title }),
        cache: "no-store",
        signal,
    });

    if (response.status === 401) {
        throw new UnauthorizedRequestError();
    }

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("chat.update"));
    }

    const normalized = normalizeConversationSummary(hasResponseData(payload) ? payload.data : null);
    if (!normalized) {
        throw new Error(apiErrorMessage("chat.updateInvalid"));
    }

    return normalized;
}

function readEventField<T>(payload: Record<string, unknown>, camelCase: string, pascalCase: string): T | undefined {
    if (payload[camelCase] !== undefined) {
        return payload[camelCase] as T;
    }

    if (payload[pascalCase] !== undefined) {
        return payload[pascalCase] as T;
    }

    return undefined;
}

function parseSSEEvent(rawEvent: string): ChatStreamEvent | null {
    const lines = rawEvent.split("\n");
    let eventName = "message";
    const dataLines: string[] = [];

    for (const rawLine of lines) {
        const line = rawLine.trimEnd();
        if (!line) {
            continue;
        }
        if (line.startsWith("event:")) {
            eventName = line.slice("event:".length).trim();
            continue;
        }
        if (line.startsWith("data:")) {
            dataLines.push(line.slice("data:".length).trim());
        }
    }

    if (dataLines.length === 0) {
        return null;
    }

    const payload = dataLines.join("\n");
    try {
        const parsed = JSON.parse(payload) as Record<string, unknown>;
        const stage = readEventField<string>(parsed, "stage", "Stage");
        const message = readEventField<string>(parsed, "message", "Message");
        const delta = readEventField<string>(parsed, "delta", "Delta");
        const sources = readEventField<ChatSource[]>(parsed, "sources", "Sources");
        const usage = normalizeUsage(readEventField<unknown>(parsed, "usage", "Usage"));

        switch (eventName) {
            case "stage":
                return { type: "stage", stage, message };
            case "delta":
                return typeof delta === "string" ? { type: "delta", delta } : null;
            case "sources":
                return {
                    type: "sources",
                    sources: Array.isArray(sources)
                        ? sources.map(normalizeSource).filter((source): source is ChatSource => source !== null)
                        : [],
                };
            case "done":
                return { type: "done", stage, message, ...(usage ? { usage } : {}) };
            case "error": {
                // A API manda a causa do provedor em `detail` ("monthly usage
                // limit"); sem ela o usuario ve so "falha ao gerar" e nao sabe
                // se o problema e credito, modelo ou rede.
                const rawDetail = readEventField<string>(parsed, "detail", "Detail");
                const detail = typeof rawDetail === "string" ? rawDetail.trim() : "";
                const base = typeof message === "string" ? message : "Erro no chat.";
                return { type: "error", message: detail ? `${base} (${detail})` : base };
            }
            default:
                return null;
        }
    } catch {
        return null;
    }
}

function normalizeConversationSummary(value: unknown): ChatConversationSummary | null {
    if (!value || typeof value !== "object") {
        return null;
    }

    const record = value as Record<string, unknown>;
    const id = Number(record.id);
    const collectionId = Number(record.collection_id);

    if (!Number.isFinite(id) || id <= 0 || !Number.isFinite(collectionId) || collectionId <= 0) {
        return null;
    }

    return {
        id,
        collectionId,
        title: typeof record.title === "string" && record.title.trim() ? record.title.trim() : `Chat ${id}`,
        lastMessageAt: typeof record.last_message_at === "string" ? record.last_message_at : null,
        createdAt: typeof record.created_at === "string" ? record.created_at : new Date(0).toISOString(),
        updatedAt: typeof record.updated_at === "string" ? record.updated_at : new Date(0).toISOString(),
    };
}

function normalizeConversationDetail(value: unknown): ChatConversationDetail | null {
    if (!value || typeof value !== "object") {
        return null;
    }

    const record = value as Record<string, unknown>;
    const conversation = normalizeConversationSummary(record.conversation);
    if (!conversation) {
        return null;
    }

    const messages = Array.isArray(record.messages)
        ? record.messages
            .map(normalizePersistedMessage)
            .filter((message): message is PersistedChatMessage => message !== null)
        : [];

    return {
        conversation,
        messages,
    };
}

function normalizePersistedMessage(value: unknown): PersistedChatMessage | null {
    if (!value || typeof value !== "object") {
        return null;
    }

    const record = value as Record<string, unknown>;
    const id = Number(record.id);
    const role = record.role;
    const content = record.content;

    if (!Number.isFinite(id) || id <= 0 || (role !== "user" && role !== "assistant") || typeof content !== "string") {
        return null;
    }

    return {
        id,
        role,
        content,
        sources: Array.isArray(record.sources)
            ? record.sources
                .map(normalizeSource)
                .filter((source): source is ChatSource => source !== null)
            : [],
        ...((usage) => (usage ? { usage } : {}))(normalizeUsage(record.usage)),
        createdAt: typeof record.created_at === "string" ? record.created_at : new Date(0).toISOString(),
        updatedAt: typeof record.updated_at === "string" ? record.updated_at : new Date(0).toISOString(),
    };
}

function normalizeSource(value: unknown): ChatSource | null {
    if (!value || typeof value !== "object") {
        return null;
    }

    // O stream (dominio) usa camelCase e o historico (DTO) usa snake_case.
    const record = value as Record<string, unknown>;
    const id = typeof record.id === "string" ? record.id : "";
    const docName = typeof record.doc_name === "string" ? record.doc_name : typeof record.docName === "string" ? record.docName : "";
    const page = Number(record.page);
    const snippet = typeof record.snippet === "string" ? record.snippet : "";
    const documentId = Number(record.document_id ?? record.documentId);
    const nodeId = typeof record.node_id === "string" ? record.node_id : typeof record.nodeId === "string" ? record.nodeId : "";
    const paragraphId = Number(record.paragraph_id ?? record.paragraphId);
    const lines = Array.isArray(record.lines) ? record.lines.filter((n): n is number => typeof n === "number" && Number.isInteger(n) && n >= 0) : [];

    if (!id || !docName || !Number.isFinite(page)) {
        return null;
    }

    return {
        id,
        docName,
        page,
        snippet,
        ...(Number.isFinite(documentId) && documentId > 0 ? { documentId } : {}),
        ...(nodeId ? { nodeId } : {}),
        ...(Number.isFinite(paragraphId) && paragraphId > 0 ? { paragraphId } : {}),
        ...(lines.length > 0 ? { lines } : {}),
    };
}

/** Aceita snake_case (historico) e camelCase (stream). Sem tokens, sem bloco. */
export function normalizeUsage(value: unknown): ChatUsage | undefined {
    if (!value || typeof value !== "object") {
        return undefined;
    }
    const record = value as Record<string, unknown>;
    const promptTokens = Number(record.prompt_tokens ?? record.promptTokens);
    const completionTokens = Number(record.completion_tokens ?? record.completionTokens);
    if (!Number.isFinite(promptTokens) || !Number.isFinite(completionTokens)) {
        return undefined;
    }
    const cost = Number(record.estimated_cost_usd ?? record.estimatedCostUsd);
    const providerType = record.provider_type ?? record.providerType;
    const model = record.model;
    return {
        promptTokens,
        completionTokens,
        usageReported: Boolean(record.usage_reported ?? record.usageReported),
        ...(typeof providerType === "string" && providerType ? { providerType } : {}),
        ...(typeof model === "string" && model ? { model } : {}),
        ...(Number.isFinite(cost) && (record.estimated_cost_usd ?? record.estimatedCostUsd) != null ? { estimatedCostUsd: cost } : {}),
    };
}
