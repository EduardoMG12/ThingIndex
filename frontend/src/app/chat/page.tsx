"use client";

import { Suspense } from "react";

import ChatTab from "@/components/dashboard/ChatTab";
import AppShell from "@/components/layout/AppShell";

export default function ChatPage() {
    return (
        <AppShell contentClassName="max-w-none p-3 sm:p-4 lg:p-5">
            <Suspense fallback={null}>
                <ChatTab className="h-[calc(100vh-6.75rem)] min-h-[820px]" />
            </Suspense>
        </AppShell>
    );
}
