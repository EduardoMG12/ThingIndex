package validation

import (
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/pt_BR"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	ptBR_translations "github.com/go-playground/validator/v10/translations/pt_BR"
)

var (
	uni   *ut.UniversalTranslator
	trans ut.Translator
)

func init() {
	binding.EnableDecoderDisallowUnknownFields = true

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		pt := pt_BR.New()
		uni = ut.New(pt, pt)
		trans, _ = uni.GetTranslator("pt_BR")

		ptBR_translations.RegisterDefaultTranslations(v, trans)

	}
}

func FormatValidationErrors(err error) []string {
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
				errs = append(errs, "O campo '"+parts[1]+"' não é permitido")
			} else {
				errs = append(errs, "Requisição possui campos não permitidos")
			}
		} else {
			errs = append(errs, errMsg)
		}
	}
	return errs
}

func FormatValidationErrorsString(err error) string {
	return strings.Join(FormatValidationErrors(err), ", ")
}
