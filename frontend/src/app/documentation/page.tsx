"use client";

import { useMemo } from "react";
import { motion, type Transition } from "framer-motion";
import {
    BookOpen,
    Code2,
    Database,
    GitBranch,
    Network,
    ScrollText,
    ServerCog,
    ShieldCheck,
    Target,
    Workflow,
} from "lucide-react";

import { ArchitectureDiagram } from "@/components/documentation/ArchitectureDiagram";
import { DocumentationSection } from "@/components/documentation/DocumentationSection";
import { IndexingBenchmarkSection } from "@/components/documentation/IndexingBenchmarkSection";
import { OpenSourceOnboardingSection } from "@/components/documentation/OpenSourceOnboardingSection";
import {
    getDocumentationPageContent,
} from "@/components/documentation/page-content";
import { ProductFocusSection } from "@/components/documentation/ProductFocusSection";
import { ProviderSupportSection } from "@/components/documentation/ProviderSupportSection";
import { TableOfContents } from "@/components/documentation/TableOfContents";
import AppShell from "@/components/layout/AppShell";
import { useI18n } from "@/lib/i18n/provider";

const fadeUp = {
    initial: { opacity: 0, y: 16 },
    animate: { opacity: 1, y: 0 },
    transition: {
        duration: 0.5,
        ease: [0.22, 1, 0.36, 1],
    } as Transition,
};

