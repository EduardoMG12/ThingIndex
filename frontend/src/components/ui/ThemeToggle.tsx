"use client";

import { useSyncExternalStore } from "react";

import { AnimatePresence, motion } from "framer-motion";
import { Moon, SunMedium } from "lucide-react";

import { cn } from "@/lib/utils";

type ThemeMode = "dark" | "light";

const THEME_EVENT = "thingindex-theme-change";

function subscribeToClientRender(onStoreChange: () => void) {
    if (typeof window === "undefined") {
        return () => { };
    }

    const frame = window.requestAnimationFrame(onStoreChange);
    return () => window.cancelAnimationFrame(frame);
}

function resolveThemeSnapshot(): ThemeMode {
    if (typeof document !== "undefined") {
        const datasetTheme = document.documentElement.dataset.theme;
        if (datasetTheme === "dark" || datasetTheme === "light") {
            return datasetTheme;
        }
    }

    if (typeof window === "undefined") {
        return "dark";
    }

    return window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark";
}

function applyTheme(theme: ThemeMode) {
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;
    window.dispatchEvent(new Event(THEME_EVENT));
}

function subscribeToTheme(onStoreChange: () => void) {
    if (typeof window === "undefined") {
        return () => { };
    }

    const mediaQuery = window.matchMedia("(prefers-color-scheme: light)");
    const observer = new MutationObserver(() => onStoreChange());
    observer.observe(document.documentElement, {
        attributes: true,
        attributeFilter: ["data-theme"],
    });

    const handleThemeChange = () => onStoreChange();

    window.addEventListener(THEME_EVENT, handleThemeChange);
    mediaQuery.addEventListener("change", handleThemeChange);

    return () => {
        observer.disconnect();
        window.removeEventListener(THEME_EVENT, handleThemeChange);
        mediaQuery.removeEventListener("change", handleThemeChange);
    };
}

export function ThemeToggle() {
    const isHydrated = useSyncExternalStore(subscribeToClientRender, () => true, () => false);
    const theme = useSyncExternalStore(subscribeToTheme, resolveThemeSnapshot, () => "dark");

    const nextTheme: ThemeMode = theme === "dark" ? "light" : "dark";

    function handleToggle() {
        applyTheme(nextTheme);
    }

    if (!isHydrated) {
        return (
            <span
                aria-hidden="true"
                className="relative flex h-11 w-[88px] items-center overflow-hidden rounded-full border border-primary/20 bg-gradient-to-r from-slate-950 via-slate-900 to-slate-800 px-1.5 shadow-[0_0_24px_-16px_rgba(245,166,35,0.9)]"
            />
        );
    }

    return (
        <button
            type="button"
            onClick={handleToggle}
            suppressHydrationWarning
            aria-label={`Switch to ${nextTheme} theme`}
            title={`Switch to ${nextTheme} theme`}
            className={cn(
                "group relative flex h-11 w-[88px] items-center overflow-hidden rounded-full border px-1.5 transition-colors duration-500",
                "border-primary/20 bg-gradient-to-r shadow-[0_0_24px_-16px_rgba(245,166,35,0.9)]",
                theme === "dark"
                    ? "from-slate-950 via-slate-900 to-slate-800"
                    : "from-[#ffd977] via-[#ffbf47] to-[#ff8d36]",
                "hover:border-primary/40 hover:shadow-[0_0_36px_-12px_rgba(245,166,35,0.85)]"
            )}
        >
            <motion.span
                aria-hidden="true"
                animate={{
                    opacity: theme === "dark" ? 0.22 : 0.38,
                    scale: theme === "dark" ? 0.9 : 1.15,
                }}
                transition={{ duration: 0.35, ease: "easeOut" }}
                className={cn(
                    "absolute inset-y-0 w-1/2 rounded-full blur-xl",
                    theme === "dark" ? "left-0 bg-sky-300/30" : "right-0 bg-amber-100/70"
                )}
            />

            <div className="relative z-10 flex w-full items-center justify-between px-1.5">
                <motion.span
                    animate={{
                        scale: theme === "dark" ? 1 : 0.82,
                        opacity: theme === "dark" ? 1 : 0.55,
                        rotate: theme === "dark" ? 0 : -24,
                    }}
                    transition={{ duration: 0.3 }}
                    className={cn(
                        "flex h-7 w-7 items-center justify-center rounded-full transition-colors duration-300",
                        theme === "dark" ? "text-slate-100" : "text-slate-600"
                    )}
                >
                    <Moon className="h-4 w-4" />
                </motion.span>
                <motion.span
                    animate={{
                        scale: theme === "light" ? 1 : 0.82,
                        opacity: theme === "light" ? 1 : 0.55,
                        rotate: theme === "light" ? 0 : 24,
                    }}
                    transition={{ duration: 0.3 }}
                    className={cn(
                        "flex h-7 w-7 items-center justify-center rounded-full transition-colors duration-300",
                        theme === "light" ? "text-amber-50" : "text-amber-200/70"
                    )}
                >
                    <SunMedium className="h-4 w-4" />
                </motion.span>
            </div>

            <motion.span
                animate={{ x: theme === "dark" ? 0 : 42, rotate: theme === "dark" ? 0 : 180 }}
                transition={{ type: "spring", stiffness: 320, damping: 22, mass: 0.9 }}
                className={cn(
                    "absolute left-1.5 top-1.5 z-20 flex h-8 w-8 items-center justify-center rounded-full",
                    "bg-[radial-gradient(circle_at_30%_30%,rgba(255,255,255,0.98),rgba(255,236,174,0.92)_42%,rgba(255,174,64,0.88))]",
                    "shadow-[0_12px_30px_-10px_rgba(255,179,71,0.95)] ring-1 ring-white/35"
                )}
            >
                <AnimatePresence mode="wait" initial={false}>
                    <motion.span
                        key={theme}
                        initial={{ opacity: 0, scale: 0.55, rotate: -90 }}
                        animate={{ opacity: 1, scale: 1, rotate: 0 }}
                        exit={{ opacity: 0, scale: 0.55, rotate: 90 }}
                        transition={{ duration: 0.24 }}
                        className="flex items-center justify-center text-slate-950"
                    >
                        {theme === "dark" ? <Moon className="h-4 w-4" /> : <SunMedium className="h-4 w-4" />}
                    </motion.span>
                </AnimatePresence>
            </motion.span>

            <AnimatePresence mode="wait" initial={false}>
                <motion.span
                    key={theme}
                    initial={{ opacity: 0, y: 4 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -4 }}
                    transition={{ duration: 0.22 }}
                    className="pointer-events-none absolute -bottom-6 left-1/2 -translate-x-1/2 text-[9px] font-mono uppercase tracking-[0.28em] text-muted-foreground"
                >
                    {theme}
                </motion.span>
            </AnimatePresence>
        </button>
    );
}
