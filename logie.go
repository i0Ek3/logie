package logie

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

var std = New()

const (
	FmtEmptySeparate = ""
)

type (
	//
	Level uint8

	//
	Option func(*options)
)

const (
	TraceLevel Level = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	PanicLevel
	FatalLevel
)

var LevelMapping = map[Level]string{
	TraceLevel: "Trace",
	DebugLevel: "Debug",
	InfoLevel:  "Info",
	WarnLevel:  "Warn",
	ErrorLevel: "Error",
	PanicLevel: "Panic",
	FatalLevel: "Fatal",
}

type Logger struct {
	opt       *options
	mu        sync.Mutex
	entryPool *sync.Pool
}

func New(opts ...Option) *Logger {
	logger := &Logger{opt: initOptions(opts...)}
	logger.entryPool = &sync.Pool{New: func() interface{} {
		return entry(logger)
	}}
	return logger
}

func StdLogger() *Logger {
	return std
}

func SetOptions(opts ...Option) {
	std.SetOptions(opts...)
}

func (l *Logger) SetOptions(opts ...Option) {
	l.mu.Lock()
	for _, opt := range opts {
		opt(l.opt)
	}
	l.mu.Unlock()
}

func Writer() io.Writer {
	return std
}

func (l *Logger) Writer() io.Writer {
	return l
}

func (l *Logger) Write(data []byte) (int, error) {
	l.entry().write(l.opt.stdLevel, FmtEmptySeparate, *(*string)(unsafe.Pointer(&data)))
	return 0, nil
}

func (l *Logger) entry() *Entry {
	return l.entryPool.Get().(*Entry)
}

func (l *Logger) Debug(args ...interface{}) {
	l.entry().write(DebugLevel, FmtEmptySeparate, args...)
}

