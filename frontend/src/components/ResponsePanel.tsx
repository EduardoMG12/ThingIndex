"use client";

import * as React from "react";
import { Bot, User } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/components/ui/Card";
import { cn } from "@/lib/utils";

export type Message = {
    role: "user" | "assistant";
    content: string;
};

interface ResponsePanelProps {
    messages: Message[];
}

export default function ResponsePanel({ messages }: ResponsePanelProps) {
    const scrollRef = React.useRef<HTMLDivElement>(null);

    React.useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
        }
    }, [messages]);

    return (
        <Card className="flex-1 flex flex-col min-h-[400px]">
            <CardHeader>
                <h3 className="text-lg font-semibold">Respostas</h3>
            </CardHeader>
            <CardContent className="flex-1 overflow-y-auto max-h-[500px] space-y-4" ref={scrollRef}>
                {messages.length === 0 ? (
                    <div className="flex flex-col items-center justify-center h-full text-muted-foreground py-12">
                        <Bot className="w-12 h-12 mb-4 opacity-20" />
                        <p className="text-sm">Aguardando sua primeira pergunta...</p>
                    </div>
                ) : (
                    messages.map((msg, i) => (
                        <div
                            key={i}
                            className={cn(
                                "flex gap-3 p-4 rounded-lg",
                                msg.role === "assistant" ? "bg-secondary/30 ring-1 ring-border" : "bg-primary/5 ml-8"
                            )}
                        >
                            <div className={cn(
                                "w-8 h-8 rounded-full flex items-center justify-center shrink-0",
                                msg.role === "assistant" ? "bg-primary text-primary-foreground" : "bg-secondary text-foreground"
                            )}>
                                {msg.role === "assistant" ? <Bot className="w-5 h-5" /> : <User className="w-5 h-5" />}
                            </div>
                            <div className="space-y-1">
                                <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                                    {msg.role === "assistant" ? "Assistant" : "Você"}
                                </p>
                                <p className="text-sm leading-relaxed">{msg.content}</p>
                            </div>
                        </div>
                    ))
                )}
            </CardContent>
        </Card>
    );
}
