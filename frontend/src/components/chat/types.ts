import type { Source } from "@/components/chat/SourceReference";
import type { ChatUsage } from "@/lib/api/chat";

export type ChatWorkspaceMessage = {
    id: string;
    role: "user" | "assistant";
    content: string;
    sources?: Source[];
    usage?: ChatUsage;
    createdAt: string;
    /** "error": a resposta falhou (provedor fora, limite); errorDetail e o motivo. */
    status?: "error";
    errorDetail?: string;
};