export default function DocumentationPage() {
    const { locale } = useI18n();
    const content = useMemo(() => getDocumentationPageContent(locale), [locale]);

    return (
        <AppShell>
            <motion.section
                {...fadeUp}
                className="relative mb-12 overflow-hidden rounded-[28px] border border-primary/15 bg-card/85 px-8 py-10 shadow-card"
            >
                <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(245,166,35,0.16),transparent_22%),radial-gradient(circle_at_bottom_right,rgba(82,168,255,0.1),transparent_28%)]" />
                <div className="relative">
                    <div className="flex flex-wrap items-center gap-2">
                        {content.heroBadges.map((badge, index) => (
                            <span
                                key={badge}
                                className={index === 0
                                    ? "rounded-full border border-primary/20 bg-primary/10 px-3 py-1 font-mono text-[10px] uppercase tracking-[0.24em] text-primary"
                                    : "rounded-full border border-border bg-background/70 px-3 py-1 font-mono text-[10px] uppercase tracking-[0.24em] text-muted-foreground"}
                            >
                                {badge}
                            </span>
                        ))}
                    </div>

                    <h1 className="mt-6 max-w-4xl font-display text-4xl font-bold tracking-tight text-foreground sm:text-5xl">
                        {content.heroTitle.start}
                        <span className="text-primary"> {content.heroTitle.highlight}</span>
                    </h1>

                    <p className="mt-5 max-w-3xl text-base leading-relaxed text-muted-foreground">
                        {content.heroDescription}
                    </p>

                    <div className="mt-8 grid gap-4 lg:grid-cols-4">
                        {content.heroSnapshot.map((item) => (
                            <div key={item.label} className="rounded-2xl border border-border bg-background/60 p-4">
                                <p className="text-[10px] font-bold uppercase tracking-[0.22em] text-muted-foreground">{item.label}</p>
                                <p className="mt-2 text-sm font-medium leading-relaxed text-foreground/90">{item.value}</p>
                            </div>
                        ))}
                    </div>
                </div>
            </motion.section>

            <div className="grid gap-8 lg:grid-cols-[250px_minmax(0,1fr)]">
                <motion.nav
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: 0.15, duration: 0.5 }}
                    className="hidden min-w-0 lg:block"
                >
                    <TableOfContents items={content.toc} />
                </motion.nav>

                <div className="min-w-0 space-y-10">
                    <DocumentationSection icon={Target} title={content.overview.title} id="overview" delay={0.1}>
                        {content.overview.paragraphs.map((paragraph, index) => (
                            <p key={paragraph} className={index > 0 ? "mt-4" : undefined}>
                                {paragraph}
                            </p>
                        ))}
                    </DocumentationSection>

                    <DocumentationSection icon={Target} title={content.productFocusTitle} id="product-focus" delay={0.12}>
                        <ProductFocusSection />
                    </DocumentationSection>

                    <DocumentationSection icon={Workflow} title={content.platformSurfaces.title} id="platform-surfaces" delay={0.14}>
                        <p className="mb-6">{content.platformSurfaces.intro}</p>
                        <div className="grid gap-4 md:grid-cols-2">
                            {content.platformSurfaces.items.map((item) => (
                                <div key={item.title} className="rounded-xl border border-border bg-background/50 p-5">
                                    <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                                    <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{item.body}</p>
                                </div>
                            ))}
                        </div>
                    </DocumentationSection>

                    <DocumentationSection icon={BookOpen} title={content.principles.title} id="principles" delay={0.16}>
                        <div className="grid gap-4 md:grid-cols-2">
                            {content.principles.items.map((item) => (
                                <div key={item.title} className="rounded-xl border border-border bg-background/50 p-5">
                                    <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                                    <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{item.body}</p>
                                </div>
                            ))}
                        </div>
                    </DocumentationSection>

                    <DocumentationSection icon={Network} title={content.architecture.title} id="architecture" delay={0.2}>
                        <p className="mb-6">{content.architecture.intro}</p>

                        <div className="grid gap-4 lg:grid-cols-2">
                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">{content.architecture.layersTitle}</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    {content.architecture.layers.map((item) => (
                                        <li key={item.label}>
                                            <strong className="text-foreground">{item.label}:</strong> {item.description}
                                        </li>
                                    ))}
                                </ul>
                            </div>

                            <div className="rounded-xl border border-border bg-background/50 p-5">
                                <h4 className="text-xs font-bold uppercase tracking-[0.2em] text-primary">{content.architecture.goalsTitle}</h4>
                                <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                                    {content.architecture.goals.map((item) => (
                                        <li key={item}>{item}</li>
                                    ))}
                                </ul>
                            </div>
                        </div>

                        <ArchitectureDiagram items={content.architecture.layers} />
                    </DocumentationSection>

                    <DocumentationSection icon={Workflow} title={content.processing.title} id="processing" delay={0.24}>
                        <p className="mb-6">{content.processing.intro}</p>

                        <div className="space-y-4">
                            {content.processing.steps.map((item) => (
                                <div key={item.step} className="rounded-xl border border-border bg-background/50 p-5">
                                    <div className="flex items-center gap-3">
                                        <span className="rounded-md bg-primary/10 px-2 py-1 text-[10px] font-bold uppercase tracking-[0.2em] text-primary">
                                            {item.step}
                                        </span>
                                        <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                                    </div>
                                    <p className="mt-3 text-sm leading-relaxed text-muted-foreground">{item.description}</p>
                                </div>
                            ))}
                        </div>
                    </DocumentationSection>

                    <DocumentationSection icon={Database} title={content.dataModel.title} id="data-model" delay={0.28}>
                        {content.dataModel.paragraphs.map((paragraph, index) => (
                            <p key={paragraph} className={index > 0 ? "mt-4" : undefined}>
                                {paragraph}
                            </p>
                        ))}
                    </DocumentationSection>

                    <DocumentationSection icon={Database} title={content.retrieval.title} id="retrieval" delay={0.32}>
                        {content.retrieval.paragraphs.map((paragraph, index) => (
                            <p key={paragraph} className={index > 0 ? "mt-4" : undefined}>
                                {paragraph}
                            </p>
                        ))}
                    </DocumentationSection>

                    <DocumentationSection icon={Target} title={content.indexingBenchmarkTitle} id="indexing-benchmark" delay={0.36}>
                        <IndexingBenchmarkSection />
                    </DocumentationSection>

                    <DocumentationSection icon={ServerCog} title={content.providersTitle} id="providers" delay={0.4}>
                        <ProviderSupportSection />
                    </DocumentationSection>

                    <DocumentationSection icon={ShieldCheck} title={content.security.title} id="security" delay={0.44}>
                        {content.security.paragraphs.map((paragraph, index) => (
                            <p key={paragraph} className={index > 0 ? "mt-4" : undefined}>
                                {paragraph}
                            </p>
                        ))}
                    </DocumentationSection>

                    <DocumentationSection icon={Code2} title={content.quality.title} id="quality" delay={0.48}>
                        <p className="mb-6">{content.quality.intro}</p>
                        <ul className="space-y-3">
                            {content.quality.items.map((item) => (
                                <li key={item}>{item}</li>
                            ))}
                        </ul>
                        <p className="mt-6 rounded-xl border border-primary/15 bg-primary/10 px-4 py-4 text-sm leading-relaxed text-foreground/90">
                            {content.quality.note}
                        </p>
                    </DocumentationSection>

                    <DocumentationSection icon={ServerCog} title={content.deployment.title} id="deployment" delay={0.52}>
                        {content.deployment.paragraphs.map((paragraph, index) => (
                            <p key={paragraph} className={index > 0 ? "mt-4" : undefined}>
                                {paragraph}
                            </p>
                        ))}
                    </DocumentationSection>

                    <DocumentationSection icon={BookOpen} title={content.openSourceTitle} id="open-source" delay={0.56}>
                        <OpenSourceOnboardingSection apiBaseUrl={process.env.NEXT_PUBLIC_API_URL} />
                    </DocumentationSection>

                    <DocumentationSection icon={GitBranch} title={content.roadmap.title} id="roadmap" delay={0.6}>
                        <ul className="space-y-3">
                            {content.roadmap.items.map((item) => (
                                <li key={item}>{item}</li>
                            ))}
                        </ul>
                    </DocumentationSection>

                    <DocumentationSection icon={ScrollText} title={content.academic.title} id="academic" delay={0.64}>
                        {content.academic.paragraphs.map((paragraph, index) => (
                            <p key={paragraph} className={index > 0 ? "mt-4" : undefined}>
                                {paragraph}
                            </p>
                        ))}
                    </DocumentationSection>

                    <DocumentationSection icon={GitBranch} title={content.contributing.title} id="contributing" delay={0.68}>
                        {content.contributing.paragraphs.map((paragraph, index) => (
                            <p key={paragraph} className={index > 0 ? "mt-4" : undefined}>
                                {paragraph}
                            </p>
                        ))}
                    </DocumentationSection>

                    <DocumentationSection icon={ScrollText} title={content.references.title} id="references" delay={0.72}>
                        <div className="space-y-4 text-muted-foreground">
                            <p>{content.references.intro}</p>
                            <ul className="space-y-3">
                                {content.references.tracks.map((item) => (
                                    <li key={item.label}>
                                        <strong className="text-foreground">{item.label}:</strong>{" "}
                                        {item.references.map((reference, index) => (
                                            <span key={`${item.label}-${reference}`}>
                                                {index > 0 ? ", " : null}
                                                {reference === "this page" || reference === "esta página"
                                                    ? reference
                                                    : <code>{reference}</code>}
                                            </span>
                                        ))}
                                        .
                                    </li>
                                ))}
                            </ul>
                        </div>
                    </DocumentationSection>
                </div>
            </div>
        </AppShell>
    );
}
