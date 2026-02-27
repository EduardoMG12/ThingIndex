"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { motion } from "framer-motion";
import { Network, FileText, Sparkles } from "lucide-react";
import { cn } from "@/lib/utils";

export default function Header() {
    const pathname = usePathname();

    const links = [
        { href: "/", label: "Workspace", icon: Sparkles },
        { href: "/monitoring", label: "Monitoramento", icon: Network },
        { href: "/docs", label: "Documentação", icon: FileText },
    ];

    return (
        <header className="sticky top-0 z-50 border-b border-border bg-background/80 backdrop-blur-xl">
            <div className="container flex h-16 items-center justify-between">
                <Link href="/" className="flex items-center gap-2.5">
                    <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary shadow-gold">
                        <Network className="h-5 w-5 text-primary-foreground" />
                    </div>
                    <span className="text-lg font-bold text-gradient-gold">GraphRAG</span>
                </Link>

                <nav className="flex items-center gap-1">
                    {links.map(({ href, label, icon: Icon }) => {
                        const isActive = pathname === href;
                        return (
                            <Link
                                key={href}
                                href={href}
                                className="relative px-4 py-2 text-sm font-medium transition-colors"
                            >
                                <span className={cn(
                                    "flex items-center gap-2 transition-colors",
                                    isActive ? "text-primary" : "text-muted-foreground hover:text-foreground"
                                )}>
                                    <Icon className="h-4 w-4" />
                                    {label}
                                </span>
                                {isActive && (
                                    <motion.div
                                        layoutId="nav-indicator"
                                        className="absolute inset-x-2 -bottom-[1.05rem] h-0.5 rounded-full bg-primary"
                                    />
                                )}
                            </Link>
                        );
                    })}
                </nav>
            </div>
        </header>
    );
}
