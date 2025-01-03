package he

import (
	"log/slog"
	"net/http"
)

type AppError struct {
	Error       error
	HTTPMessage string
	Desc        string
	Code        int
}

type AppHandler func(w http.ResponseWriter, r *http.Request) *AppError

func (fn AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		slog.Error("Error in handler:", "desc", e.Desc, "httpMessage", e.HTTPMessage, "code", e.Code)
		http.Error(w, e.HTTPMessage, e.Code)
	}
}

func InternalError(err error, desc string) *AppError {
	return &AppError{
		HTTPMessage: "Internal server error",
		Desc:        desc,
		Code:        http.StatusInternalServerError,
		Error:       err,
	}
}

func BadRequestError(err error, desc string) *AppError {
	return &AppError{
		HTTPMessage: "Bad request",
		Desc:        desc,
		Code:        http.StatusBadRequest,
		Error:       err,
	}
}

func UnauthorizedError(err error, desc string) *AppError {
	return &AppError{
		HTTPMessage: "Unauthorized",
		Desc:        desc,
		Code:        http.StatusUnauthorized,
		Error:       err,
	}
}
