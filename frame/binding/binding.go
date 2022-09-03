package binding

import "net/http"

//实现绑定器  将json参数 xml参数 等其他参数的处理抽象为接口，赋予不同的实现，方便维护
type Binding interface {
	Name() string
	Bind(*http.Request, any) error
}

var JSON = jsonBinding{}
var XML = xmlBinding{}
