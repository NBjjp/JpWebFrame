package frame

import (
	"errors"
	"github.com/NBjjp/JpWebFrame/binding"
	jplog "github.com/NBjjp/JpWebFrame/log"
	"github.com/NBjjp/JpWebFrame/render"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

const defaultMultipartMemory = 32 << 20 //32M

//封装信息
type Context struct {
	W      http.ResponseWriter
	R      *http.Request
	engine *Engine
	//用于存储参数
	queryCache url.Values
	//
	//Form 用于获取get post put参数
	//PostForm用于获取表单参数
	formCache url.Values
	//状态码
	StatusCode            int
	DisallowUnknownFields bool
	IsValidate            bool
	Logger                *jplog.Logger
	//账号密码 认证服务
	Keys map[string]any
	mu   sync.RWMutex
	//安全性操作
	sameSite http.SameSite
}

func (ctx *Context) SetSameSite(s http.SameSite) {
	ctx.sameSite = s
}
func (ctx *Context) Set(key string, value string) {
	ctx.mu.Lock()
	if ctx.Keys == nil {
		ctx.Keys = make(map[string]any)
	}
	ctx.Keys[key] = value
	ctx.mu.Unlock()
}
func (ctx *Context) SetBasicAuth(username, password string) {
	ctx.R.Header.Set("Authorization", "Basic "+BasicAuth(username, password))
}
func (ctx *Context) Get(key string) (value any, exists bool) {
	ctx.mu.RLock()
	value, exists = ctx.Keys[key]
	ctx.mu.RUnlock()
	return
}

//用于获取参数      ?id=1&name=zhangsan
func (ctx *Context) GetQuery(key string) string {
	ctx.initQueryCache()
	return ctx.queryCache.Get(key)
}

//用于获取参数   一个key对应多个value
func (ctx *Context) GetQueryArray(key string) ([]string, bool) {
	ctx.initQueryCache()
	values, ok := ctx.queryCache[key]
	return values, ok
}

//如果key不存在则返回默认值
func (ctx *Context) GetDefaultQuery(key, defaultValue string) string {
	values, ok := ctx.GetQueryArray(key)
	if !ok {
		return defaultValue
	}
	return values[0]
}

//用于map参数的获取     ?user[id]=1&user[name]=张三  与下面返回值不同
func (c *Context) QueryMap(key string) (dicts map[string]string) {
	dicts, _ = c.GetQueryMap(key)
	return
}

//用于map参数的获取     ?user[id]=1&user[name]=张三
func (ctx *Context) GetQueryMap(key string) (map[string]string, bool) {
	ctx.initQueryCache()
	return ctx.getMap(ctx.queryCache, key)
}

//key 为user   返回[]中的key
func (ctx *Context) getMap(cache map[string][]string, key string) (map[string]string, bool) {
	dicts := make(map[string]string)
	exist := false
	//user[id]=1&user[name]=张三
	for k, value := range cache {
		if i := strings.IndexByte(k, '['); i >= 1 && k[:i] == key {
			if j := strings.IndexByte(k[i+1:], ']'); j >= 1 {

				exist = true
				dicts[k[i+1:j+i+1]] = value[0]
			}
		}
	}
	return dicts, exist
}

//初始化ctx.QueryCache,向其中添加值
func (ctx *Context) initQueryCache() {
	if ctx.R != nil {
		ctx.queryCache = ctx.R.URL.Query() //map[string][]string
	} else {
		ctx.queryCache = url.Values{}
	}
}

//获取表单参数借助 http.Request.PostForm
//Form属性包含了post表单和url后面跟的get参数。
//PostForm属性只包含了post表单参数。
//初始化ctx.FormCache,向其中添加值
func (ctx *Context) initPostFormCache() {
	if ctx.R != nil {
		//ParseMultipartForm 支持传输文件
		if err := ctx.R.ParseMultipartForm(defaultMultipartMemory); err != nil {
			//如果请求时传输文件会报错（http.ErrNotMultipart）
			//忽略该错误
			if !errors.Is(err, http.ErrNotMultipart) {
				log.Println(err)
			}
		}
		ctx.formCache = ctx.R.PostForm //map[string][]string
	} else {
		ctx.formCache = url.Values{}
	}
}

func (ctx *Context) GetPostForm(key string) (string, bool) {
	if values, ok := ctx.GetPostFormArray(key); ok {
		return values[0], ok
	}
	return "", false
}

func (ctx *Context) PostFormArray(key string) (values []string) {
	values, _ = ctx.GetPostFormArray(key)
	return
}

func (ctx *Context) GetPostFormArray(key string) (values []string, ok bool) {
	ctx.initPostFormCache()
	values, ok = ctx.formCache[key]
	return
}

func (ctx *Context) GetPostFormMap(key string) (map[string]string, bool) {
	ctx.initPostFormCache()
	return ctx.getMap(ctx.formCache, key)
}

func (c *Context) PostFormMap(key string) (dicts map[string]string) {
	dicts, _ = c.GetPostFormMap(key)
	return
}

//处理文件参数   借助http.Request.FormFile
func (ctx *Context) FormFile(name string) *multipart.FileHeader {
	file, header, err := ctx.R.FormFile(name)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	return header
}

//简化文件存储需求
func (ctx *Context) SavaUploadFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

//原生框架提供的Form
//type Form struct {
//	Value map[string][]string
//	File  map[string][]*FileHeader
//}
func (ctx *Context) MultipartForm() (*multipart.Form, error) {
	err := ctx.R.ParseMultipartForm(defaultMultipartMemory)
	return ctx.R.MultipartForm, err
}

//将参数解析为JSON结构体
func (ctx *Context) BindJSON(obj any) error {
	json := binding.JSON
	json.DisallowUnknownFields = true
	json.IsValidate = true
	return ctx.MustBindWith(obj, json)
}

//将参数解析为xml
func (ctx *Context) BindXML(obj any) error {
	xml := binding.XML
	return ctx.MustBindWith(obj, xml)
}

func (ctx *Context) MustBindWith(obj any, bind binding.Binding) error {
	if err := ctx.ShouldBind(obj, bind); err != nil {
		ctx.W.WriteHeader(http.StatusBadRequest)
		return err
	}
	return nil
}
func (ctx *Context) ShouldBind(obj any, bind binding.Binding) error {
	return bind.Bind(ctx.R, obj)
}

//gin等框架在做校验时，是使用了`https://github.com/go-playground/validator` 组件，我们也将其集成进来

//原始版本
//func (ctx *Context) HTML(status int, html string) error {
//	//设置头信息中的返回格式
//	ctx.W.Header().Set("Content-Type", "text/html;charset=utf-8")
//	//设置返回状态，不设置的话，如果调用了write方法，默认状态为200
//	ctx.W.WriteHeader(status)
//	_, err := ctx.W.Write([]byte(html))
//	return err
//}

//增加返回状态              TODO
//ParseFiles
func (ctx *Context) HTMLTemplate(name string, data any, filenames ...string) error {
	//设置头信息中的返回格式
	ctx.W.Header().Set("Content-Type", "text/html;charset=utf-8")
	t := template.New(name)
	//ParseFiles方法解析filenames指定的文件里的模板定义并将解析结果与t关联
	t, err1 := t.ParseFiles(filenames...)
	if err1 != nil {
		return err1
	}
	err2 := t.Execute(ctx.W, data)
	return err2
}

//ParseGlob
func (ctx *Context) HTMLTemplateGlob(name string, data any, pattern string) error {
	//设置头信息中的返回格式
	ctx.W.Header().Set("Content-Type", "text/html;charset=utf-8")
	t := template.New(name)
	//ParseGlob方法解析匹配pattern的文件里的模板定义并将解析结果与t关联   统配
	t, err1 := t.ParseGlob(pattern)
	if err1 != nil {
		return err1
	}
	err2 := t.Execute(ctx.W, data)
	return err2
}

//提前将模板加载到内存中    	原始版本
//func (ctx *Context) Template(name string, data any) error {
//	//设置头信息中的返回格式
//	ctx.W.Header().Set("Content-Type", "text/html;charset=utf-8")
//	err := ctx.engine.HTMLRender.Template.ExecuteTemplate(ctx.W, name, data)
//	return err
//}

//返回JSON格式   原始版本
//func (ctx *Context) JSON(status int, data any) error {
//	//设置头信息中的返回格式
//	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
//	ctx.W.WriteHeader(status)
//	jsonData, err := json.Marshal(data)
//	if err != nil {
//		return err
//	}
//	_, err1 := ctx.W.Write(jsonData)
//	return err1
//}

//返回XML格式    原始版本
//func (ctx *Context) XML(status int, data any) error {
//	//设置头信息中的返回格式
//	ctx.W.Header().Set("Content-Type", "application/xml; charset=utf-8")
//	ctx.W.WriteHeader(status)
//	//xmlData, err := xml.Marshal(data)
//	//if err != nil {
//	//	return err
//	//}
//	//_, err1 := ctx.W.Write(xmlData)
//	err := xml.NewEncoder(ctx.W).Encode(data)
//	return err
//}

//返回文件（下载文件）
func (ctx *Context) File(filename string) {
	http.ServeFile(ctx.W, ctx.R, filename)
}

//返回文件（下载文件）   返回的文件名为修改为指定的文件名
func (ctx *Context) FileAttachment(filepath, filename string) {
	if isASCII(filename) {
		ctx.W.Header().Set("Content-Disposition", `attachment;filename="`+filename+`"`)
	} else {
		ctx.W.Header().Set("Content-Disposition", `attachment;filename*=UTF-8''`+url.QueryEscape(filename))
	}
	http.ServeFile(ctx.W, ctx.R, filepath)
}

//从制定的文件系统中下载文件
func (ctx *Context) FileFromFS(filepath string, fs http.FileSystem) {
	defer func(old string) {
		ctx.R.URL.Path = old
	}(ctx.R.URL.Path)
	ctx.R.URL.Path = filepath
	http.FileServer(fs).ServeHTTP(ctx.W, ctx.R)
}

//重定向     原始版本   TODO  缺返回error
//func (ctx *Context) Redirect(status int, url string) {
//	if (status < http.StatusMultipleChoices || status > http.StatusPermanentRedirect) && status != http.StatusCreated {
//		panic(fmt.Sprintf("Cannot redirect with status code %d", status))
//	}
//	http.Redirect(ctx.W, ctx.R, url, status)
//}

//string支持   原始版本
//func (ctx *Context) String(status int, format string, values ...any) error {
//	ctx.W.Header().Set("Content-Type", "text/plain; charset=utf-8")
//	ctx.W.WriteHeader(status)
//	if len(values) > 0 {
//		_, err := fmt.Fprintf(ctx.W, format, values...)
//		return err
//	}
//	_, err1 := ctx.W.Write(StringtiBytes(format))
//	return err1
//}
//多次调用WriteHeader  会产生 http: superfluous response.WriteHeader call  警告  TODO
func (ctx *Context) Render(status int, r render.Render) error {
	ctx.W.WriteHeader(status)
	err := r.Render(ctx.W)
	ctx.StatusCode = status

	return err
}

//string支持   重构版本
func (ctx *Context) String(status int, format string, values ...any) error {
	return ctx.Render(status, &render.String{Format: format, Data: values})
}

//返回XML格式   重构版本
func (ctx *Context) XML(status int, data any) error {
	//设置头信息中的返回格式
	return ctx.Render(status, &render.XML{
		Data: data,
	})
}

//返回JSON格式   重构版本
func (ctx *Context) JSON(status int, data any) error {
	return ctx.Render(status, &render.JSON{
		Data: data,
	})
}

// 重构版本
func (ctx *Context) HTML(status int, html string) error {
	return ctx.Render(status, &render.HTML{
		Data:       html,
		IsTemplate: false,
	})
}

//提前将模板加载到内存中    	重构版本
func (ctx *Context) Template(name string, data any) error {
	return ctx.Render(http.StatusOK, &render.HTML{
		Data:       data,
		Name:       name,
		IsTemplate: true,
		Template:   ctx.engine.HTMLRender.Template,
	})
}

//重定向     重构版本     TODO 重复写writeheader   存在问题  需要修改
func (ctx *Context) Redirect(status int, url string) error {
	return ctx.Render(status, &render.Redirect{
		Code:     status,
		Request:  ctx.R,
		Location: url,
	})
}

//返回错误信息
func (ctx *Context) Fail(code int, obj string) {
	ctx.String(code, obj)
}
func (ctx *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	if path == "" {
		path = "/"
	}
	http.SetCookie(ctx.W, &http.Cookie{
		Name:     name,
		Value:    url.QueryEscape(value),
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		SameSite: ctx.sameSite,
		Secure:   secure,
		HttpOnly: httpOnly,
	})
}
