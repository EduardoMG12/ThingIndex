"use client";

import { motion } from "framer-motion";
import { LucideIcon } from "lucide-react";

const fadeUp = { initial: { opacity: 0, y: 16 }, animate: { opacity: 1, y: 0 } };

interface SectionProps {
    icon: LucideIcon;
    title: string;
    id: string;
    children: React.ReactNode;
    delay?: number;
}

export function Section({ icon: Icon, title, id, children, delay = 0 }: SectionProps) {
    return (
        <motion.section
            id={id}
            {...fadeUp}
            transition={{ delay, duration: 0.5 }}
            className="surface-card rounded-lg overflow-hidden border border-border bg-card"
        >
            <div className="border-b border-border px-6 py-4 bg-muted/30">
                <div className="flex items-center gap-2.5">
                    <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary/10">
                        <Icon className="h-3.5 w-3.5 text-primary" />
                    </div>
                    <h2 className="font-display text-base font-bold text-foreground">{title}</h2>
                </div>
            </div>
            <div className="px-6 py-5 text-sm leading-relaxed text-muted-foreground">
                {children}
            </div>
        </motion.section>
    );
}
