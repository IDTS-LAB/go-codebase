package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"go.uber.org/fx"
)

var Module = fx.Module("validator", fx.Provide(New))

type Validator struct {
	validate *validator.Validate
}

func New() *Validator {
	v := validator.New()
	return &Validator{validate: v}
}

func (v *Validator) Validate(s interface{}) error {
	if err := v.validate.Struct(s); err != nil {
		var errs []string
		for _, e := range err.(validator.ValidationErrors) {
			errs = append(errs, fmt.Sprintf("%s: %s", e.Field(), e.Tag()))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(errs, ", "))
	}
	return nil
}

func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}
