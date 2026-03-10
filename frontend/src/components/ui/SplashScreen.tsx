"use client";

import { motion, AnimatePresence } from "framer-motion";
import { useEffect, useState } from "react";
import { Logo } from "./Logo";

const SPLASH_STORAGE_KEY = "thingindex_last_splash";
const SHOW_INTERVAL_MS = 3 * 24 * 60 * 60 * 1000; // 3 days

export const SplashScreen = () => {
    const [isVisible, setIsVisible] = useState(false);
    const [isMounted, setIsMounted] = useState(false);

    useEffect(() => {
        setIsMounted(true);
        const lastShow = localStorage.getItem(SPLASH_STORAGE_KEY);
        const now = Date.now();

        if (!lastShow || now - parseInt(lastShow) > SHOW_INTERVAL_MS) {
            setIsVisible(true);
            localStorage.setItem(SPLASH_STORAGE_KEY, now.toString());

            // Auto-hide after animation
            const timer = setTimeout(() => {
                setIsVisible(false);
            }, 3500);
            return () => clearTimeout(timer);
        }
    }, []);

    if (!isMounted) return null;

    return (
        <AnimatePresence>
            {isVisible && (
                <motion.div
                    initial={{ opacity: 1 }}
                    exit={{ opacity: 0, transition: { duration: 0.8, ease: "easeInOut" } }}
                    className="fixed inset-0 z-[9999] flex items-center justify-center bg-zinc-950 overflow-hidden"
                >
                    {/* Background Points (Connecting Graph Vibe) */}
                    <div className="absolute inset-0 pointer-events-none opacity-20">
                        {Array.from({ length: 15 }).map((_, i) => (
                            <motion.div
                                key={i}
                                className="absolute h-1.5 w-1.5 rounded-full bg-primary"
                                initial={{
                                    x: Math.random() * 100 + "%",
                                    y: Math.random() * 100 + "%",
                                }}
                                animate={{
                                    x: [
                                        (Math.random() * 100) + "%",
                                        (Math.random() * 100) + "%",
                                    ],
                                    y: [
                                        (Math.random() * 100) + "%",
                                        (Math.random() * 100) + "%",
                                    ],
                                }}
                                transition={{
                                    duration: Math.random() * 10 + 10,
                                    repeat: Infinity,
                                    ease: "linear",
                                }}
                            />
                        ))}
                    </div>

                    {/* Central Logo and Branding */}
                    <div className="relative flex flex-col items-center">
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
                            <h1 className="text-4xl font-bold tracking-tighter text-white">
                                thing<span className="text-primary">index</span>
                            </h1>
                            <div className="mt-2 h-0.5 w-12 bg-primary/40 rounded-full" />
                            <p className="mt-4 font-mono text-[10px] uppercase tracking-[0.3em] text-zinc-500">
                                document orchestration · rag · v0.1.0
                            </p>
                        </motion.div>
                    </div>

                    {/* Progress Bar Loader */}
                    <motion.div
                        className="fixed bottom-0 left-0 h-1 bg-primary"
                        initial={{ width: "0%" }}
                        animate={{ width: "100%" }}
                        transition={{ duration: 3, ease: "easeInOut" }}
                    />
                </motion.div>
            )}
        </AnimatePresence>
    );
};
