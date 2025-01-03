package gowsLog

import (
	"fmt"
	waLog "go.mau.fi/whatsmeow/util/log"
	"strings"
)

type stdoutLogger struct {
	mod   string
	color bool
	min   int
}

var colors = map[string]string{
	"INFO":  "\033[36m",
	"WARN":  "\033[33m",
	"ERROR": "\033[31m",
}

var levelToInt = map[string]int{
	"":      -2,
	"TRACE": -1,
	"DEBUG": 0,
	"INFO":  1,
	"WARN":  2,
	"ERROR": 3,
}

func (s *stdoutLogger) outputf(level, msg string, args ...interface{}) {
	if levelToInt[level] < s.min {
		return
	}
	var colorStart, colorReset string
	if s.color {
		colorStart = colors[level]
		colorReset = "\033[0m"
	}
	fmt.Printf("%s%s | [%s] %s%s\n", colorStart, level, s.mod, fmt.Sprintf(msg, args...), colorReset)
}

func (s *stdoutLogger) Errorf(msg string, args ...interface{}) { s.outputf("ERROR", msg, args...) }
func (s *stdoutLogger) Warnf(msg string, args ...interface{})  { s.outputf("WARN", msg, args...) }
func (s *stdoutLogger) Infof(msg string, args ...interface{})  { s.outputf("INFO", msg, args...) }
func (s *stdoutLogger) Debugf(msg string, args ...interface{}) {
	// If mod ends with Send or Recv - increase it to TRACE, too wordy
	if strings.HasSuffix(s.mod, "Send") || strings.HasSuffix(s.mod, "Recv") {
		s.outputf("TRACE", msg, args...)
		return
	}
	s.outputf("DEBUG", msg, args...)
}
func (s *stdoutLogger) Tracef(msg string, args ...interface{}) { s.outputf("TRACE", msg, args...) }
func (s *stdoutLogger) Sub(mod string) waLog.Logger {
	return &stdoutLogger{mod: fmt.Sprintf("%s/%s", s.mod, mod), color: s.color, min: s.min}
}

// Stdout is a simple Logger implementation that outputs to stdout. The module name given is included in log lines.
//
// minLevel specifies the minimum log level to output. An empty string will output all logs.
//
// If color is true, then info, warn and error logs will be colored cyan, yellow and red respectively using ANSI color escape codes.
func Stdout(module string, minLevel string, color bool) waLog.Logger {
	return &stdoutLogger{mod: module, color: color, min: levelToInt[strings.ToUpper(minLevel)]}
}
