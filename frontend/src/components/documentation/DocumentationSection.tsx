"use client";

import { motion } from "framer-motion";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

const documentationSectionAnimation = {
    initial: { opacity: 0, y: 16 },
    animate: { opacity: 1, y: 0 },
};

interface DocumentationSectionProps {
    icon: LucideIcon;
    title: string;
    id: string;
    children: ReactNode;
    delay?: number;
}

export function DocumentationSection({
    icon: Icon,
    title,
    id,
    children,
    delay = 0,
}: DocumentationSectionProps) {
    return (
        <motion.section
            id={id}
            {...documentationSectionAnimation}
            transition={{ delay, duration: 0.5 }}
            className="surface-card scroll-mt-24 overflow-hidden rounded-lg border border-border bg-card"
        >
            <div className="border-b border-border bg-muted/30 px-6 py-4">
                <div className="flex items-center gap-2.5">
                    <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary/10">
                        <Icon className="h-3.5 w-3.5 text-primary" />
                    </div>
                    <h2 className="font-display text-base font-bold text-foreground">{title}</h2>
                </div>
            </div>
            <div className="min-w-0 px-6 py-5 text-sm leading-relaxed text-muted-foreground">{children}</div>
        </motion.section>
    );
}
