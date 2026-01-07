package utils

import (
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ValidateInput(data interface{}) error {
	return validate.Struct(data)
}
