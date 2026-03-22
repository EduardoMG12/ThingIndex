import type { AppLocale } from "@/lib/i18n/config";
import { I18nProvider } from "@/lib/i18n/provider";
import type { NextPreview } from "@storybook/nextjs-vite";

import "../src/app/globals.css";

const preview: NextPreview<never> = {
    globalTypes: {
        locale: {
            name: "Locale",
            description: "Active locale for Storybook stories",
            defaultValue: "en",
            toolbar: {
                icon: "globe",
                dynamicTitle: true,
                items: [
                    { value: "en", title: "English" },
                    { value: "pt-BR", title: "Português" },
                    { value: "es", title: "Español" },
                ],
            },
        },
    },
    parameters: {
        layout: "padded",
        nextjs: {
            appDirectory: true,
        },
        options: {
            storySort: {
                order: [
                    "Introduction",
                    "UI",
                    "Runtime",
                    "Settings",
                    "Documents",
                ],
            },
        },
        controls: {
            expanded: true,
            matchers: {
                color: /(background|color)$/i,
                date: /Date$/i,
            },
        },
        backgrounds: {
            default: "workspace",
            values: [
                { name: "workspace", value: "#f5f2e8" },
                { name: "panel", value: "#ffffff" },
                { name: "night", value: "#111827" },
            ],
        },
    },
    decorators: [
        (Story, context) => {
            const locale = (context.globals.locale as AppLocale | undefined) ?? "en";

            return (
                <I18nProvider locale={locale}>
                    <div className="min-h-screen bg-background px-4 py-6 text-foreground">
                        <div className="mx-auto max-w-6xl">
                            <Story />
                        </div>
                    </div>
                </I18nProvider>
            );
        },
    ],
};

export default preview;
