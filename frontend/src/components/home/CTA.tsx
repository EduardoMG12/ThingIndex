"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { ArrowRight, GitBranch } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n/provider";

const fadeUp = {
    initial: { opacity: 0, y: 24 },
    whileInView: { opacity: 1, y: 0 },
    viewport: { once: true },
};

export const CTA = () => {
    const { t, href } = useI18n();

    return (
        <section className="mx-auto max-w-6xl px-6 py-20 text-center">
            <motion.div {...fadeUp} transition={{ duration: 0.6 }}>
                <h2 className="font-display text-2xl font-bold sm:text-3xl">
                    {t("landing.cta.title")}
                </h2>
                <p className="mx-auto mt-3 max-w-md text-sm text-muted-foreground">
                    {t("landing.cta.description")}
                </p>
                <div className="mt-6 flex items-center justify-center gap-3">
                    <Link href={href("/register")}>
                        <Button size="lg" className="gap-2 text-sm font-semibold rounded-full px-8">
                            {t("landing.cta.primary")}
                            <ArrowRight className="h-4 w-4" />
                        </Button>
                    </Link>
                    <a
                        href="https://github.com/thingindex"
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        <Button variant="outline" size="lg" className="gap-2 text-sm rounded-full px-8">
                            <GitBranch className="h-4 w-4" />
                            GitHub
                        </Button>
                    </a>
                </div>
            </motion.div>
        </section>
    );
};
