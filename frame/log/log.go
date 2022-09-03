package log

import (
	"fmt"
	"github.com/NBjjp/JpWebFrame/internal/jpstrings"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

const (
	greenBg   = "\033[97;42m"
	whiteBg   = "\033[90;47m"
	yellowBg  = "\033[90;43m"
	redBg     = "\033[97;41m"
	blueBg    = "\033[97;44m"
	magentaBg = "\033[97;45m"
	cyanBg    = "\033[97;46m"
	green     = "\033[32m"
	white     = "\033[37m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
	reset     = "\033[0m"
)

//实现分级日志
const (
	LevelDebug LoggerLevel = iota
	LevelInfo
	LevelError
)

//格式化日志接口
type LoggingFormatter interface {
	Format(param *LoggingFormatterParam) string
}

//增加分级日志字段信息
type Fields map[string]any

type LoggerLevel int //日志级别

type Logger struct {
	Formatter    LoggingFormatter
	Level        LoggerLevel
	Outs         []*LoggerWriter //输出   可以放置多种输出的方式
	LoggerFields Fields
	LogPath      string
	LogFileSize  int64
}
type LoggerWriter struct {
	Level LoggerLevel
	Out   io.Writer
}

func (level LoggerLevel) Level() string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	default:
		return ""
	}
}

type LoggingFormatterParam struct {
	Level        LoggerLevel
	IsColor      bool
	LoggerFields Fields
	Obj          any
}

//格式化打印日志
type LoggerFormatter struct {
	Level        LoggerLevel
	IsColor      bool
	LoggerFields Fields
}

//设置默认值
func Default() *Logger {
	logger := New()
	logger.Level = LevelDebug
	w := &LoggerWriter{
		Level: LevelDebug,
		Out:   os.Stdout,
	}
	logger.Outs = append(logger.Outs, w)
	logger.Formatter = &TextFormatter{}
	return logger
}

func New() *Logger {
	return &Logger{}
}

//向结构体添 额外字段
func (l *Logger) WithFields(fields Fields) *Logger {
	return &Logger{
		Formatter:    l.Formatter,
		Outs:         l.Outs,
		Level:        l.Level,
		LoggerFields: fields,
	}
}

func (l *Logger) Info(obj any) {
	l.Print(LevelInfo, obj)
}
func (l *Logger) Debug(obj any) {
	l.Print(LevelDebug, obj)
}
func (l *Logger) Error(obj any) {
	l.Print(LevelError, obj)
}
func (l *Logger) Print(level LoggerLevel, obj any) {
	//当前的级别大于输入级别，不打印对应的级别日志
	if l.Level > level {
		return
	}
	param := &LoggingFormatterParam{
		Level:        level,
		LoggerFields: l.LoggerFields,
		Obj:          obj,
	}
	str := l.Formatter.Format(param)
	//存在问题   输出方式队列中最初包含一种输出方式    TODO
	for _, out := range l.Outs {
		if out.Out == os.Stdout {
			param.IsColor = true
			str = l.Formatter.Format(param)
			_, _ = fmt.Fprintln(out.Out, str)
		}
		if out.Level == -1 || out.Level == level {
			_, _ = fmt.Fprintln(out.Out, str)
			l.CheckFileSize(out)
		}

	}
}

//格式化打印日志
func (f *LoggerFormatter) format(obj any) string {
	now := time.Now()
	if f.IsColor {
		//要带颜色  error的颜色 为红色 info为绿色 debug为蓝色
		levelColor := f.LevelColor()
		msgColor := f.ObjColor()
		return fmt.Sprintf("%s [msgo] %s %s%v%s | level= %s %s %s | msg=%s %#v %s | fields=%#v",
			yellow, reset, blue, now.Format("2006/01/02 - 15:04:05"), reset,
			levelColor, f.Level.Level(), reset, msgColor, obj, reset, f.LoggerFields,
		)
	}
	return fmt.Sprintf("[msgo] %v | level=%s | msg=%#v | fields=%#v",
		now.Format("2006/01/02 - 15:04:05"),
		f.Level.Level(), obj, f.LoggerFields,
	)
}

func (f *LoggerFormatter) LevelColor() string {
	switch f.Level {
	case LevelDebug:
		return blue
	case LevelInfo:
		return green
	case LevelError:
		return red
	default:
		return cyan
	}
}
func (f *LoggerFormatter) ObjColor() string {
	switch f.Level {
	case LevelDebug:
		return ""
	case LevelInfo:
		return ""
	case LevelError:
		return red
	default:
		return cyan
	}
}

//实现日志的文件存储
func FileWriter(name string) (io.Writer, error) {
	w, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	return w, err
}

func (l *Logger) SetLogPath(logPath string) {
	l.LogPath = logPath
	//写入文件
	all, err := FileWriter(path.Join(l.LogPath, "all.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, &LoggerWriter{Level: -1, Out: all})
	debug, err := FileWriter(path.Join(l.LogPath, "debug.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, &LoggerWriter{Level: LevelDebug, Out: debug})
	info, err := FileWriter(path.Join(l.LogPath, "info.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, &LoggerWriter{Level: LevelInfo, Out: info})
	logError, err := FileWriter(path.Join(l.LogPath, "error.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, &LoggerWriter{Level: LevelError, Out: logError})
}

//根据文件大小切分文件
func (l *Logger) CheckFileSize(out *LoggerWriter) {
	osFile := out.Out.(*os.File)
	if osFile != nil {
		stat, err := osFile.Stat()
		if err != nil {
			log.Println("logger checkFileSize error info :", err)
			return
		}
		size := stat.Size()
		//检查大小，如果满足条件 就重新创建文件，并且更换logger中的输出
		if l.LogFileSize <= 0 {
			l.LogFileSize = 100 << 20
		}
		if size >= l.LogFileSize {
			_, filename := path.Split(osFile.Name())
			name := filename[0:strings.Index(filename, ".")]
			w, err := FileWriter(path.Join(l.LogPath, jpstrings.JoinStrings(name, ".", time.Now().UnixMilli(), ".log")))
			if err != nil {
				log.Println("logger checkFileSize error info :", err)
				return
			}
			out.Out = w
		}
	}
}
