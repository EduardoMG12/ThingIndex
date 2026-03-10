package validation

import (
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
	} else {
		errMsg := err.Error()
		if strings.Contains(errMsg, "unknown field") {
			parts := strings.Split(errMsg, "\"")
			if len(parts) >= 2 {
				if language == "pt_BR" {
					errs = append(errs, "O campo '"+parts[1]+"' não é permitido")
				} else {
					errs = append(errs, "Field '"+parts[1]+"' is not allowed")
				}
			} else {
				if language == "pt_BR" {
					errs = append(errs, "Requisição possui campos não permitidos")
				} else {
					errs = append(errs, "Request contains forbidden fields")
				}
			}
		} else {
			errs = append(errs, errMsg)
		}
	}
	return errs
}
