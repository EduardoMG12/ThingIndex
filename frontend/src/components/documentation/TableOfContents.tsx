"use client";

import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, ChevronRight } from "lucide-react";
import { useEffect, useState } from "react";

import type { DocumentationTocItem } from "@/components/documentation/page-content";
import { useI18n } from "@/lib/i18n/provider";
import { cn } from "@/lib/utils";

export function TableOfContents({
    items,
}: {
    items: DocumentationTocItem[];
}) {
    const [isExpanded, setIsExpanded] = useState(true);
    const [activeId, setActiveId] = useState(items[0]?.id ?? "");
    const { locale } = useI18n();
    const navigationLabel = locale === "pt-BR"
        ? "Navegação"
        : locale === "es"
            ? "Navegación"
            : "Navigation";

    useEffect(() => {
        if (items.length === 0) {
            return;
        }

        const sections = items
            .map((item) => document.getElementById(item.id))
            .filter((section): section is HTMLElement => section instanceof HTMLElement);

        if (sections.length === 0) {
            return;
        }

        const scrollContainer = document.getElementById("main-content");
        const activationOffset = scrollContainer ? 120 : 160;
        let frameId: number | null = null;

        const updateActiveSection = () => {
            const containerTop = scrollContainer?.getBoundingClientRect().top ?? 0;
            let nextActiveId = sections[0]?.id ?? items[0]?.id ?? "";

            for (const section of sections) {
                const relativeTop = section.getBoundingClientRect().top - containerTop;
                if (relativeTop <= activationOffset) {
                    nextActiveId = section.id;
                    continue;
                }
                break;
            }

            setActiveId((current) => current === nextActiveId ? current : nextActiveId);
        };

        const scheduleActiveSectionUpdate = () => {
            if (frameId !== null) {
                window.cancelAnimationFrame(frameId);
            }

            frameId = window.requestAnimationFrame(() => {
                frameId = null;
                updateActiveSection();
            });
        };

        const scrollTarget: HTMLElement | Window = scrollContainer ?? window;
        scheduleActiveSectionUpdate();

        scrollTarget.addEventListener("scroll", updateActiveSection, { passive: true });
        window.addEventListener("hashchange", scheduleActiveSectionUpdate);
        window.addEventListener("resize", scheduleActiveSectionUpdate);

        return () => {
            if (frameId !== null) {
                window.cancelAnimationFrame(frameId);
            }

            scrollTarget.removeEventListener("scroll", updateActiveSection);
            window.removeEventListener("hashchange", scheduleActiveSectionUpdate);
            window.removeEventListener("resize", scheduleActiveSectionUpdate);
        };
    }, [items]);

    return (
        <div className="sticky top-20 surface-elevated rounded-xl border border-border bg-card p-5 shadow-sm">
            <button
                type="button"
                onClick={() => setIsExpanded((current) => !current)}
                className="mb-4 flex w-full items-center justify-between text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground transition-colors hover:text-primary"
            >
                <span className="flex items-center gap-2">{navigationLabel}</span>
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
                        {items.map((item) => {
                            const isActive = item.id === activeId;

                            return (
                                <li key={item.id}>
                                    <a
                                        href={`#${item.id}`}
                                        aria-current={isActive ? "location" : undefined}
                                        onClick={() => setActiveId(item.id)}
                                        className={cn(
                                            "flex items-center gap-2 rounded-xl border px-3 py-2 text-[11px] font-medium transition-all active:scale-[0.98]",
                                            isActive
                                                ? "border-primary/20 bg-primary/10 text-foreground shadow-[0_10px_30px_-24px_rgba(245,166,35,0.95)]"
                                                : "border-transparent text-muted-foreground hover:bg-primary/5 hover:text-primary"
                                        )}
                                    >
                                        <span
                                            className={cn(
                                                "h-1.5 w-1.5 rounded-full transition-colors",
                                                isActive ? "bg-primary" : "bg-border"
                                            )}
                                        />
                                        {item.label}
                                    </a>
                                </li>
                            );
                        })}
                    </motion.ul>
                )}
            </AnimatePresence>
        </div>
    );
}
