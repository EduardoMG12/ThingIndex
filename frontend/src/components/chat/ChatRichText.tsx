"use client";

import type { ReactNode } from "react";

function renderInline(text: string, keyPrefix: string): ReactNode[] {
    return text
        .split(/(\*\*[^*]+\*\*)/g)
        .filter(Boolean)
        .map((segment, index) => {
            if (segment.startsWith("**") && segment.endsWith("**") && segment.length > 4) {
                return (
                    <strong key={`${keyPrefix}-strong-${index}`} className="font-semibold text-foreground">
                        {segment.slice(2, -2)}
                    </strong>
                );
            }

            return <span key={`${keyPrefix}-text-${index}`}>{segment}</span>;
        });
}

function renderParagraphLines(lines: string[], paragraphIndex: number) {
    return lines.map((line, lineIndex) => (
        <span key={`paragraph-${paragraphIndex}-line-${lineIndex}`}>
            {lineIndex > 0 ? <br /> : null}
            {renderInline(line, `paragraph-${paragraphIndex}-line-${lineIndex}`)}
        </span>
    ));
}

export function ChatRichText({ content }: { content: string }) {
    const normalizedContent = content.replace(/\r\n/g, "\n").trim();
    if (!normalizedContent) {
        return null;
    }

    return normalizedContent.split(/\n{2,}/).map((paragraph, paragraphIndex) => {
        const lines = paragraph.split("\n").map((line) => line.trimEnd());

        if (lines.length === 1 && /^#{1,3}\s+/.test(lines[0])) {
            return (
                <h4 key={`paragraph-${paragraphIndex}`} className="text-base font-semibold leading-7 text-foreground">
                    {renderInline(lines[0].replace(/^#{1,3}\s+/, ""), `heading-${paragraphIndex}`)}
                </h4>
            );
        }

        const bulletLines = lines.filter((line) => /^[-*]\s+/.test(line));
        if (bulletLines.length === lines.length && bulletLines.length > 0) {
            return (
                <ul key={`paragraph-${paragraphIndex}`} className="list-disc space-y-2 pl-5 leading-7">
                    {bulletLines.map((line, lineIndex) => (
                        <li key={`paragraph-${paragraphIndex}-item-${lineIndex}`}>
                            {renderInline(line.replace(/^[-*]\s+/, ""), `bullet-${paragraphIndex}-${lineIndex}`)}
                        </li>
                    ))}
                </ul>
            );
        }

        const numberedLines = lines.filter((line) => /^\d+\.\s+/.test(line));
        if (numberedLines.length === lines.length && numberedLines.length > 0) {
            const firstNumber = Number(numberedLines[0].match(/^(\d+)\./)?.[1] ?? "1");

            return (
                <ol key={`paragraph-${paragraphIndex}`} start={firstNumber} className="list-decimal space-y-2 pl-5 leading-7">
                    {numberedLines.map((line, lineIndex) => (
                        <li key={`paragraph-${paragraphIndex}-item-${lineIndex}`}>
                            {renderInline(line.replace(/^\d+\.\s+/, ""), `number-${paragraphIndex}-${lineIndex}`)}
                        </li>
                    ))}
                </ol>
            );
        }

        const quoteLines = lines.filter((line) => /^>\s?/.test(line));
        if (quoteLines.length === lines.length && quoteLines.length > 0) {
            return (
                <blockquote
                    key={`paragraph-${paragraphIndex}`}
                    className="rounded-2xl border-l-4 border-primary/35 bg-muted/35 px-4 py-3 text-sm leading-7 text-muted-foreground"
                >
                    {quoteLines.map((line, lineIndex) => (
                        <p key={`paragraph-${paragraphIndex}-quote-${lineIndex}`}>
                            {renderInline(line.replace(/^>\s?/, ""), `quote-${paragraphIndex}-${lineIndex}`)}
                        </p>
                    ))}
                </blockquote>
            );
        }

        return (
            <p key={`paragraph-${paragraphIndex}`} className="leading-7">
                {renderParagraphLines(lines, paragraphIndex)}
            </p>
        );
    });
}
