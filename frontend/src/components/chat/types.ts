import type { Source } from "@/components/chat/SourceReference";

export type ChatWorkspaceMessage = {
    id: string;
    role: "user" | "assistant";
    content: string;
    sources?: Source[];
    createdAt: string;
};
