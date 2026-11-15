package exceptions

import (
	"net/http"
)

type Causes struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type RestExceptions struct {
	Message string   `json:"message"`
	Err     string   `json:"err"`
	Code    int      `json:"code"`
	Causes  []Causes `json:"causes"`
}


func (r *RestExceptions) Error() string {
	return r.Message
}

func NewBadRequestException(message string, causes ...Causes) *RestExceptions {
	return &RestExceptions{
		Message: message,
		Err:     "bad_request",
		Code:    http.StatusBadRequest,
		Causes:  causes,
	}
}

func NewInternalServerException(message string) *RestExceptions {
	return &RestExceptions{
		Message: message,
		Err:     "internal_server",
		Code:    http.StatusInternalServerError,
		Causes:  nil,
	}
}

func NewNotFoundException(message string) *RestExceptions {
	return &RestExceptions{
		Message: message,
		Err:     "not_found",
		Code:    http.StatusNotFound,
		Causes:  nil,
	}
}

func NewRateLimitException(message string) *RestExceptions {
	return &RestExceptions{
		Message: message,
		Err:     "too_many_requests",
		Code:    http.StatusTooManyRequests,
		Causes:  nil,
	}
}
