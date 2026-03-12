package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/pt_BR"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	ptBR_translations "github.com/go-playground/validator/v10/translations/pt_BR"
)

var (
	uni *ut.UniversalTranslator
)

func init() {
	binding.EnableDecoderDisallowUnknownFields = true

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		pt := pt_BR.New()
		enLoc := en.New()
		uni = ut.New(enLoc, pt, enLoc)

		transPT, _ := uni.GetTranslator("pt_BR")
		ptBR_translations.RegisterDefaultTranslations(v, transPT)

		transEN, _ := uni.GetTranslator("en")
		en_translations.RegisterDefaultTranslations(v, transEN)
	}
}

func GetTranslator(lang string) ut.Translator {
	trans, _ := uni.GetTranslator(lang)
	return trans
}

func FormatValidationErrors(c *gin.Context, err error) []string {
	lang, _ := c.Get("language")
	language, ok := lang.(string)
	if !ok {
		language = "en"
	}

	trans := GetTranslator(language)

	var errs []string
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range validationErrs {
			errs = append(errs, fe.Translate(trans))
		}
		return errs
	}

	return []string{safeBindingErrorMessage(language, err)}
}

func safeBindingErrorMessage(language string, err error) string {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		fieldName := normalizedFieldName(typeErr.Field)
		if fieldName != "" {
			if language == "pt_BR" {
				return fmt.Sprintf("O campo '%s' possui um tipo de valor inválido", fieldName)
			}
			return fmt.Sprintf("Field '%s' has an invalid value type", fieldName)
		}
	}

	if fieldName, ok := extractUnknownFieldName(err); ok {
		if language == "pt_BR" {
			return fmt.Sprintf("O campo '%s' não é permitido", fieldName)
		}
		return fmt.Sprintf("Field '%s' is not allowed", fieldName)
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) || errors.Is(err, io.EOF) {
		if language == "pt_BR" {
			return "Corpo da requisição inválido"
		}
		return "Invalid request body"
	}

	if language == "pt_BR" {
		return "Requisição inválida"
	}
	return "Invalid request"
}

func extractUnknownFieldName(err error) (string, bool) {
	errMsg := err.Error()
	if !strings.Contains(errMsg, "unknown field") {
		return "", false
	}

	parts := strings.Split(errMsg, "\"")
	if len(parts) < 2 {
		return "", false
	}

	fieldName := normalizedFieldName(parts[1])
	if fieldName == "" {
		return "", false
	}

	return fieldName, true
}

func normalizedFieldName(field string) string {
	trimmed := strings.TrimSpace(field)
	if trimmed == "" {
		return ""
	}

	segments := strings.Split(trimmed, ".")
	return segments[len(segments)-1]
}
