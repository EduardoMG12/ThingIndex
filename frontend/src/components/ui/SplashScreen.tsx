"use client";

import { motion, AnimatePresence } from "framer-motion";
import { useEffect, useMemo, useState, useSyncExternalStore } from "react";
import { Logo } from "./Logo";

const SPLASH_STORAGE_KEY = "thingindex_last_splash";
const SHOW_INTERVAL_MS = 3 * 24 * 60 * 60 * 1000; // 3 days
const THEME_EVENT = "thingindex-theme-change";

type SplashPoint = {
    id: number;
    initialX: string;
    initialY: string;
    targetX: [string, string];
    targetY: [string, string];
    duration: number;
};

function createSplashPoints(): SplashPoint[] {
    return Array.from({ length: 15 }, (_, index) => ({
        id: index,
        initialX: `${Math.random() * 100}%`,
        initialY: `${Math.random() * 100}%`,
        targetX: [`${Math.random() * 100}%`, `${Math.random() * 100}%`],
        targetY: [`${Math.random() * 100}%`, `${Math.random() * 100}%`],
        duration: Math.random() * 10 + 10,
    }));
}

function resolveTheme() {
    if (typeof document === "undefined") {
        return "dark";
    }

    return document.documentElement.dataset.theme === "light" ? "light" : "dark";
}

function shouldShowSplash() {
    if (typeof window === "undefined") {
        return false;
    }

    const lastShow = window.localStorage.getItem(SPLASH_STORAGE_KEY);
    const now = Date.now();

    return !lastShow || now - parseInt(lastShow, 10) > SHOW_INTERVAL_MS;
}

function subscribeToClientRender(onStoreChange: () => void) {
    if (typeof window === "undefined") {
        return () => {};
    }

    const frame = window.requestAnimationFrame(onStoreChange);
    return () => window.cancelAnimationFrame(frame);
}

function subscribeToTheme(onStoreChange: () => void) {
    if (typeof window === "undefined") {
        return () => {};
    }

    const observer = new MutationObserver(() => onStoreChange());
    observer.observe(document.documentElement, {
        attributes: true,
        attributeFilter: ["data-theme"],
    });

    const mediaQuery = window.matchMedia("(prefers-color-scheme: light)");
    const handleThemeChange = () => onStoreChange();

    window.addEventListener(THEME_EVENT, handleThemeChange);
    window.addEventListener("storage", handleThemeChange);
    mediaQuery.addEventListener("change", handleThemeChange);

    return () => {
        observer.disconnect();
        window.removeEventListener(THEME_EVENT, handleThemeChange);
        window.removeEventListener("storage", handleThemeChange);
        mediaQuery.removeEventListener("change", handleThemeChange);
    };
}

export const SplashScreen = () => {
    const isHydrated = useSyncExternalStore(subscribeToClientRender, () => true, () => false);
    const theme = useSyncExternalStore(subscribeToTheme, resolveTheme, () => "dark");
    const [isDismissed, setIsDismissed] = useState(false);
    const points = useMemo(() => (isHydrated ? createSplashPoints() : []), [isHydrated]);
    const shouldDisplay = useMemo(() => (isHydrated ? shouldShowSplash() : false), [isHydrated]);
    const isVisible = isHydrated && shouldDisplay && !isDismissed;

    useEffect(() => {
        if (!isVisible) {
            return;
        }

        const timer = setTimeout(() => {
            // localStorage.setItem(SPLASH_STORAGE_KEY, Date.now().toString());
            setIsDismissed(true);
        }, 3500);

        return () => {
            clearTimeout(timer);
        };
    }, [isVisible]);

    return (
        <AnimatePresence>
            {isVisible && (
                <motion.div
                    initial={{ opacity: 1 }}
                    exit={{ opacity: 0, transition: { duration: 0.8, ease: "easeInOut" } }}
                    className="fixed inset-0 z-[9999] flex items-center justify-center overflow-hidden"
                    style={{
                        background:
                            theme === "light"
                                ? "radial-gradient(circle at top left, rgba(255,214,102,0.34), transparent 24%), radial-gradient(circle at top right, rgba(255,157,64,0.18), transparent 26%), #f7f2e8"
                                : "radial-gradient(circle at top left, rgba(245,166,35,0.18), transparent 22%), radial-gradient(circle at top right, rgba(82,168,255,0.12), transparent 24%), #090b11",
                    }}
                >
                    {/* Background Points (Connecting Graph Vibe) */}
                    <div className="absolute inset-0 pointer-events-none opacity-30">
                        {points.map((point) => (
                            <motion.div
                                key={point.id}
                                className="absolute h-1.5 w-1.5 rounded-full"
                                style={{
                                    backgroundColor: theme === "light" ? "rgba(217,119,6,0.85)" : "hsl(var(--primary))",
                                }}
                                initial={{
                                    x: point.initialX,
                                    y: point.initialY,
                                }}
                                animate={{
                                    x: point.targetX,
                                    y: point.targetY,
                                }}
                                transition={{
                                    duration: point.duration,
                                    repeat: Infinity,
                                    ease: "linear",
                                }}
                            />
                        ))}
                    </div>

                    {/* Central Logo and Branding */}
                    <div className="relative flex flex-col items-center">
                        <motion.div
                            initial={{ opacity: 0, scale: 0.92 }}
                            animate={{ opacity: 1, scale: 1 }}
                            transition={{ duration: 0.6 }}
                            className="absolute inset-[-38px] -z-10 rounded-[2rem] blur-2xl"
                            style={{
                                background:
                                    theme === "light"
                                        ? "radial-gradient(circle, rgba(255,208,92,0.36), transparent 66%)"
                                        : "radial-gradient(circle, rgba(245,166,35,0.22), transparent 68%)",
                            }}
                        />
                        <motion.div
                            initial={{ scale: 0.8, opacity: 0 }}
                            animate={{ scale: 1, opacity: 1 }}
                            transition={{ duration: 0.8, ease: [0.22, 1, 0.36, 1] }}
                        >
                            <Logo size={120} className="text-primary" />
                        </motion.div>

                        <motion.div
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 1.2, duration: 0.8 }}
                            className="mt-8 flex flex-col items-center"
                        >
                            <h1
                                className="text-4xl font-bold tracking-tighter"
                                style={{ color: theme === "light" ? "#121826" : "white" }}
                            >
                                thing<span className="text-primary">index</span>
                            </h1>
                            <div className="mt-2 h-0.5 w-12 bg-primary/40 rounded-full" />
                            <p
                                className="mt-4 font-mono text-[10px] uppercase tracking-[0.3em]"
                                style={{ color: theme === "light" ? "rgba(15,23,42,0.55)" : "rgb(113 113 122)" }}
                            >
                                document orchestration · rag · v0.1.0
                            </p>
                        </motion.div>
                    </div>

                    {/* Progress Bar Loader */}
                    <motion.div
                        className="fixed bottom-0 left-0 h-1"
                        style={{
                            background:
                                theme === "light"
                                    ? "linear-gradient(90deg, rgba(217,119,6,0.9), rgba(249,115,22,1))"
                                    : "linear-gradient(90deg, hsl(var(--primary)), rgba(255,196,92,1))",
                        }}
                        initial={{ width: "0%" }}
                        animate={{ width: "100%" }}
                        transition={{ duration: 3, ease: "easeInOut" }}
                    />
                </motion.div>
            )}
        </AnimatePresence>
    );
};
