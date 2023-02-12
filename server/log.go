package server

import (
	"fmt"
	"github.com/emersion/go-smtp"
	"github.com/gorilla/websocket"
	"heckel.io/ntfy/log"
	"heckel.io/ntfy/util"
	"net/http"
	"strings"
	"unicode/utf8"
)

// logr creates a new log event with HTTP request fields
func logr(r *http.Request) *log.Event {
	return log.Fields(httpContext(r))
}

// logr creates a new log event with visitor fields
func logv(v *visitor) *log.Event {
	return log.With(v)
}

// logr creates a new log event with HTTP request and visitor fields
func logvr(v *visitor, r *http.Request) *log.Event {
	return logv(v).Fields(httpContext(r))
}

// logvrm creates a new log event with HTTP request, visitor fields and message fields
func logvrm(v *visitor, r *http.Request, m *message) *log.Event {
	return logvr(v, r).With(m)
}

// logvrm creates a new log event with visitor fields and message fields
func logvm(v *visitor, m *message) *log.Event {
	return logv(v).With(m)
}

// logem creates a new log event with email fields
func logem(state *smtp.ConnectionState) *log.Event {
	return log.
		Tag(tagSMTP).
		Fields(log.Context{
			"smtp_hostname":    state.Hostname,
			"smtp_remote_addr": state.RemoteAddr.String(),
		})
}

func httpContext(r *http.Request) log.Context {
	requestURI := r.RequestURI
	if requestURI == "" {
		requestURI = r.URL.Path
	}
	return log.Context{
		"http_method": r.Method,
		"http_path":   requestURI,
	}
}

func websocketErrorContext(err error) log.Context {
	if c, ok := err.(*websocket.CloseError); ok {
		return log.Context{
			"error":      c.Error(),
			"error_code": c.Code,
			"error_type": "websocket.CloseError",
		}
	}
	return log.Context{
		"error": err.Error(),
	}
}

func renderHTTPRequest(r *http.Request) string {
	peekLimit := 4096
	lines := fmt.Sprintf("%s %s %s\n", r.Method, r.URL.RequestURI(), r.Proto)
	for key, values := range r.Header {
		for _, value := range values {
			lines += fmt.Sprintf("%s: %s\n", key, value)
		}
	}
	lines += "\n"
	body, err := util.Peek(r.Body, peekLimit)
	if err != nil {
		lines = fmt.Sprintf("(could not read body: %s)\n", err.Error())
	} else if utf8.Valid(body.PeekedBytes) {
		lines += string(body.PeekedBytes)
		if body.LimitReached {
			lines += fmt.Sprintf(" ... (peeked %d bytes)", peekLimit)
		}
		lines += "\n"
	} else {
		if body.LimitReached {
			lines += fmt.Sprintf("(peeked bytes not UTF-8, peek limit of %d bytes reached, hex: %x ...)\n", peekLimit, body.PeekedBytes)
		} else {
			lines += fmt.Sprintf("(peeked bytes not UTF-8, %d bytes, hex: %x)\n", len(body.PeekedBytes), body.PeekedBytes)
		}
	}
	r.Body = body // Important: Reset body, so it can be re-read
	return strings.TrimSpace(lines)
}