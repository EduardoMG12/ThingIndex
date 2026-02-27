"use client";

import { motion } from "framer-motion";
import { Bot, User } from "lucide-react";

export type Message = {
    role: "user" | "assistant";
    content: string;
};

type Props = {
    messages: Message[];
};

export default function ResponsePanel({ messages }: Props) {
    if (messages.length === 0) return null;

    return (
        <div className="surface-card rounded-lg p-5">
            <h3 className="mb-4 text-sm font-semibold">Respostas</h3>
            <div className="space-y-4 max-h-[400px] overflow-y-auto pr-1 custom-scrollbar">
                {messages.map((msg, i) => (
                    <motion.div
                        key={i}
                        initial={{ opacity: 0, y: 8 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: i * 0.1 }}
                        className="flex gap-3"
                    >
                        <div
                            className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-md ${msg.role === "user" ? "bg-secondary" : "bg-primary/20"
                                }`}
                        >
                            {msg.role === "user" ? (
                                <User className="h-3.5 w-3.5 text-secondary-foreground" />
                            ) : (
                                <Bot className="h-3.5 w-3.5 text-primary" />
                            )}
                        </div>
                        <div className="min-w-0 flex-1">
                            <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                                {msg.role === "user" ? "Você" : "Assistente"}
                            </p>
                            <div className="whitespace-pre-wrap text-sm leading-relaxed text-foreground">
                                {msg.content}
                            </div>
                        </div>
                    </motion.div>
                ))}
            </div>
        </div>
    );
}
