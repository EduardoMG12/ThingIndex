"use client";

import { motion } from "framer-motion";

interface LogoProps {
    className?: string;
    size?: number;
}

export const Logo = ({ className, size = 32 }: LogoProps) => {
    // Points for the "T" shape:
    // 1: Top Left, 2: Top Center, 3: Top Right, 4: Bottom Center
    const points = [
        { id: 1, x: 20, y: 25 }, // Top-left
        { id: 2, x: 50, y: 25 }, // Top-center (Stem start)
        { id: 3, x: 80, y: 25 }, // Top-right
        { id: 4, x: 50, y: 75 }, // Bottom-center (Stem end)
    ];

    const connections = [
        { from: 1, to: 2 },
        { from: 2, to: 3 },
        { from: 2, to: 4 },
    ];

    return (
        <svg
            width={size}
            height={size}
            viewBox="0 0 100 100"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            className={className}
        >
            {/* Connection Lines */}
            {connections.map((conn, i) => {
                const p1 = points.find(p => p.id === conn.from)!;
                const p2 = points.find(p => p.id === conn.to)!;
                return (
                    <motion.line
                        key={`line-${i}`}
                        x1={p1.x}
                        y1={p1.y}
                        x2={p2.x}
                        y2={p2.y}
                        stroke="currentColor"
                        strokeWidth="6"
                        strokeLinecap="round"
                        strokeOpacity="0.3"
                        initial={{ pathLength: 0, opacity: 0 }}
                        animate={{ pathLength: 1, opacity: 1 }}
                        transition={{
                            delay: 0.5 + i * 0.2,
                            duration: 0.4,
                            ease: "easeInOut"
                        }}
                    />
                );
            })}

            {/* Nodes */}
            {points.map((p, i) => (
                <motion.circle
                    key={`node-${i}`}
                    cx={p.x}
                    cy={p.y}
                    r={p.id === 2 ? "6" : "5"}
                    stroke="currentColor"
                    strokeWidth="2.5"
                    fill={p.id === 2 ? "transparent" : "currentColor"}
                    initial={{ scale: 0, opacity: 0 }}
                    animate={{ scale: 1, opacity: 1 }}
                    transition={{
                        delay: i * 0.15,
                        duration: 0.3,
                        type: "spring",
                        stiffness: 260,
                        damping: 20
                    }}
                />
            ))}
        </svg>
    );
};
