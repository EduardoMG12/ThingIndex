"use client";

import { motion } from "framer-motion";
import { useMemo, useSyncExternalStore } from "react";

type BackgroundBlob = {
    color: string;
    initialX: string;
    initialY: string;
    targetX: [string, string, string];
    targetY: [string, string, string];
    duration: number;
};

function subscribeToClientRender(onStoreChange: () => void) {
    if (typeof window === "undefined") {
        return () => {};
    }

    const frame = window.requestAnimationFrame(onStoreChange);
    return () => window.cancelAnimationFrame(frame);
}

function createBlobs(): BackgroundBlob[] {
    const colors = [
        "hsl(40 90% 55%)",
        "hsl(30 85% 45%)",
        "hsl(45 100% 60%)",
    ];

    return Array.from({ length: 3 }, (_, index) => ({
        color: colors[index % colors.length],
        initialX: `${Math.random() * 30 + 70}%`,
        initialY: `${Math.random() * 40 - 10}%`,
        targetX: [
            `${Math.random() * 30 + 70}%`,
            `${Math.random() * 20 + 85}%`,
            `${Math.random() * 30 + 70}%`,
        ],
        targetY: [
            `${Math.random() * 40 - 15}%`,
            `${Math.random() * 30 - 5}%`,
            `${Math.random() * 40 - 15}%`,
        ],
        duration: Math.random() * 35 + 35,
    }));
}

export const AnimatedBackground = () => {
    const isHydrated = useSyncExternalStore(subscribeToClientRender, () => true, () => false);
    const blobs = useMemo(() => (isHydrated ? createBlobs() : []), [isHydrated]);

    if (!isHydrated) return null;

    return (
        <div className="fixed inset-0 -z-10 overflow-hidden pointer-events-none">
            {blobs.map((blob, i) => {
                return (
                    <motion.div
                        key={i}
                        className="absolute rounded-full opacity-[0.25] blur-[150px]"
                        initial={{
                            x: blob.initialX,
                            y: blob.initialY,
                            scale: 1,
                        }}
                        animate={{
                            x: blob.targetX,
                            y: blob.targetY,
                            scale: [1, 1.4, 1],
                            rotate: [0, 90, 0],
                        }}
                        transition={{
                            duration: blob.duration,
                            repeat: Infinity,
                            ease: "easeInOut",
                        }}
                        style={{
                            width: "900px",
                            height: "900px",
                            background: blob.color,
                        }}
                    />
                );
            })}
            <div className="absolute inset-0 bg-background/20 backdrop-blur-[180px]" />
        </div>
    );
};
