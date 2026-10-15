// Classes do <Toaster>, fora do layout para dar para testar. Texto vem do
// tema (branco fixo sumia no tema claro); min-w so a partir de sm.
export const TOAST_CLASS_NAMES = {
    toast: "group rounded-2xl border-border bg-card text-card-foreground p-6 min-w-0 sm:min-w-[440px] shadow-card backdrop-blur-3xl backdrop-saturate-200 transition-all duration-300",
    error: "!border-red-500/60 !bg-red-500/15 !text-card-foreground shadow-[0_0_40px_-10px_rgba(239,68,68,0.6)]",
    success: "!border-emerald-500/40 !bg-emerald-500/15 !text-card-foreground shadow-[0_0_40px_-10px_rgba(16,185,129,0.4)]",
    warning: "!border-amber-500/40 !bg-amber-500/15 !text-card-foreground shadow-[0_0_40px_-10px_rgba(245,158,11,0.4)]",
    info: "!border-blue-500/40 !bg-blue-500/15 !text-card-foreground shadow-[0_0_40px_-10px_rgba(59,130,246,0.4)]",
    default: "!border-border !bg-card !text-card-foreground",
    title: "text-xl font-bold",
    description: "mt-1 text-base font-normal !text-muted-foreground",
} as const;
