"use client";

import { useState } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import { LogIn, Eye, EyeOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { login } from "./actions";
import { Logo } from "@/components/ui/Logo";
import { useI18n } from "@/lib/i18n/provider";

export const LoginForm = () => {
    const [showPassword, setShowPassword] = useState(false);
    const [isLoading, setIsLoading] = useState(false);
    const { t, href } = useI18n();

    const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        setIsLoading(true);
        const formData = new FormData(e.currentTarget);
        const res = await login(formData);
        setIsLoading(false);

        if (res?.error) {
            const type = (res.type as "error" | "success" | "warning" | "info") || "error";
            toast[type](res.error, {
                description: res.errors && res.errors.length > 0
                    ? res.errors.join(", ")
                    : t("auth.login.fallbackError")
            });
        }
    };

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: [0.22, 1, 0.36, 1] }}
            className="w-full max-w-sm"
        >
            {/* Brand */}
            <div className="mb-10 text-center">
                <motion.div
                    className="mx-auto mb-4 flex items-center justify-center"
                    initial={{ scale: 0.8, opacity: 0 }}
                    animate={{ scale: 1, opacity: 1 }}
                    transition={{ delay: 0.1, duration: 0.5 }}
                >
                    <Logo className="text-primary" size={48} />
                </motion.div>
                <h1 className="font-display text-xl font-bold tracking-tight">
                    thing<span className="text-primary">index</span>
                </h1>
                <p className="mt-1 font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
                    {t("auth.login.strapline")}
                </p>
            </div>

            {/* Card */}
            <div className="rounded-xl border border-border bg-secondary/20 p-6 backdrop-blur-sm">
                <form onSubmit={handleSubmit} className="space-y-4" noValidate>
                    <div className="space-y-1.5">
                        <Label htmlFor="email" className="text-[11px] uppercase tracking-wider text-muted-foreground">{t("auth.shared.email")}</Label>
                        <Input
                            id="email"
                            name="email"
                            type="email"
                            placeholder="you@company.com"
                            required
                            className="h-9 bg-background text-sm"
                        />
                    </div>
                    <div className="space-y-1.5">
                        <Label htmlFor="password" className="text-[11px] uppercase tracking-wider text-muted-foreground">{t("auth.login.password")}</Label>
                        <div className="relative">
                            <Input
                                id="password"
                                name="password"
                                type={showPassword ? "text" : "password"}
                                placeholder="••••••••"
                                required
                                className="h-9 bg-background pr-9 text-sm"
                            />
                            <button
                                type="button"
                                onClick={() => setShowPassword(!showPassword)}
                                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                                aria-label={showPassword ? t("auth.login.hidePassword") : t("auth.login.showPassword")}
                            >
                                {showPassword ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                            </button>
                        </div>
                    </div>
                    <Button
                        type="submit"
                        className="h-9 w-full gap-2 text-xs font-semibold"
                        disabled={isLoading}
                    >
                        {isLoading ? (
                            <div className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
                        ) : (
                            <LogIn className="h-3.5 w-3.5" />
                        )}
                        {t("auth.login.submit")}
                    </Button>
                </form>

                <div className="mt-5 flex items-center gap-3">
                    <div className="h-px flex-1 bg-border" />
                    <span className="font-mono text-[9px] uppercase tracking-widest text-muted-foreground">{t("auth.shared.or")}</span>
                    <div className="h-px flex-1 bg-border" />
                </div>

                <p className="mt-4 text-center text-xs text-muted-foreground">
                    {t("auth.login.noAccount")}{" "}
                    <Link href={href("/register")} className="font-medium text-primary hover:underline">
                        {t("auth.login.createAccount")}
                    </Link>
                </p>
            </div>

            <p className="mt-8 text-center font-mono text-[9px] text-muted-foreground/40">
                thingindex v0.1.0 · encrypted · {t("auth.shared.city")}
            </p>
        </motion.div>
    );
};
