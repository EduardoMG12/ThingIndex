"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";
import { motion } from "framer-motion";
import { Home, AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n/provider";

export const NotFoundContent = () => {
    const pathname = usePathname();
    const { href, t } = useI18n();

    useEffect(() => {
        console.error("404 Error: Usuário tentou acessar rota inexistente:", pathname);
    }, [pathname]);

    return (
        <div className="relative flex min-h-screen items-center justify-center bg-background bg-dots p-4">
            <motion.div
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
                className="w-full max-w-md text-center"
            >
                <motion.div
                    initial={{ y: 20, opacity: 0 }}
                    animate={{ y: 0, opacity: 1 }}
                    transition={{ delay: 0.1 }}
                    className="mb-8 flex justify-center"
                >
                    <div className="relative">
                        <div className="absolute -inset-4 rounded-full bg-primary/20 blur-xl animate-pulse" />
                        <div className="relative flex h-24 w-24 items-center justify-center rounded-2xl border border-primary/20 bg-secondary/20 backdrop-blur-sm">
                            <AlertTriangle className="h-12 w-12 text-primary" />
                        </div>
                    </div>
                </motion.div>

                <motion.h1
                    initial={{ y: 20, opacity: 0 }}
                    animate={{ y: 0, opacity: 1 }}
                    transition={{ delay: 0.2 }}
                    className="font-display text-7xl font-bold tracking-tighter text-foreground"
                >
                    404
                </motion.h1>

                <motion.div
                    initial={{ y: 20, opacity: 0 }}
                    animate={{ y: 0, opacity: 1 }}
                    transition={{ delay: 0.3 }}
                >
                    <h2 className="mt-4 text-xl font-semibold text-foreground">
                        {t("notFound.title")}
                    </h2>
                    <p className="mt-2 text-muted-foreground">
                        {t("notFound.routePrefix")}{" "}
                        <code className="rounded bg-muted px-1 py-0.5 font-mono text-sm text-primary">{pathname}</code>{" "}
                        {t("notFound.routeSuffix")}
                    </p>
                </motion.div>

                <motion.div
                    initial={{ y: 20, opacity: 0 }}
                    animate={{ y: 0, opacity: 1 }}
                    transition={{ delay: 0.4 }}
                    className="mt-10"
                >
                    <Link href={href("/workspace")} className="inline-block">
                        <Button variant="primary" size="lg" className="gap-2 px-8">
                            <Home className="h-4 w-4 text-primary-foreground" />
                            <span>{t("notFound.backHome")}</span>
                        </Button>
                    </Link>
                </motion.div>

                <motion.p
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 0.4 }}
                    transition={{ delay: 0.6 }}
                    className="mt-12 font-mono text-[10px] uppercase tracking-widest text-muted-foreground"
                >
                    thingindex v0.1.0 · sao paulo · brazil
                </motion.p>
            </motion.div>
        </div>
    );
};
