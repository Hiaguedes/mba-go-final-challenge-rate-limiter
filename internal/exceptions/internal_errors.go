package exceptions

import "net/http"

type GenericError struct {
	Message string `json:"message"`
	Err     string `json:"err"`
	Code    int    `json:"code"`
}

func NewOpenDatabaseException(message string, err error) *GenericError {
	return &GenericError{
		Message: message,
		Err:     err.Error(),
		Code:    http.StatusInternalServerError,
	}
}

func NewConfigException(message string, err error) *GenericError {
	return &GenericError{
		Message: message,
		Err:     err.Error(),
		Code:    http.StatusInternalServerError,
	}
}
