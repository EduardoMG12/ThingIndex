import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface DocumentationStatusMessageProps {
    tone?: "loading" | "error";
    children: ReactNode;
}

export function DocumentationStatusMessage({
    tone = "loading",
    children,
}: DocumentationStatusMessageProps) {
    return (
        <div
            className={cn(
                "rounded-xl border p-5 text-sm",
                tone === "error"
                    ? "border-destructive/30 bg-destructive/10 text-destructive"
                    : "border-border bg-background/50 text-muted-foreground"
            )}
        >
            {children}
        </div>
    );
}
