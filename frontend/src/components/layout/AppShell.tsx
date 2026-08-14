"use client";

import { useEffect, useRef, useState, type ChangeEvent } from "react";
import { Camera, Loader2, LogOut, Settings, LayoutDashboard, Database, BookOpen, MessageSquare, ChevronDown, ChevronRight, Pencil, Check, X, MessageSquarePlus, Trash2, Coins } from "lucide-react";
import Link from "next/link";
import Image from "next/image";
import { usePathname, useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { logout } from "@/components/auth/actions";
import { Logo } from "@/components/ui/Logo";
import { ThemeToggle } from "@/components/ui/ThemeToggle";
import { LanguageSwitcher } from "@/components/ui/LanguageSwitcher";
import { DangerConfirmModal } from "@/components/ui/danger-confirm-modal";
import { useI18n } from "@/lib/i18n/provider";
import { ScrollArea } from "@/components/ui/scroll-area";
import { deleteChatConversation, updateChatConversationTitle, type ChatConversationSummary } from "@/lib/api/chat";
import { fetchCurrentUser, type CurrentUser, updateCurrentUserProfile } from "@/lib/api/current-user";
import { toast } from "sonner";

type ChatSidebarSyncDetail = {
    selectedCollectionId: number | null;
    activeConversationId: number | null;
    conversations: ChatConversationSummary[];
};

interface AppShellProps {
    children: React.ReactNode;
    contentClassName?: string;
}

function getUserInitials(user: CurrentUser | null, fallbackLabel: string) {
    const source = user?.name?.trim() || user?.email?.trim() || fallbackLabel;
    const parts = source
        .split(/\s+/)
        .map((part) => part.trim())
        .filter(Boolean);

    if (parts.length === 0) {
        return "TI";
    }

    if (parts.length === 1) {
        return parts[0].slice(0, 2).toUpperCase();
    }

    return `${parts[0][0] ?? ""}${parts[1][0] ?? ""}`.toUpperCase();
}

const AppShell = ({ children, contentClassName }: AppShellProps) => {
    const pathname = usePathname();
    const router = useRouter();
    const { t, href } = useI18n();
    const [isChatListOpen, setIsChatListOpen] = useState(true);
    const [chatSidebarState, setChatSidebarState] = useState<ChatSidebarSyncDetail>({
        selectedCollectionId: null,
        activeConversationId: null,
        conversations: [],
    });
    const [editingConversationId, setEditingConversationId] = useState<number | null>(null);
    const [editingTitle, setEditingTitle] = useState("");
    const [isSavingConversationTitle, setIsSavingConversationTitle] = useState(false);
    const [conversationPendingDelete, setConversationPendingDelete] = useState<ChatConversationSummary | null>(null);
    const [conversationDeleteConfirmationValue, setConversationDeleteConfirmationValue] = useState("");
    const [isDeletingConversation, setIsDeletingConversation] = useState(false);
    const [currentUser, setCurrentUser] = useState<CurrentUser | null>(null);
    const [isCurrentUserLoading, setIsCurrentUserLoading] = useState(true);
    const [isEditingProfile, setIsEditingProfile] = useState(false);
    const [profileNameDraft, setProfileNameDraft] = useState("");
    const [profileAvatarFile, setProfileAvatarFile] = useState<File | null>(null);
    const [profileAvatarPreviewUrl, setProfileAvatarPreviewUrl] = useState<string | null>(null);
    const [isSavingProfile, setIsSavingProfile] = useState(false);
    const [hasAvatarLoadError, setHasAvatarLoadError] = useState(false);
    const avatarInputRef = useRef<HTMLInputElement | null>(null);

    const navItems = [
        { label: t("shell.nav.dashboard"), icon: LayoutDashboard, href: href("/dashboard") },
        { label: t("shell.nav.chat"), icon: MessageSquare, href: href("/chat") },
        { label: t("shell.nav.documents"), icon: Database, href: href("/documents") },
        { label: t("shell.nav.futureFeature"), icon: Coins, href: href("/cost-visibility") },
        { label: t("shell.nav.documentation"), icon: BookOpen, href: href("/documentation") },
        { label: t("shell.nav.settings"), icon: Settings, href: href("/settings") },
    ];
    const activeItem = navItems.find((item) => pathname === item.href || pathname.startsWith(`${item.href}/`));
    const chatHref = href("/chat");
    const isChatRoute = pathname === chatHref || pathname.startsWith(`${chatHref}/`);

    useEffect(() => {
        if (!isChatRoute) {
            return;
        }

        const handleSync = (event: Event) => {
            const detail = (event as CustomEvent<ChatSidebarSyncDetail>).detail;
            if (!detail) {
                return;
            }
            setChatSidebarState(detail);
        };

        window.addEventListener("thingindex:chat-sidebar-sync", handleSync as EventListener);

        return () => {
            window.removeEventListener("thingindex:chat-sidebar-sync", handleSync as EventListener);
        };
    }, [isChatRoute]);

    useEffect(() => {
        const controller = new AbortController();

        async function loadCurrentUser() {
            try {
                setIsCurrentUserLoading(true);
                const nextUser = await fetchCurrentUser(controller.signal);
                setCurrentUser(nextUser);
            } catch {
                if (!controller.signal.aborted) {
                    setCurrentUser(null);
                }
            } finally {
                if (!controller.signal.aborted) {
                    setIsCurrentUserLoading(false);
                }
            }
        }

        void loadCurrentUser();

        return () => controller.abort();
    }, []);

    useEffect(() => {
        if (!isEditingProfile) {
            setProfileNameDraft(currentUser?.name ?? "");
        }
    }, [currentUser, isEditingProfile]);

    const displayedAvatarSrc = profileAvatarPreviewUrl || (currentUser?.avatarUrl ? `/api/user/me/avatar${currentUser.updatedAt ? `?updatedAt=${encodeURIComponent(currentUser.updatedAt)}` : ""}` : null);

    useEffect(() => {
        setHasAvatarLoadError(false);
    }, [displayedAvatarSrc]);

    const userName = currentUser?.name?.trim() || currentUser?.email?.trim() || t("shell.user.accountFallback");
    const userDetail = currentUser?.email?.trim() || t("shell.user.sessionFallback");
    const userInitials = getUserInitials(currentUser, t("shell.user.accountFallback"));
    const hasProfileChanges =
        profileNameDraft.trim() !== (currentUser?.name?.trim() || "") ||
        profileAvatarFile !== null;

    const resetProfileEditor = () => {
        if (profileAvatarPreviewUrl?.startsWith("blob:")) {
            URL.revokeObjectURL(profileAvatarPreviewUrl);
        }
        setIsEditingProfile(false);
        setProfileAvatarFile(null);
        setProfileAvatarPreviewUrl(null);
        setProfileNameDraft(currentUser?.name ?? "");
    };

    const handleProfileAvatarSelection = (event: ChangeEvent<HTMLInputElement>) => {
        const nextFile = event.target.files?.[0] ?? null;

        if (profileAvatarPreviewUrl?.startsWith("blob:")) {
            URL.revokeObjectURL(profileAvatarPreviewUrl);
        }

        if (!nextFile) {
            setProfileAvatarFile(null);
            setProfileAvatarPreviewUrl(null);
            return;
        }

        setProfileAvatarFile(nextFile);
        setProfileAvatarPreviewUrl(URL.createObjectURL(nextFile));
        event.target.value = "";
    };

    const handleSaveProfile = async () => {
        if (!hasProfileChanges || !profileNameDraft.trim() || isSavingProfile) {
            return;
        }

        setIsSavingProfile(true);
        try {
            const updatedUser = await updateCurrentUserProfile({
                name: profileNameDraft.trim(),
                avatar: profileAvatarFile,
            });
            setCurrentUser(updatedUser);
            resetProfileEditor();
            toast.success(t("shell.user.updateSuccess"));
        } catch (error) {
            toast.error(error instanceof Error ? error.message : t("shell.user.updateError"));
        } finally {
            setIsSavingProfile(false);
        }
    };

    const handleChatSectionClick = () => {
        if (isChatRoute) {
            setIsChatListOpen((current) => !current);
            return;
        }

        setIsChatListOpen(true);
        router.push(chatHref);
    };

    const startEditingConversation = (conversation: ChatConversationSummary) => {
        setEditingConversationId(conversation.id);
        setEditingTitle(conversation.title);
    };

    const cancelEditingConversation = () => {
        setEditingConversationId(null);
        setEditingTitle("");
        setIsSavingConversationTitle(false);
    };

    const saveConversationTitle = async (conversationId: number) => {
        const normalized = editingTitle.trim();
        if (!normalized || isSavingConversationTitle) {
            return;
        }

        setIsSavingConversationTitle(true);
        try {
            const updated = await updateChatConversationTitle(conversationId, normalized);
            setChatSidebarState((current) => ({
                ...current,
                conversations: current.conversations.map((conversation) =>
                    conversation.id === conversationId ? updated : conversation
                ),
            }));
            window.dispatchEvent(new CustomEvent("thingindex:chat-conversation-updated", {
                detail: updated,
            }));
            cancelEditingConversation();
        } catch {
            setIsSavingConversationTitle(false);
        }
    };

    const confirmDeleteConversation = async () => {
        if (!conversationPendingDelete || isDeletingConversation) {
            return;
        }

        setIsDeletingConversation(true);
        try {
            await deleteChatConversation(conversationPendingDelete.id);
            setChatSidebarState((current) => ({
                ...current,
                activeConversationId: current.activeConversationId === conversationPendingDelete.id ? null : current.activeConversationId,
                conversations: current.conversations.filter((conversation) => conversation.id !== conversationPendingDelete.id),
            }));
            window.dispatchEvent(new CustomEvent("thingindex:chat-conversation-deleted", {
                detail: {
                    id: conversationPendingDelete.id,
                },
            }));

            if (chatSidebarState.activeConversationId === conversationPendingDelete.id) {
                const params = new URLSearchParams();
                if (chatSidebarState.selectedCollectionId) {
                    params.set("collection", String(chatSidebarState.selectedCollectionId));
                }
                params.set("new", String(Date.now()));
                router.push(`${chatHref}${params.toString() ? `?${params.toString()}` : ""}`);
            }

            setConversationPendingDelete(null);
            setConversationDeleteConfirmationValue("");
        } finally {
            setIsDeletingConversation(false);
        }
    };

    const handleCreateChatFromSidebar = () => {
        const params = new URLSearchParams();
        if (chatSidebarState.selectedCollectionId) {
            params.set("collection", String(chatSidebarState.selectedCollectionId));
        }
        params.set("new", String(Date.now()));
        router.push(`${chatHref}${params.toString() ? `?${params.toString()}` : ""}`);
    };

    return (
        <div className="flex h-screen w-full bg-background text-foreground overflow-hidden">
            <aside className="w-64 border-r border-border bg-card flex flex-col shrink-0">
                <div className="p-6 border-b border-border flex items-center gap-3">
                    <Logo className="text-primary" size={32} />
                    <div>
                        <h2 className="text-sm font-bold tracking-tight">thing<span className="text-primary">index</span></h2>
                        <p className="text-[10px] text-muted-foreground font-mono uppercase tracking-widest">v0.1.0</p>
                    </div>
                </div>

                <nav className="flex-1 p-4 space-y-1">
                    {navItems.map((item) => {
                        const isActive = pathname === item.href || pathname.startsWith(`${item.href}/`);
                        const Icon = item.icon;

                        if (item.href === chatHref) {
                            return (
                                <div key={item.href} className="space-y-1">
                                    <button
                                        type="button"
                                        onClick={handleChatSectionClick}
                                        className={cn(
                                            "flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm transition-all group",
                                            isActive
                                                ? "bg-primary/10 text-primary font-medium"
                                                : "text-muted-foreground hover:bg-muted hover:text-foreground"
                                        )}
                                    >
                                        <Icon className={cn("h-4 w-4 shrink-0", isActive ? "text-primary" : "text-muted-foreground group-hover:text-foreground")} />
                                        <span className="flex-1 text-left">{item.label}</span>
                                        {isChatListOpen ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
                                    </button>

                                    {isChatRoute && isChatListOpen && (
                                        <div className="ml-3 rounded-xl border border-border/70 bg-background/45 px-2 py-2">
                                            <button
                                                type="button"
                                                onClick={handleCreateChatFromSidebar}
                                                className="mb-2 flex w-full items-center gap-2 rounded-lg bg-primary/10 px-2.5 py-2 text-left text-[11px] font-medium text-primary transition-colors hover:bg-primary/15"
                                            >
                                                <MessageSquarePlus className="h-3.5 w-3.5" />
                                                {t("dashboard.chat.newConversation")}
                                            </button>
                                            <ScrollArea className="max-h-48">
                                                <div className="space-y-1 pr-2">
                                                    {chatSidebarState.conversations.length === 0 ? (
                                                        <div className="rounded-lg px-2 py-2 text-[11px] leading-relaxed text-muted-foreground">
                                                            {t("shell.chat.empty")}
                                                        </div>
                                                    ) : (
                                                        chatSidebarState.conversations.map((conversation) => {
                                                            const isConversationActive = conversation.id === chatSidebarState.activeConversationId;
                                                            const isEditing = editingConversationId === conversation.id;

                                                            return (
                                                                <div
                                                                    key={conversation.id}
                                                                    className={cn(
                                                                        "group rounded-lg px-2 py-2 transition-colors",
                                                                        isConversationActive
                                                                            ? "bg-primary/12 text-primary"
                                                                            : "text-muted-foreground hover:bg-muted hover:text-foreground"
                                                                    )}
                                                                >
                                                                    {isEditing ? (
                                                                        <div className="space-y-2">
                                                                            <input
                                                                                value={editingTitle}
                                                                                onChange={(event) => setEditingTitle(event.target.value)}
                                                                                onKeyDown={(event) => {
                                                                                    if (event.key === "Enter") {
                                                                                        event.preventDefault();
                                                                                        void saveConversationTitle(conversation.id);
                                                                                    }
                                                                                    if (event.key === "Escape") {
                                                                                        event.preventDefault();
                                                                                        cancelEditingConversation();
                                                                                    }
                                                                                }}
                                                                                autoFocus
                                                                                className="w-full rounded-md border border-border bg-background px-2 py-1 text-[11px] text-foreground outline-none"
                                                                            />
                                                                            <div className="flex justify-end gap-1">
                                                                                <button
                                                                                    type="button"
                                                                                    onClick={cancelEditingConversation}
                                                                                    className="rounded-md p-1 text-muted-foreground hover:bg-muted"
                                                                                >
                                                                                    <X className="h-3.5 w-3.5" />
                                                                                </button>
                                                                                <button
                                                                                    type="button"
                                                                                    onClick={() => void saveConversationTitle(conversation.id)}
                                                                                    disabled={isSavingConversationTitle || !editingTitle.trim()}
                                                                                    className="rounded-md p-1 text-primary hover:bg-primary/10 disabled:opacity-50"
                                                                                >
                                                                                    <Check className="h-3.5 w-3.5" />
                                                                                </button>
                                                                            </div>
                                                                        </div>
                                                                    ) : (
                                                                        <div className="flex items-start gap-2">
                                                                            <button
                                                                                type="button"
                                                                                onClick={() => router.push(`${chatHref}?collection=${conversation.collectionId}&conversation=${conversation.id}`)}
                                                                                className="flex-1 text-left"
                                                                            >
                                                                                <span className="line-clamp-2 text-[11px] leading-snug">
                                                                                    {conversation.title}
                                                                                </span>
                                                                            </button>
                                                                            <button
                                                                                type="button"
                                                                                onClick={() => startEditingConversation(conversation)}
                                                                                aria-label={t("dashboard.chat.renameConversationLabel")}
                                                                                className="rounded-md p-1 text-muted-foreground opacity-0 transition-opacity hover:bg-muted hover:text-foreground group-hover:opacity-100 focus-visible:opacity-100"
                                                                            >
                                                                                <Pencil className="h-3.5 w-3.5" />
                                                                            </button>
                                                                            <button
                                                                                type="button"
                                                                                onClick={() => {
                                                                                    setConversationPendingDelete(conversation);
                                                                                    setConversationDeleteConfirmationValue("");
                                                                                }}
                                                                                aria-label={t("dashboard.chat.deleteConversationLabel")}
                                                                                className="rounded-md p-1 text-muted-foreground opacity-0 transition-opacity hover:bg-destructive/10 hover:text-destructive group-hover:opacity-100 focus-visible:opacity-100"
                                                                            >
                                                                                <Trash2 className="h-3.5 w-3.5" />
                                                                            </button>
                                                                        </div>
                                                                    )}
                                                                </div>
                                                            );
                                                        })
                                                    )}
                                                </div>
                                            </ScrollArea>
                                        </div>
                                    )}
                                </div>
                            );
                        }

                        return (
                            <Link
                                key={item.href}
                                href={item.href}
                                className={cn(
                                    "flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-all group",
                                    isActive
                                        ? "bg-primary/10 text-primary font-medium"
                                        : "text-muted-foreground hover:bg-muted hover:text-foreground"
                                )}
                            >
                                <Icon className={cn("h-4 w-4", isActive ? "text-primary" : "text-muted-foreground group-hover:text-foreground")} />
                                {item.label}
                            </Link>
                        );
                    })}
                </nav>

                <div className="p-4 border-t border-border space-y-2">
                    <div className="rounded-lg bg-secondary/30 border border-border px-3 py-3">
                        <div className="flex items-start gap-3">
                            <div className="relative shrink-0">
                                {displayedAvatarSrc && !hasAvatarLoadError ? (
                                    <Image
                                        src={displayedAvatarSrc}
                                        alt={userName}
                                        width={40}
                                        height={40}
                                        unoptimized
                                        className="h-10 w-10 rounded-full object-cover border border-border"
                                        onError={() => setHasAvatarLoadError(true)}
                                    />
                                ) : (
                                    <div className="h-10 w-10 rounded-full bg-primary/20 flex items-center justify-center border border-primary/15">
                                        <span className="text-[11px] font-bold uppercase tracking-wide text-primary">{userInitials}</span>
                                    </div>
                                )}
                                {isEditingProfile ? (
                                    <button
                                        type="button"
                                        onClick={() => avatarInputRef.current?.click()}
                                        className="absolute -bottom-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full border border-border bg-background text-muted-foreground transition hover:text-foreground"
                                        aria-label={t("shell.user.changePhoto")}
                                    >
                                        <Camera className="h-3 w-3" />
                                    </button>
                                ) : null}
                            </div>

                            <div className="min-w-0 flex-1">
                                {isCurrentUserLoading ? (
                                    <>
                                        <div className="h-3 w-24 rounded bg-muted animate-pulse" />
                                        <div className="mt-2 h-2.5 w-28 rounded bg-muted animate-pulse" />
                                    </>
                                ) : (
                                    <>
                                        <div className="flex items-start justify-between gap-2">
                                            <div className="min-w-0">
                                                <p className="text-xs font-semibold truncate leading-none">{userName}</p>
                                                <p className="text-[10px] text-muted-foreground truncate mt-1">{userDetail}</p>
                                            </div>
                                            <button
                                                type="button"
                                                onClick={() => setIsEditingProfile(true)}
                                                className="rounded-md p-1 text-muted-foreground transition hover:bg-muted hover:text-foreground"
                                                aria-label={t("shell.user.edit")}
                                            >
                                                <Pencil className="h-3.5 w-3.5" />
                                            </button>
                                        </div>

                                        {isEditingProfile ? (
                                            <div className="mt-3 space-y-3">
                                                <input
                                                    ref={avatarInputRef}
                                                    type="file"
                                                    accept="image/png,image/jpeg,image/webp"
                                                    className="hidden"
                                                    onChange={handleProfileAvatarSelection}
                                                />
                                                <div className="space-y-1">
                                                    <label htmlFor="shell-user-name" className="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                                                        {t("shell.user.nameLabel")}
                                                    </label>
                                                    <input
                                                        id="shell-user-name"
                                                        value={profileNameDraft}
                                                        onChange={(event) => setProfileNameDraft(event.target.value)}
                                                        className="w-full rounded-md border border-border bg-background px-2.5 py-2 text-xs text-foreground outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/20"
                                                    />
                                                </div>
                                                <div className="flex flex-wrap items-center gap-2">
                                                    <button
                                                        type="button"
                                                        onClick={() => avatarInputRef.current?.click()}
                                                        className="inline-flex items-center gap-2 rounded-md border border-border bg-background px-2.5 py-2 text-[11px] font-medium text-foreground transition hover:bg-muted"
                                                    >
                                                        <Camera className="h-3.5 w-3.5" />
                                                        {t("shell.user.changePhoto")}
                                                    </button>
                                                    <span className="text-[10px] text-muted-foreground">
                                                        {profileAvatarFile?.name || t("shell.user.photoHint")}
                                                    </span>
                                                </div>
                                                <div className="flex justify-end gap-2">
                                                    <button
                                                        type="button"
                                                        onClick={resetProfileEditor}
                                                        disabled={isSavingProfile}
                                                        className="inline-flex items-center gap-2 rounded-md border border-border bg-background px-2.5 py-2 text-[11px] font-medium text-muted-foreground transition hover:bg-muted disabled:opacity-60"
                                                    >
                                                        <X className="h-3.5 w-3.5" />
                                                        {t("shell.user.cancel")}
                                                    </button>
                                                    <button
                                                        type="button"
                                                        onClick={() => void handleSaveProfile()}
                                                        disabled={!hasProfileChanges || !profileNameDraft.trim() || isSavingProfile}
                                                        className="inline-flex items-center gap-2 rounded-md border border-primary/20 bg-primary/10 px-2.5 py-2 text-[11px] font-medium text-primary transition hover:bg-primary/15 disabled:opacity-60"
                                                    >
                                                        {isSavingProfile ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />}
                                                        {t("shell.user.save")}
                                                    </button>
                                                </div>
                                            </div>
                                        ) : null}
                                    </>
                                )}
                            </div>
                        </div>
                    </div>
                    <form action={logout}>
                        <button type="submit" className="flex w-full items-center gap-3 px-3 py-2 rounded-lg text-sm text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-all">
                            <LogOut className="h-4 w-4" />
                            {t("shell.user.logout")}
                        </button>
                    </form>
                </div>
            </aside>

            <DangerConfirmModal
                isOpen={conversationPendingDelete !== null}
                eyebrow={t("shell.chat.deleteConversation.eyebrow")}
                title={conversationPendingDelete
                    ? t("shell.chat.deleteConversation.titleWithName", { title: conversationPendingDelete.title })
                    : t("shell.chat.deleteConversation.title")}
                description={t("shell.chat.deleteConversation.description")}
                confirmLabel={t("shell.chat.deleteConversation.confirmLabel")}
                cancelLabel={t("common.dangerConfirm.cancelLabel")}
                closeLabel={t("common.dangerConfirm.closeLabel")}
                expectedValue={conversationPendingDelete?.title}
                confirmationValue={conversationDeleteConfirmationValue}
                confirmationPrompt={conversationPendingDelete
                    ? t("common.dangerConfirm.typeToConfirm", { value: conversationPendingDelete.title })
                    : undefined}
                inputLabel={t("common.dangerConfirm.inputLabel")}
                onConfirmationChange={setConversationDeleteConfirmationValue}
                onCancel={() => {
                    if (isDeletingConversation) {
                        return;
                    }

                    setConversationPendingDelete(null);
                    setConversationDeleteConfirmationValue("");
                }}
                onConfirm={() => void confirmDeleteConversation()}
                isPending={isDeletingConversation}
            />

            <main className="flex-1 flex flex-col overflow-hidden">
                <header className="h-14 border-b border-border bg-background/50 backdrop-blur-sm flex items-center justify-between px-8 shrink-0">
                    <div className="flex items-center gap-2">
                        <span className="text-[10px] font-mono text-muted-foreground uppercase tracking-widest">{t("shell.workspace")}</span>
                        <span className="text-muted-foreground/30">/</span>
                        <span className="text-[10px] font-mono font-medium uppercase tracking-widest">{activeItem?.label || t("shell.workspace")}</span>
                    </div>
                    <div className="flex items-center gap-4">
                        <LanguageSwitcher />
                        <ThemeToggle />
                    </div>
                </header>

                <div className="flex-1 overflow-y-auto custom-scrollbar" id="main-content">
                    <div className={cn("mx-auto max-w-7xl p-8", contentClassName)}>
                        {children}
                    </div>
                </div>
            </main>
        </div>
    );
};

export default AppShell;
