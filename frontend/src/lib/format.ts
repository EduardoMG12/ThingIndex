export function formatUSD(value: number, locale: string) {
    return new Intl.NumberFormat(locale, {
        style: "currency",
        currency: "USD",
        minimumFractionDigits: 6,
        maximumFractionDigits: 6,
    }).format(value);
}

// Valores ausentes, negativos ou NaN aparecem como US$ 0.
export function formatNonNegativeUSD(value: number, locale: string) {
    return formatUSD(value > 0 ? value : 0, locale);
}

export function formatDateTime(value: string | undefined, locale: string) {
    if (!value) {
        return "-";
    }

    return new Intl.DateTimeFormat(locale, {
        dateStyle: "short",
        timeStyle: "short",
    }).format(new Date(value));
}
