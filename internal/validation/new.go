package validation

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

func NewValidator() (func(v interface{}) error, error) {
	v := validator.New()
	reg, err := regexp.Compile("^t2_[a-z0-9]{8}$")
	if err != nil {
		return nil, fmt.Errorf("Couldn't compile a regexp: %w", err)
	}
	if err := v.RegisterValidation("author", func(fl validator.FieldLevel) bool {
		return reg.MatchString(fl.Field().String())
	}); err != nil {
		return nil, fmt.Errorf("Couldn't register a validation")
	}
	return v.Struct, nil
}
