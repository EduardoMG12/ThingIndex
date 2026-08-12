"use client";

import { Suspense } from "react";

import ChatTab from "@/components/dashboard/ChatTab";
import AppShell from "@/components/layout/AppShell";

export default function ChatPage() {
    return (
        <AppShell contentClassName="max-w-none p-0">
            <Suspense fallback={null}>
                <ChatTab className="h-[calc(100vh-4.5rem)] min-h-[720px]" />
            </Suspense>
        </AppShell>
    );
}
