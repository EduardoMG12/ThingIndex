import { translateRuntime, type TranslationVariables } from "@/lib/i18n/translate";

export function apiErrorMessage(key: string, variables?: TranslationVariables) {
    return translateRuntime(`apiErrors.${key}`, variables);
}
