package cos

import (
	"fmt"
	"io"
	"log"
	"os"
)

// ------------------------------------------------------------
// Exported symbols

// Minimalist logger inspired by:
// - https://dave.cheney.net/2015/11/05/lets-talk-about-logging
// - https://dave.cheney.net/2017/01/23/the-package-level-logger-anti-pattern
type Logger interface {
	Info(a ...interface{})
	Detail(a ...interface{})
}

func NewLogger(verbose bool) Logger {
	if verbose {
		return verboseLogger{ infoLogger {out: os.Stderr} }
	}
	return terseLogger{ infoLogger {out: os.Stderr} }
}

// ------------------------------------------------------------
// Unexported symbols

// ------------------------------
// infoLogger

// Partial base Logger implementation
type infoLogger struct {
	out io.Writer
}

// Logger.Info() implementation: log to stderr
func (l infoLogger) Info(a ...interface{}) {
	_, err := fmt.Fprintln(l.out, a...)
	if err != nil {
		log.Fatal(err)
	}
}

// ------------------------------
// terseLogger

// Logger implementation with no-op Detail()
type terseLogger struct {
	infoLogger
}

// No-op Logger.Detail() impelementation
func (l terseLogger) Detail(a ...interface{}) {
	// does nothing
}

// ------------------------------
// verboseLogger

// Logger implementation forwarding Detail() to Info()
type verboseLogger struct {
	infoLogger
}

// Logger.Detail() implementation: forward to Info()
func (l verboseLogger) Detail(a ...interface{}) {
	l.Info(a...)
}
