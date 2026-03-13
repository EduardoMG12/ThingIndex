"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { LanguageSwitcher } from "@/components/ui/LanguageSwitcher";
import { Logo } from "@/components/ui/Logo";
import { useI18n } from "@/lib/i18n/provider";

export const Navbar = () => {
    const { t, href } = useI18n();

    return (
        <header className="sticky top-0 z-50 border-b border-border bg-background/80 backdrop-blur-xl">
            <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-6">
                <Link href={href("/")} className="flex items-center gap-2.5">
                    <Logo className="text-primary" size={28} />
                    <span className="font-display text-base font-bold tracking-tight">
                        thing<span className="text-primary">index</span>
                    </span>
                    <span className="rounded-full border border-border bg-secondary px-2 py-0.5 font-mono text-[9px] uppercase tracking-wider text-muted-foreground">
                        {t("common.beta")}
                    </span>
                </Link>
                <div className="flex items-center gap-3">
                    <LanguageSwitcher />
                    <Link
                        href={href("/documentation")}
                        className="text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
                    >
                        {t("landing.nav.docs")}
                    </Link>
                    <Link href={href("/login")}>
                        <Button variant="ghost" size="sm" className="text-xs h-8">
                            {t("landing.nav.login")}
                        </Button>
                    </Link>
                    <Link href={href("/register")}>
                        <Button size="sm" className="text-xs h-8">
                            {t("landing.nav.start")}
                        </Button>
                    </Link>
                </div>
            </div>
        </header>
    );
};
