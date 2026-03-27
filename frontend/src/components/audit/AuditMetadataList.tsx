"use client";

import { getAuditMetadataEntries } from "@/lib/audit";

export default function AuditMetadataList({
    metadata,
}: {
    metadata: Record<string, unknown>;
}) {
    const entries = getAuditMetadataEntries(metadata);

    if (entries.length === 0) {
        return null;
    }

    return (
        <div className="mt-3 flex flex-wrap gap-2">
            {entries.map((entry) => (
                <span
                    key={entry.key}
                    className="rounded-full border border-border px-2 py-1 text-[10px] font-mono text-muted-foreground"
                >
                    {entry.key}: {entry.value}
                </span>
            ))}
        </div>
    );
}
