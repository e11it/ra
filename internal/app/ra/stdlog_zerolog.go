package ra

import (
	stdlog "log"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// StdLogWriter перенаправляет сообщения stdlib `log` в zerolog (JSON).
type StdLogWriter struct {
	Component string
	Level     zerolog.Level
}

// Write implements io.Writer for use with stdlog.New.
func (w StdLogWriter) Write(p []byte) (n int, err error) {
	s := strings.TrimSpace(string(p))
	if s == "" {
		return len(p), nil
	}
	ev := log.WithLevel(w.Level)
	if w.Component != "" {
		ev = ev.Str("component", w.Component)
	}
	ev.Msg(s)
	return len(p), nil
}

// newStdToZerolog returns *stdlog.Logger for APIs that only accept that (httputil.ReverseProxy, http.Server).
func newStdToZerolog(component string, level zerolog.Level) *stdlog.Logger {
	return stdlog.New(StdLogWriter{Component: component, Level: level}, "", 0)
}

// ReverseProxyErrorLog is httputil.ReverseProxy.ErrorLog — JSON in stderr via zerolog.
func ReverseProxyErrorLog() *stdlog.Logger {
	return newStdToZerolog("httputil.ReverseProxy", zerolog.WarnLevel)
}

// HTTPServerErrorLog is http.Server.ErrorLog (e.g. "net/http: abort Handler").
func HTTPServerErrorLog() *stdlog.Logger {
	return newStdToZerolog("net/http.Server", zerolog.WarnLevel)
}
