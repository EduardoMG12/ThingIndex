"use client";

import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, ChevronRight } from "lucide-react";
import { useState } from "react";

import { DOCUMENTATION_TABLE_OF_CONTENTS } from "@/components/documentation/constants";

export function TableOfContents() {
    const [isExpanded, setIsExpanded] = useState(true);

    return (
        <div className="sticky top-20 surface-elevated rounded-xl border border-border bg-card p-5 shadow-sm">
            <button
                type="button"
                onClick={() => setIsExpanded((current) => !current)}
                className="mb-4 flex w-full items-center justify-between text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground transition-colors hover:text-primary"
            >
                <span className="flex items-center gap-2">Navegação</span>
                {isExpanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
            </button>
            <AnimatePresence>
                {isExpanded && (
                    <motion.ul
                        initial={{ height: 0, opacity: 0 }}
                        animate={{ height: "auto", opacity: 1 }}
                        exit={{ height: 0, opacity: 0 }}
                        className="space-y-1 overflow-hidden"
                    >
                        {DOCUMENTATION_TABLE_OF_CONTENTS.map((item) => (
                            <li key={item.id}>
                                <a
                                    href={`#${item.id}`}
                                    className="block rounded-lg px-3 py-2 text-[11px] font-medium text-muted-foreground transition-all hover:bg-primary/5 hover:text-primary active:scale-[0.98]"
                                >
                                    {item.label}
                                </a>
                            </li>
                        ))}
                    </motion.ul>
                )}
            </AnimatePresence>
        </div>
    );
}
