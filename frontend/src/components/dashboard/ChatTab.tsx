"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Send, Bot, User, Loader2, Sparkles, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import SourceReference, { type Source } from "@/components/chat/SourceReference";
import SubQueryProgress, { type SubQuery } from "@/components/chat/SubQueryProgress";
import { cn } from "@/lib/utils";

type Message = {
    id: string;
    role: "user" | "assistant";
    content: string;
    sources?: Source[];
    subQueries?: SubQuery[];
};

const MOCK_SOURCES: Source[] = [
    { id: "s1", docName: "petição-inicial.pdf", page: 12, snippet: "A receita líquida apresentou crescimento de 23% no terceiro trimestre." },
    { id: "s2", docName: "contestação.pdf", page: 4, snippet: "Cláusula 7.2 — O prestador deverá entregar relatórios mensais até o 5º dia útil." },
];

const MOCK_SUB_QUERIES: SubQuery[] = [
    { id: "q1", question: "Localizar cláusulas de rescisão", status: "done" },
    { id: "q2", question: "Verificar prazos de entrega", status: "done" },
];

const ChatTab = () => {
    const [messages, setMessages] = useState<Message[]>([]);
    const [input, setInput] = useState("");
    const [isLoading, setIsLoading] = useState(false);
    const [activeSubQueries, setActiveSubQueries] = useState<SubQuery[]>([]);
    const scrollRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        scrollRef.current?.scrollIntoView({ behavior: "smooth" });
    }, [messages, activeSubQueries]);

    const simulateResponse = useCallback(() => {
        setIsLoading(true);
        const subQueries = MOCK_SUB_QUERIES.map(q => ({ ...q, status: "pending" as const }));
        setActiveSubQueries(subQueries);

        setTimeout(() => {
            setActiveSubQueries(prev => prev.map((q, i) => i === 0 ? { ...q, status: "running" } : q));
        }, 800);

        setTimeout(() => {
            setActiveSubQueries(prev => prev.map((q, i) => i === 0 ? { ...q, status: "done" } : i === 1 ? { ...q, status: "running" } : q));
        }, 2000);

        setTimeout(() => {
            setActiveSubQueries([]);
            setMessages(prev => [...prev, {
                id: crypto.randomUUID(),
                role: "assistant",
                content: "Encontrei as informações solicitadas. Na **Petição Inicial**, a fundamentação se baseia nos atrasos sistemáticos. Na **Contestação**, a defesa foca na força maior.\n\nRecomendo revisar a cláusula 7.2 do contrato.",
                sources: MOCK_SOURCES,
            }]);
            setIsLoading(false);
        }, 4000);
    }, []);

    const handleSend = () => {
        if (!input.trim() || isLoading) return;
        setMessages(prev => [...prev, { id: crypto.randomUUID(), role: "user", content: input }]);
        setInput("");
        simulateResponse();
    };

    return (
        <div className="surface-elevated flex h-[600px] flex-col rounded-lg overflow-hidden border border-border shadow-lg">
            <ScrollArea className="flex-1 p-6 bg-background/30">
                <div className="space-y-6">
                    {messages.length === 0 && (
                        <div className="flex h-96 flex-col items-center justify-center text-center opacity-40">
                            <Sparkles className="mb-4 h-12 w-12 text-primary" />
                            <h3 className="text-sm font-semibold">Inicie uma Consulta Contextual</h3>
                            <p className="mt-2 max-w-xs text-xs leading-relaxed">
                                Nossa IA ira decompor sua pergunta e reconstruir o contexto da collection a partir das tabelas relacionais.
                            </p>
                        </div>
                    )}

                    <AnimatePresence>
                        {messages.map((msg) => (
                            <motion.div
                                key={msg.id}
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                className={cn("flex gap-4", msg.role === "user" ? "flex-row-reverse" : "flex-row")}
                            >
                                <div className={cn(
                                    "flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border",
                                    msg.role === "user" ? "bg-primary border-primary text-primary-foreground" : "bg-card border-border text-primary"
                                )}>
                                    {msg.role === "user" ? <User className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
                                </div>

                                <div className={cn(
                                    "max-w-[85%] space-y-2",
                                    msg.role === "user" ? "items-end text-right" : "items-start text-left"
                                )}>
                                    <div className={cn(
                                        "rounded-2xl px-4 py-2.5 text-sm leading-relaxed shadow-sm transition-all",
                                        msg.role === "user"
                                            ? "bg-primary text-primary-foreground rounded-tr-none"
                                            : "bg-card border border-border rounded-tl-none text-foreground"
                                    )}>
                                        {msg.content.split(/(\*\*.*?\*\*)/).map((part, i) =>
                                            part.startsWith("**") ? <strong key={i} className="font-bold">{part.slice(2, -2)}</strong> : part
                                        )}
                                    </div>

                                    {msg.sources && msg.sources.length > 0 && (
                                        <div className="flex flex-wrap gap-2 pt-1">
                                            {msg.sources.map(s => <SourceReference key={s.id} source={s} />)}
                                        </div>
                                    )}
                                </div>
                            </motion.div>
                        ))}
                    </AnimatePresence>

                    {activeSubQueries.length > 0 && (
                        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
                            <SubQueryProgress queries={activeSubQueries} />
                        </motion.div>
                    )}

                    <div ref={scrollRef} />
                </div>
            </ScrollArea>

            <div className="border-t border-border p-4 bg-muted/20">
                <div className="relative flex items-end gap-2 pr-12">
                    <Textarea
                        placeholder="Pergunte sobre os processos indexados..."
                        value={input}
                        onChange={(e) => setInput(e.target.value)}
                        onKeyDown={(e) => {
                            if (e.key === "Enter" && !e.shiftKey) {
                                e.preventDefault();
                                handleSend();
                            }
                        }}
                        className="min-h-[44px] max-h-32 resize-none bg-background py-3 text-sm focus:ring-1 focus:ring-primary/30"
                        rows={1}
                    />
                    <Button
                        onClick={handleSend}
                        disabled={isLoading || !input.trim()}
                        size="icon"
                        className="absolute right-0 bottom-0.5 h-10 w-10 transition-transform active:scale-95 shadow-gold"
                    >
                        {isLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
                    </Button>
                </div>
                <div className="mt-2 flex items-center justify-between">
                    <p className="text-[10px] text-muted-foreground font-mono flex items-center gap-1.5 uppercase">
                        <AlertCircle className="h-3 w-3" /> IA pode alucinar · Verifique as fontes
                    </p>
                    <p className="text-[9px] text-muted-foreground font-mono uppercase tracking-widest">
                        Shift + Enter para nova linha
                    </p>
                </div>
            </div>
        </div>
    );
};

export default ChatTab;
