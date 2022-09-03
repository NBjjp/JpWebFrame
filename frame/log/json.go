package log

import (
	"encoding/json"
	"fmt"
	"time"
)

type JsonFormatter struct {
	TimeDisplay bool
}

func (f *JsonFormatter) Format(param *LoggingFormatterParam) string {
	if param.LoggerFields == nil {
		param.LoggerFields = make(Fields)
	}
	now := time.Now()
	if f.TimeDisplay {
		timeNow := now.Format("2006/01/02 - 15:04:05")
		param.LoggerFields["log_time"] = timeNow
		param.LoggerFields["log_level"] = param.Level.Level()
	}
	param.LoggerFields["obj"] = param.Obj
	marshal, err := json.Marshal(param.LoggerFields)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s", string(marshal))
}
