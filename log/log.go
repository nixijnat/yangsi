package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

func init() {
	_, err := os.Stat(logDir)
	if err != nil {
		if !os.IsNotExist(err) {
			ErrorLog("log dir failed: %s, %s", logDir, err.Error())
			os.Exit(1)
		}
		err = os.MkdirAll(logDir, os.ModePerm)
		if err != nil {
			ErrorLog("mkdir failed: %s", logDir)
			os.Exit(1)
		}
	}
	filename := fmt.Sprintf("%s/%s.log", logDir, time.Now().Format("20060102"))
	fileWriter, err = os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		ErrorLog("open log file failed: %s, %s", filename, err.Error())
		os.Exit(1)
	}
	stdWriter = os.Stdout
	InfoLog("============================ start =========================")
}

var (
	logDir = "./log"
)

const (
	UNKNOWN Level = iota
	ERROR
	WARN
	INFO
	REALTIME
)

type Level uint

func (l Level) String() string {
	switch l {
	case ERROR:
		return "ERROR"
	case WARN:
		return "WARN"
	case INFO:
		return "INFO"
	case REALTIME:
		return "REALTIME"
	default:
		return "UNKNOWN"
	}
}

type env struct {
	file string
	fn   string
	line int
}

func (e *env) String() string {
	return fmt.Sprintf("%s.%s:%d", e.file, e.fn, e.line)
}

func getEnv(skip int) env {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return env{}
	}
	i := strings.LastIndex(file, "src/")
	return env{
		file: string(file[i+4:]),
		fn:   runtime.FuncForPC(pc).Name(),
		line: line,
	}
}

var (
	fileWriter io.Writer
	stdWriter  io.Writer
)

func write(l Level, log []byte) {
	if fileWriter != nil && l != REALTIME {
		fileWriter.Write(log)
	}
	stdWriter.Write(log)
}

func fmtLog(e env, l Level, str string) string {
	return fmt.Sprintf("%s [%s] %s %s\r\n", time.Now().Format("2006-01-02 15:04:05"), l.String(), e.String(), str)
}

func ErrorLog(format string, args ...interface{}) {
	write(ERROR, []byte(fmtLog(getEnv(2), ERROR, fmt.Sprintf(format, args...))))
}

func WarnLog(format string, args ...interface{}) {
	write(WARN, []byte(fmtLog(getEnv(2), WARN, fmt.Sprintf(format, args...))))
}

func InfoLog(format string, args ...interface{}) {
	write(INFO, []byte(fmtLog(getEnv(2), INFO, fmt.Sprintf(format, args...))))
}

func RealtimeLog(format string, args ...interface{}) {
	write(REALTIME, []byte(fmtLog(getEnv(2), REALTIME, fmt.Sprintf(format, args...))))
}

///////////////

type localError struct {
	e   env
	l   Level
	str string
}

func (e localError) Error() string {
	return fmtLog(e.e, e.l, e.str)
}

func wrapError(l Level, str string) error {
	return localError{
		e:   getEnv(3),
		l:   l,
		str: str,
	}
}

func NewError(format string, args ...interface{}) error {
	return wrapError(ERROR, fmt.Sprintf(format, args...))
}

func NewWarn(format string, args ...interface{}) error {
	return wrapError(WARN, fmt.Sprintf(format, args...))
}

func NewInfo(format string, args ...interface{}) error {
	return wrapError(INFO, fmt.Sprintf(format, args...))
}

func NewRealtime(format string, args ...interface{}) error {
	return wrapError(REALTIME, fmt.Sprintf(format, args...))
}

func WriteError(e error, format string, args ...interface{}) {
	var str string
	if format != "" {
		str = fmt.Sprintf(format, args...) + ": " + e.Error()
	} else {
		str = e.Error()
	}
	err, ok := e.(localError)
	if !ok {
		write(UNKNOWN, []byte(fmtLog(getEnv(2), UNKNOWN, str)))
		return
	}
	write(err.l, []byte(str))
}
