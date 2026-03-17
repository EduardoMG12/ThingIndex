"use client";

import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";

type ProviderModelFieldProps = {
    providerType: string;
    label: string;
    optional?: boolean;
    placeholder: string;
    value: string;
    suggestions?: string[];
    onChange: (value: string) => void;
};

export default function ProviderModelField({
    providerType,
    label,
    optional,
    placeholder,
    value,
    suggestions = [],
    onChange,
}: ProviderModelFieldProps) {
    const listId = `${providerType.toLowerCase()}-model-suggestions`;
    const quickSuggestions = suggestions.slice(0, 6);

    return (
        <div className="space-y-2">
            <Label htmlFor={`${providerType}-model`}>
                {label}
                {optional ? (
                    <span className="ml-2 text-[10px] uppercase tracking-[0.18em] text-muted-foreground">Optional</span>
                ) : null}
            </Label>
            <Input
                id={`${providerType}-model`}
                list={suggestions.length > 0 ? listId : undefined}
                value={value}
                onChange={(event) => onChange(event.target.value)}
                placeholder={placeholder}
                autoComplete="off"
                spellCheck={false}
            />
            {suggestions.length > 0 ? (
                <datalist id={listId}>
                    {suggestions.map((suggestion) => (
                        <option key={suggestion} value={suggestion} />
                    ))}
                </datalist>
            ) : null}
            {quickSuggestions.length > 0 ? (
                <div className="space-y-2">
                    <p className="text-[11px] text-muted-foreground">
                        Choose one of the suggested model IDs or keep typing a custom value.
                    </p>
                    <div className="flex flex-wrap gap-2">
                        {quickSuggestions.map((suggestion) => (
                            <button
                                key={suggestion}
                                type="button"
                                onClick={() => onChange(suggestion)}
                                className="rounded-full border border-border bg-background/70 px-2.5 py-1 text-xs text-foreground/85 transition-colors hover:border-primary/30 hover:text-primary"
                            >
                                {suggestion}
                            </button>
                        ))}
                    </div>
                </div>
            ) : null}
        </div>
    );
}
