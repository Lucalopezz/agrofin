package auth

import "errors"

type validationError struct {
	message string
}

func (e validationError) Error() string {
	return e.message
}

func invalidInput(message string) error {
	return validationError{message: message}
}

func isValidationError(err error) bool {
	var target validationError
	return errors.As(err, &target)
}
