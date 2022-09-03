package log

import (
	"fmt"
	"strings"
	"time"
)

type TextFormatter struct {
}

func (f *TextFormatter) Format(param *LoggingFormatterParam) string {
	now := time.Now()
	fieldsString := ""
	if param.LoggerFields != nil {
		var sb strings.Builder
		var count = 0
		var lens = len(param.LoggerFields)
		for k, v := range param.LoggerFields {
			fmt.Fprintf(&sb, "%s=%v", k, v)
			if count < lens-1 {
				fmt.Fprintf(&sb, ",")
			}
			count++
		}
		fieldsString = sb.String()
	}
	var jpinfo = "\n jpgo:"
	if param.Level == LevelError {
		jpinfo = "\n Error Cause By:"
	}
	if param.IsColor {
		//要带颜色  error的颜色 为红色 info为绿色 debug为蓝色
		levelColor := f.LevelColor(param)
		msgColor := f.ObjColor(param)
		return fmt.Sprintf("%s [msgo] %s %s%v%s | level= %s %s %s %s %s %v %s | fields=%#v",
			yellow, reset, blue, now.Format("2006/01/02 - 15:04:05"), reset,
			levelColor, param.Level.Level(), reset, msgColor, jpinfo, param.Obj, reset, fieldsString,
		)
	}
	return fmt.Sprintf("[msgo] %v | level=%s %s %#v | fields=%v",
		now.Format("2006/01/02 - 15:04:05"),
		param.Level.Level(), jpinfo, param.Obj, fieldsString,
	)
}

func (f *TextFormatter) LevelColor(param *LoggingFormatterParam) string {
	switch param.Level {
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
func (f *TextFormatter) ObjColor(param *LoggingFormatterParam) string {
	switch param.Level {
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
