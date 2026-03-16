import { UnauthorizedRequestError } from "@/lib/api/fetch-with-auth-redirect";

export type ChatHistoryMessage = {
    role: "user" | "assistant";
    content: string;
};

export type ChatSource = {
    id: string;
    docName: string;
    page: number;
    snippet: string;
};

export type ChatStreamEvent =
    | { type: "stage"; stage?: string; message?: string }
    | { type: "delta"; delta: string }
    | { type: "sources"; sources: ChatSource[] }
    | { type: "done"; stage?: string; message?: string }
    | { type: "error"; message: string };

type StreamInput = {
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
        throw new Error(payload?.message || "Nao foi possivel iniciar o chat.");
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
        const parsed = JSON.parse(payload) as {
            type?: string;
            stage?: string;
            message?: string;
            delta?: string;
            sources?: ChatSource[];
        };

        switch (eventName) {
            case "stage":
                return { type: "stage", stage: parsed.stage, message: parsed.message };
            case "delta":
                return { type: "delta", delta: typeof parsed.delta === "string" ? parsed.delta : "" };
            case "sources":
                return { type: "sources", sources: Array.isArray(parsed.sources) ? parsed.sources : [] };
            case "done":
                return { type: "done", stage: parsed.stage, message: parsed.message };
            case "error":
                return { type: "error", message: typeof parsed.message === "string" ? parsed.message : "Erro no chat." };
            default:
                return null;
        }
    } catch {
        return null;
    }
}
