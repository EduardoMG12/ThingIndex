"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { TOC } from "./constants";
import { motion, AnimatePresence } from "framer-motion";

export function TableOfContents() {
    const [expandedToc, setExpandedToc] = useState(true);

    return (
        <div className="sticky top-20 surface-elevated rounded-xl border border-border bg-card p-5 shadow-sm">
            <button
                onClick={() => setExpandedToc(!expandedToc)}
                className="flex w-full items-center justify-between text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground mb-4 hover:text-primary transition-colors"
            >
                <span className="flex items-center gap-2">
                    Índice
                </span>
                {expandedToc ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
            </button>
            <AnimatePresence>
                {expandedToc && (
                    <motion.ul
                        initial={{ height: 0, opacity: 0 }}
                        animate={{ height: "auto", opacity: 1 }}
                        exit={{ height: 0, opacity: 0 }}
                        className="space-y-1 overflow-hidden"
                    >
                        {TOC.map((item) => (
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
