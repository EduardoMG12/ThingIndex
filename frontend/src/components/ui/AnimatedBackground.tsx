"use client";

import { motion } from "framer-motion";
import { useEffect, useState } from "react";

export const AnimatedBackground = () => {
    const [mounted, setMounted] = useState(false);

    useEffect(() => {
        setMounted(true);
    }, []);

    if (!mounted) return null;

    // Reduced to 3 blobs for performance, focusing on the top right
    const blobs = Array.from({ length: 3 });

    return (
        <div className="fixed inset-0 -z-10 overflow-hidden pointer-events-none">
            {blobs.map((_, i) => {
                const colors = [
                    "hsl(40 90% 55%)", // Primary Orange
                    "hsl(30 85% 45%)", // Deep Amber
                    "hsl(45 100% 60%)", // Golden Yellow
                ];
                const color = colors[i % colors.length];

                // Positioned on the top right (x: 70-100%, y: -10-30%)
                const startX = (Math.random() * 30 + 70) + "%";
                const startY = (Math.random() * 40 - 10) + "%";

                return (
                    <motion.div
                        key={i}
                        className="absolute rounded-full opacity-[0.25] blur-[150px]"
                        initial={{
                            x: startX,
                            y: startY,
                            scale: 1,
                        }}
                        animate={{
                            x: [
                                (Math.random() * 30 + 70) + "%",
                                (Math.random() * 20 + 85) + "%",
                                (Math.random() * 30 + 70) + "%",
                            ],
                            y: [
                                (Math.random() * 40 - 15) + "%",
                                (Math.random() * 30 - 5) + "%",
                                (Math.random() * 40 - 15) + "%",
                            ],
                            scale: [1, 1.4, 1],
                            rotate: [0, 90, 0],
                        }}
                        transition={{
                            duration: Math.random() * 35 + 35,
                            repeat: Infinity,
                            ease: "easeInOut",
                        }}
                        style={{
                            width: "900px",
                            height: "900px",
                            background: color,
                        }}
                    />
                );
            })}
            <div className="absolute inset-0 bg-background/20 backdrop-blur-[180px]" />
        </div>
    );
};
