package frame

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

//Golang标准日志库提供的日志输出方法有Print、Fatal、Panic
//
//- Print用于记录一个普通的程序日志，开发者想记点什么都可以。
//- Fatal用于记录一个导致程序崩溃的日志，并会退出程序。
//- Panic用于记录一个异常日志，并触发panic。
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

var DefaultWriter io.Writer = os.Stdout

type LoggerFormatter func(params *LogFormatterParams) string

//日志配置
type LoggerConfig struct {
	Formatter LoggerFormatter
	out       io.Writer
}

//日志参数
type LogFormatterParams struct {
	Request    *http.Request
	TimeStamp  time.Time
	StatusCode int
	Latency    time.Duration
	ClientIP   net.IP
	Method     string
	Path       string
	//是否显示颜色
	IsDisplayColor bool
}

func (p *LogFormatterParams) StatusCodeColor() string {
	code := p.StatusCode
	switch code {
	case http.StatusOK:
		return green
	default:
		return red
	}
}

func (p *LogFormatterParams) resetColor() string {
	return reset
}

var defaultFormatter = func(params *LogFormatterParams) string {
	//在控制台打印上述日志，并不好查看，如何能带上颜色，看起来就更加明显和明确一些了
	var statusCodeColor = params.StatusCodeColor()
	var resetColor = params.resetColor()
	//如果执行时间超过1一分钟，则转化为显示秒
	if params.Latency > time.Minute {
		params.Latency = params.Latency.Truncate(time.Second)
	}
	if params.IsDisplayColor {
		return fmt.Sprintf("%s [msgo] %s |%s %v %s| %s %3d %s |%s %13v %s| %15s  |%s %-7s %s %s %#v %s",
			yellow, resetColor, blue, params.TimeStamp.Format("2006/01/02 - 15:04:05"), resetColor,
			statusCodeColor, params.StatusCode, resetColor,
			red, params.Latency, resetColor,
			params.ClientIP,
			magenta, params.Method, resetColor,
			cyan, params.Path, resetColor,
		)
	}
	return fmt.Sprintf("[msgo] %v | %s %3d %s | %13v | %15s |%-7s %#v",
		params.TimeStamp.Format("2006/01/02 - 15:04:05"),
		statusCodeColor, params.StatusCode, resetColor,
		params.Latency, params.ClientIP, params.Method, params.Path,
	)

}

//日志打印时间、状态、ip、方法、路径    中间件
func LoggerWithConfig(conf LoggerConfig, next HandlerFunc) HandlerFunc {
	formatter := conf.Formatter
	if formatter == nil {
		formatter = defaultFormatter
	}
	isDisplayColor := false
	out := conf.out
	if out == nil {
		out = DefaultWriter
		//如果输出为标准输出则带颜色
		isDisplayColor = true
	}
	return func(ctx *Context) {
		r := ctx.R
		param := &LogFormatterParams{
			Request:        r,
			IsDisplayColor: isDisplayColor,
		}
		//开始时间
		start := time.Now()
		path := ctx.R.URL.Path
		//请求参数
		raw := ctx.R.URL.RawQuery
		//执行业务
		next(ctx)
		stop := time.Now()
		//执行业务用时  stop-start
		latency := stop.Sub(start)
		//获取ip地址
		ip, _, _ := net.SplitHostPort(strings.TrimSpace(ctx.R.RemoteAddr))
		clientIP := net.ParseIP(ip)
		method := ctx.R.Method
		statusCode := ctx.StatusCode
		if raw != "" {
			path = path + "?" + raw
		}
		param.StatusCode = statusCode
		param.Latency = latency
		param.TimeStamp = stop
		param.Path = path
		param.ClientIP = clientIP
		param.Method = method
		fmt.Fprint(out, formatter(param))
	}
}

func Logging(next HandlerFunc) HandlerFunc {
	return LoggerWithConfig(LoggerConfig{}, next)
}
