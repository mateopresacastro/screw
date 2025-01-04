package herr

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

type Error struct {
	Error       error
	HTTPMessage string
	Desc        string
	Code        int
}

type Wrap func(w http.ResponseWriter, r *http.Request) *Error

func (fn Wrap) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		slog.Error("Error in handler:", "desc", e.Desc, "httpMessage", e.HTTPMessage, "code", e.Code)
		http.Error(w, e.HTTPMessage, e.Code)
	}
}

func Internal(err error, desc string) *Error {
	return &Error{
		HTTPMessage: "Internal server error",
		Desc:        desc,
		Code:        http.StatusInternalServerError,
		Error:       err,
	}
}

func BadRequest(err error, desc string) *Error {
	return &Error{
		HTTPMessage: "Bad request",
		Desc:        desc,
		Code:        http.StatusBadRequest,
		Error:       err,
	}
}

func Unauthorized(err error, desc string) *Error {
	return &Error{
		HTTPMessage: "Unauthorized",
		Desc:        desc,
		Code:        http.StatusUnauthorized,
		Error:       err,
	}
}

func WS(conn *websocket.Conn, err error, desc string) {
	code := websocket.CloseInternalServerErr
	if errors.Is(err, context.Canceled) {
		code = websocket.CloseGoingAway
	}

	slog.Error("WebSocket error", "desc", desc, "error", err)
	conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(code, desc))
}

func WSClose(conn *websocket.Conn, desc string) {
	slog.Info("WebSocket closing", "desc", desc)
	conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, desc))
}
