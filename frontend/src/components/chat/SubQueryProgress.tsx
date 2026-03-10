"use client";

import { motion } from "framer-motion";
import { CheckCircle2, Circle, Loader2, Search } from "lucide-react";
import { cn } from "@/lib/utils";

export type SubQuery = {
    id: string;
    question: string;
    status: "pending" | "running" | "done";
};

interface SubQueryProgressProps {
    queries: SubQuery[];
}

const SubQueryProgress = ({ queries }: SubQueryProgressProps) => {
    return (
        <div className="space-y-2 rounded-lg border border-border bg-muted/30 p-4">
            <div className="flex items-center gap-2 mb-3">
                <Search className="h-3.5 w-3.5 text-primary" />
                <h4 className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground">Decomposição de Query</h4>
            </div>

            <div className="space-y-3">
                {queries.map((q) => (
                    <div key={q.id} className="flex items-start gap-3">
                        <div className="mt-0.5">
                            {q.status === "done" ? (
                                <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />
                            ) : q.status === "running" ? (
                                <Loader2 className="h-3.5 w-3.5 text-primary animate-spin" />
                            ) : (
                                <Circle className="h-3.5 w-3.5 text-muted-foreground/30" />
                            )}
                        </div>
                        <div className="flex-1 min-w-0">
                            <p className={cn(
                                "text-[11px] leading-tight transition-colors",
                                q.status === "done" ? "text-muted-foreground" : "text-foreground font-medium"
                            )}>
                                {q.question}
                            </p>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default SubQueryProgress;
