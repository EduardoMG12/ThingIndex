package i18n

import (
	"strings"
)

var (
	Messages = map[string]map[string]string{
		"pt": {
			"auth.user_exists":      "E-mail já cadastrado no sistema",
			"auth.invalid_creds":    "E-mail ou senha incorretos",
			"auth.success_login":    "Login realizado com sucesso",
			"auth.success_register": "Usuário registrado com sucesso",
			"error.internal":        "Erro interno do servidor",
			"error.validation":      "Erro de validação nos campos",
			"error.unauthorized":    "Não autorizado",
		},
		"en": {
			"auth.user_exists":      "Email already registered in the system",
			"auth.invalid_creds":    "Incorrect email or password",
			"auth.success_login":    "Logged in successfully",
			"auth.success_register": "User registered successfully",
			"error.internal":        "Internal server error",
			"error.validation":      "Field validation error",
			"error.unauthorized":    "Unauthorized",
		},
	}
)

func T(lang string, key string) string {
	lang = strings.ToLower(lang)
	if !strings.HasPrefix(lang, "pt") && !strings.HasPrefix(lang, "en") {
		lang = "en"
	} else if strings.HasPrefix(lang, "pt") {
		lang = "pt"
	} else {
		lang = "en"
	}

	if msg, ok := Messages[lang][key]; ok {
		return msg
	}
	return key
}
