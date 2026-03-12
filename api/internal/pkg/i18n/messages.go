package i18n

import (
	"strings"
)

var (
	Messages = map[string]map[string]string{
		"pt": {
			"auth.user_exists":                       "E-mail já cadastrado no sistema",
			"auth.invalid_creds":                     "E-mail ou senha incorretos",
			"auth.success_login":                     "Login realizado com sucesso",
			"auth.success_register":                  "Usuário registrado com sucesso",
			"auth.success_forgot_password":           "Se o e-mail existir, um link de recuperação será enviado",
			"auth.success_reset_password":            "Senha redefinida com sucesso",
			"auth.invalid_reset_token":               "Token de recuperação inválido",
			"auth.expired_reset_token":               "Token de recuperação expirado",
			"provider.error_invalid_family":          "Família de provider inválida",
			"provider.error_invalid_type":            "Tipo de provider inválido",
			"provider.error_invalid_credentials":     "Credenciais do provider inválidas",
			"provider.success_save":                  "Credenciais salvas com sucesso",
			"provider.success_list":                  "Providers recuperados com sucesso",
			"error.internal":                         "Erro interno do servidor",
			"error.validation":                       "Erro de validação nos campos",
			"error.unauthorized":                     "Não autorizado",
			"error.invalid_category_id":              "ID de categoria inválido",
			"error.invalid_collection_id":            "ID de collection inválido",
			"error.invalid_document_id":              "ID de documento inválido",
			"doc.success_upload":                     "Documento enviado com sucesso",
			"doc.error_file":                         "Arquivo é obrigatório",
			"doc.error_file_too_large":               "Arquivo excede o limite permitido",
			"doc.error_invalid_mime":                 "Tipo de arquivo não permitido",
			"doc.error_title":                        "Título é obrigatório",
			"doc.error_upload":                       "Falha ao processar upload",
			"doc.error_category_required":            "Nome da categoria é obrigatório",
			"doc.error_category_duplicate":           "Você já possui uma categoria com esse nome",
			"doc.error_category_not_found":           "Categoria não encontrada",
			"doc.error_category_patch_empty":         "Informe ao menos um campo para atualizar a categoria",
			"doc.error_collection_required":          "Collection é obrigatória",
			"doc.error_collection_duplicate":         "Você já possui uma collection com esse nome",
			"doc.error_collection_not_found":         "Collection não encontrada",
			"doc.error_collection_category_conflict": "A collection já está vinculada a outra categoria",
			"doc.error_document_not_found":           "Documento não encontrado",
			"doc.success_category_create":            "Categoria criada com sucesso",
			"doc.success_category_update":            "Categoria atualizada com sucesso",
			"doc.success_categories_list":            "Categorias recuperadas com sucesso",
			"doc.success_collection_create":          "Collection criada com sucesso",
			"doc.success_collection_get":             "Collection recuperada com sucesso",
			"doc.success_collection_context_get":     "Contexto da collection recuperado com sucesso",
			"doc.success_collection_category_update": "Categoria da collection atualizada com sucesso",
			"doc.success_collections_list":           "Collections recuperadas com sucesso",
			"doc.default_category_name":              "Sem Categoria",
			"doc.default_category_description":       "Documentos que não foram atribuídos a uma categoria específica.",
			"doc.auto_category_desc":                 "Categoria gerada automaticamente",
			"doc.success_document_get":               "Documento recuperado com sucesso",
			"doc.success_documents_list":             "Documentos recuperados com sucesso",
			"doc.success_paragraph_nodes_store":      "Parágrafos estruturados salvos com sucesso",
			"doc.success_table_sections_store":       "Tabelas estruturadas salvas com sucesso",
			"doc.success_metadata_pairs_store":       "Metadados estruturados salvos com sucesso",
		},
		"en": {
			"auth.user_exists":                       "Email already registered in the system",
			"auth.invalid_creds":                     "Incorrect email or password",
			"auth.success_login":                     "Logged in successfully",
			"auth.success_register":                  "User registered successfully",
			"auth.success_forgot_password":           "If the email exists, a reset link will be sent",
			"auth.success_reset_password":            "Password reset successfully",
			"auth.invalid_reset_token":               "Invalid reset token",
			"auth.expired_reset_token":               "Reset token expired",
			"provider.error_invalid_family":          "Invalid provider family",
			"provider.error_invalid_type":            "Invalid provider type",
			"provider.error_invalid_credentials":     "Invalid provider credentials",
			"provider.success_save":                  "Credentials saved successfully",
			"provider.success_list":                  "Providers retrieved successfully",
			"error.internal":                         "Internal server error",
			"error.validation":                       "Field validation error",
			"error.unauthorized":                     "Unauthorized",
			"error.invalid_category_id":              "Invalid category ID",
			"error.invalid_collection_id":            "Invalid collection ID",
			"error.invalid_document_id":              "Invalid document ID",
			"doc.success_upload":                     "Document uploaded successfully",
			"doc.error_file":                         "File is required",
			"doc.error_file_too_large":               "File exceeds the allowed size limit",
			"doc.error_invalid_mime":                 "File type is not allowed",
			"doc.error_title":                        "Title is required",
			"doc.error_upload":                       "Failed to process upload",
			"doc.error_category_required":            "Category name is required",
			"doc.error_category_duplicate":           "You already have a category with this name",
			"doc.error_category_not_found":           "Category not found",
			"doc.error_category_patch_empty":         "Provide at least one field to update the category",
			"doc.error_collection_required":          "Collection is required",
			"doc.error_collection_duplicate":         "You already have a collection with this name",
			"doc.error_collection_not_found":         "Collection not found",
			"doc.error_collection_category_conflict": "The collection is already linked to another category",
			"doc.error_document_not_found":           "Document not found",
			"doc.success_category_create":            "Category created successfully",
			"doc.success_category_update":            "Category updated successfully",
			"doc.success_categories_list":            "Categories retrieved successfully",
			"doc.success_collection_create":          "Collection created successfully",
			"doc.success_collection_get":             "Collection retrieved successfully",
			"doc.success_collection_context_get":     "Collection context retrieved successfully",
			"doc.success_collection_category_update": "Collection category updated successfully",
			"doc.success_collections_list":           "Collections retrieved successfully",
			"doc.default_category_name":              "Uncategorized",
			"doc.default_category_description":       "Documents that have not been assigned to a specific category.",
			"doc.auto_category_desc":                 "Auto-generated category",
			"doc.success_document_get":               "Document retrieved successfully",
			"doc.success_documents_list":             "Documents retrieved successfully",
			"doc.success_paragraph_nodes_store":      "Structured paragraphs stored successfully",
			"doc.success_table_sections_store":       "Structured tables stored successfully",
			"doc.success_metadata_pairs_store":       "Structured metadata stored successfully",
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
