package frame

import (
	"encoding/base64"
	"net/http"
)

type Accounts struct {
	//如果认知失败 进行处理 ；默认返回401；使用者可设置专门的处理
	UnAuthHandler func(ctx *Context)
	//用户账号密码
	Users map[string]string
}

//从header中获取base64字符串 进行匹配
//中间件
func (a *Accounts) BasicAuth(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		username, password, ok := ctx.R.BasicAuth()
		if !ok {
			a.UnAuthHandlers(ctx)
			return
		}
		pwd, exist := a.Users[username]
		if !exist {
			a.UnAuthHandlers(ctx)
			return
		}
		if pwd != password {
			a.UnAuthHandlers(ctx)
			return
		}
		ctx.Set("user", username)
		next(ctx)
	}
}

func (a *Accounts) UnAuthHandlers(ctx *Context) {
	if a.UnAuthHandler != nil {
		a.UnAuthHandler(ctx)
	} else {
		ctx.W.WriteHeader(http.StatusUnauthorized)
	}
}
func BasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