func (l *Logger) Info(args ...interface{}) {
	l.entry().write(InfoLevel, FmtEmptySeparate, args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.entry().write(WarnLevel, FmtEmptySeparate, args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.entry().write(ErrorLevel, FmtEmptySeparate, args...)
}

func (l *Logger) Panic(args ...interface{}) {
	l.entry().write(PanicLevel, FmtEmptySeparate, args...)
	panic(fmt.Sprint(args...))
}

func (l *Logger) Fatal(args ...interface{}) {
	l.entry().write(FatalLevel, FmtEmptySeparate, args...)
	os.Exit(1)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.entry().write(DebugLevel, format, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.entry().write(InfoLevel, format, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.entry().write(WarnLevel, format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.entry().write(ErrorLevel, format, args...)
}

func (l *Logger) Panicf(format string, args ...interface{}) {
	l.entry().write(PanicLevel, format, args...)
	panic(fmt.Sprintf(format, args...))
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.entry().write(FatalLevel, format, args...)
	os.Exit(1)
}

// std logger
func Debug(args ...interface{}) {
	std.entry().write(DebugLevel, FmtEmptySeparate, args...)
}

func Info(args ...interface{}) {
	std.entry().write(InfoLevel, FmtEmptySeparate, args...)
}

func Warn(args ...interface{}) {
	std.entry().write(WarnLevel, FmtEmptySeparate, args...)
}

func Error(args ...interface{}) {
	std.entry().write(ErrorLevel, FmtEmptySeparate, args...)
}

func Panic(args ...interface{}) {
	std.entry().write(PanicLevel, FmtEmptySeparate, args...)
	panic(fmt.Sprint(args...))
}

func Fatal(args ...interface{}) {
	std.entry().write(FatalLevel, FmtEmptySeparate, args...)
	os.Exit(1)
}

func Debugf(format string, args ...interface{}) {
	std.entry().write(DebugLevel, format, args...)
}

func Infof(format string, args ...interface{}) {
	std.entry().write(InfoLevel, format, args...)
}

func Warnf(format string, args ...interface{}) {
	std.entry().write(WarnLevel, format, args...)
}

func Errorf(format string, args ...interface{}) {
	std.entry().write(ErrorLevel, format, args...)
}

func Panicf(format string, args ...interface{}) {
	std.entry().write(PanicLevel, format, args...)
	panic(fmt.Sprintf(format, args...))
}

func Fatalf(format string, args ...interface{}) {
	std.entry().write(FatalLevel, format, args...)
	os.Exit(1)
}

type Entry struct {
	logger *Logger
	Buf    *bytes.Buffer
	Map    map[string]interface{}
	Level  Level
	Time   time.Time
	File   string
	Line   int
	Func   string
	Format string
	Args   []interface{}
}

func entry(logger *Logger) *Entry {
	return &Entry{
		logger: logger,
		Buf:    new(bytes.Buffer),
		Map:    make(map[string]interface{}, 5),
	}
}

func (e *Entry) write(lvl Level, format string, args ...interface{}) {
	if e.logger.opt.level > lvl {
		return
	}
	e.Time = time.Now()
	e.Level = lvl
	e.Format = format
	e.Args = args

	// TODO
	if !e.logger.opt.enableCaller {
		if pc, file, line, ok := runtime.Caller(2); !ok {
			e.File = "unknown"
			e.Func = "unknown"
		} else {
			e.File, e.Line, e.Func = file, line, runtime.FuncForPC(pc).Name()
			e.Func = e.Func[strings.LastIndex(e.Func, "/")+1:]
		}
	}

	e.format()
	e.writer()
	e.release()
}

func (e *Entry) format() {
	_ = e.logger.opt.formatter.Format(e)
}

func (e *Entry) writer() {
	e.logger.mu.Lock()
	_, _ = e.logger.opt.position.Write(e.Buf.Bytes())
	e.logger.mu.Unlock()
}

func (e *Entry) release() {
	e.Args, e.Line, e.File, e.Format, e.Func = nil, 0, "", "", ""
	e.Buf.Reset()
	e.logger.entryPool.Put(e)
}

type Formatter interface {
	Format(entry *Entry) error
}

type TextFormatter struct {
	IgnoreBasicFields bool
}

func (f *TextFormatter) Format(e *Entry) error {
	if !f.IgnoreBasicFields {
		e.Buf.WriteString(fmt.Sprintf("%s %s", e.Time.Format(time.RFC3339), LevelMapping[e.Level])) // allocs
		if e.File != "" {
			short := e.File
			for i := len(e.File) - 1; i > 0; i-- {
				if e.File[i] == '/' {
					short = e.File[i+1:]
					break
				}
			}
			e.Buf.WriteString(fmt.Sprintf(" %s:%d", short, e.Line))
		}
		e.Buf.WriteString(" ")
	}

	switch e.Format {
	case FmtEmptySeparate:
		e.Buf.WriteString(fmt.Sprint(e.Args...))
	default:
		e.Buf.WriteString(fmt.Sprintf(e.Format, e.Args...))
	}
	e.Buf.WriteString("\n")

	return nil
}

type JSONFormatter struct {
	IgnoreBasicFields bool
}

func (f *JSONFormatter) Format(e *Entry) error {
	if !f.IgnoreBasicFields {
		e.Map["level"] = LevelMapping[e.Level]
		e.Map["time"] = e.Time.Format(time.RFC3339)
		if e.File != "" {
			e.Map["file"] = e.File + ":" + strconv.Itoa(e.Line)
			e.Map["func"] = e.Func
		}

		switch e.Format {
		case FmtEmptySeparate:
			e.Map["message"] = fmt.Sprint(e.Args...)
		default:
			e.Map["message"] = fmt.Sprintf(e.Format, e.Args...)
		}

		return jsoniter.NewEncoder(e.Buf).Encode(e.Map)
	}

	switch e.Format {
	case FmtEmptySeparate:
		for _, arg := range e.Args {
			if err := jsoniter.NewEncoder(e.Buf).Encode(arg); err != nil {
				return err
			}
		}
	default:
		e.Buf.WriteString(fmt.Sprintf(e.Format, e.Args...))
	}

	return nil
}

type options struct {
	position     io.Writer
	level        Level
	stdLevel     Level
	formatter    Formatter
	enableCaller bool
}

func initOptions(opts ...Option) *options {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	if o.position == nil {
		o.position = os.Stderr
	}

	if o.formatter == nil {
		o.formatter = &TextFormatter{}
	}
	return o
}

func WithPosition(pos io.Writer) Option {
	return func(o *options) {
		o.position = pos
	}
}

func WithLevel(lvl Level) Option {
	return func(o *options) {
		o.level = lvl
	}
}

func WithStdLevel(lvl Level) Option {
	return func(o *options) {
		o.stdLevel = lvl
	}
}

func WithFormatter(fmt Formatter) Option {
	return func(o *options) {
		o.formatter = fmt
	}
}

func WithEnableCaller(caller bool) Option {
	return func(o *options) {
		o.enableCaller = caller
	}
}

var errUnmarshalNilLevel = errors.New("cannot unmarshal nil *Level")

func (l *Level) unmarshalText(text []byte) bool {
	switch string(text) {
	case "trace", "Trace":
		*l = TraceLevel
	case "debug", "Debug":
		*l = DebugLevel
	case "info", "Info":
		*l = InfoLevel
	case "warn", "Warn":
		*l = WarnLevel
	case "error", "Error":
		*l = ErrorLevel
	case "panic", "Panic":
		*l = PanicLevel
	case "fatal", "Fatal":
		*l = FatalLevel
	default:
		return false
	}
	return true
}

func (l *Level) UnmarshalText(text []byte) error {
	if l == nil {
		return errUnmarshalNilLevel
	}

	if !l.unmarshalText(text) && !l.unmarshalText(bytes.ToLower(text)) {
		return fmt.Errorf("unexpected level: %q", text)
	}
	return nil
}
