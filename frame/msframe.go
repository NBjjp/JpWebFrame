package frame

import (
	"fmt"
	"github.com/NBjjp/JpWebFrame/config"
	jplog "github.com/NBjjp/JpWebFrame/log"
	"github.com/NBjjp/JpWebFrame/render"
	"html/template"
	"log"
	"net/http"
	"sync"
)

const ANY = "ANY"

type HandlerFunc func(ctx *Context)

//中间件
type MiddlewareFunc func(handlerFunc HandlerFunc) HandlerFunc

//路由组   各个接口的路由功能放在一个组中
type routerGroup struct {
	//组名
	name string
	//第一个string为路由 第二个string为请求方式
	handleFuncMap map[string]map[string]HandlerFunc
	//为每个函数添加对应的中间件
	middlewareFuncMap map[string]map[string][]MiddlewareFunc

	//第一个string对应请求方式  第二个string对应路由
	//handlerMethodMap map[string][]string
	treeNode *treeNode
	//通用中间件
	middlewares []MiddlewareFunc
}

//向结构体中添加中间件
func (r *routerGroup) Use(middlewareFunc ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewareFunc...)
}

//在处理器执行前后添加中间件
func (r *routerGroup) methodHandle(name string, method string, h HandlerFunc, ctx *Context) {
	//通用中间件
	if r.middlewares != nil {
		for _, middlewareFunc := range r.middlewares {
			h = middlewareFunc(h)
		}
	}
	//路由中间件
	if r.middlewareFuncMap[name][method] != nil {
		for _, middlewareFunc := range r.middlewareFuncMap[name][method] {
			h = middlewareFunc(h)
		}
	}
	h(ctx)
}

//func (r *routergroup) Add(name string, handleFunc HandleFunc) {
//	r.handleFuncMap[name] = handleFunc
//}

func (r *routerGroup) handle(name string, method string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	_, ok := r.handleFuncMap[name]
	if !ok {
		r.handleFuncMap[name] = make(map[string]HandlerFunc)
		r.middlewareFuncMap[name] = make(map[string][]MiddlewareFunc)
	}
	_, ok = r.handleFuncMap[name][method]
	if ok {
		panic("存在重复路由")
	}
	//不存在则增加路由
	r.handleFuncMap[name][method] = handleFunc
	r.middlewareFuncMap[name][method] = append(r.middlewareFuncMap[name][method], middlewareFunc...)
	//添加到前缀树中
	r.treeNode.Put(name)
}

//处理任何访问方式  get post。。
func (r *routerGroup) Any(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, ANY, handleFunc, middlewareFunc...)
}

//处理Get请求方式
func (r *routerGroup) Get(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodGet, handlerFunc, middlewareFunc...)
}

//处理POST请求方式
func (r *routerGroup) Post(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPost, handlerFunc, middlewareFunc...)
}

func (r *routerGroup) Delete(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodDelete, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Put(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPut, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Patch(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPatch, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Options(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodOptions, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Head(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodHead, handlerFunc, middlewareFunc...)
}

type router struct {
	routergroups []*routerGroup
	engine       *Engine
}

//初始化路由组
func (r *router) Group(name string) *routerGroup {
	routergroup := &routerGroup{
		name:              name,
		handleFuncMap:     make(map[string]map[string]HandlerFunc),
		middlewareFuncMap: make(map[string]map[string][]MiddlewareFunc),
		//handlerMethodMap: make(map[string][]string),
		treeNode: &treeNode{name: "/", children: make([]*treeNode, 0)},
	}
	routergroup.Use(r.engine.Middles...)
	r.routergroups = append(r.routergroups, routergroup)
	return routergroup
}

type Engine struct {
	router
	funcMap    template.FuncMap
	HTMLRender render.HTMLRender
	pool       sync.Pool
	Logger     *jplog.Logger
	Middles    []MiddlewareFunc
}

//sync.Pool用于存储那些被分配了但是没有被使用，但是未来可能被使用的值，这样可以不用再次分配内存，提高效率。
//sync.Pool大小是可伸缩的，高负载是会动态扩容，存放在池中不活跃的对象会被自动清理。
func New() *Engine {
	engine := &Engine{
		router:     router{},
		funcMap:    nil,
		HTMLRender: render.HTMLRender{},
	}
	engine.pool.New = func() any {
		return engine.allocateContext()
	}
	return engine
}
func Default() *Engine {
	engine := New()
	engine.router.engine = engine
	engine.Logger = jplog.Default()
	engine.Use(Logging, Recovery)
	logpath, ok := config.Conf.Log["path"]
	if ok {
		engine.Logger.SetLogPath(logpath.(string))
	}
	return engine
}

func (e *Engine) allocateContext() any {
	return &Context{engine: e}
}

func (e *Engine) SetFuncMap(funcmap template.FuncMap) {
	e.funcMap = funcmap
}

//将模板提前加载到内存中
func (e *Engine) LoadTemplate(pattern string) {
	t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
	//e.HTMLRender = render.HTMLRender{Template: t}
	e.SetHTMLTemplate(t)
}

//通过toml配置文件配置实现将模板提前加载到内存中
func (e *Engine) LoadTemplateConf() {
	pattern, ok := config.Conf.Template["template"]
	if ok {
		t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern.(string)))
		//e.HTMLRender = render.HTMLRender{Template: t}
		e.SetHTMLTemplate(t)
	}

}
func (e *Engine) SetHTMLTemplate(t *template.Template) {
	e.HTMLRender = render.HTMLRender{
		Template: t,
	}
}
func (e *Engine) httpRequestHandle(ctx *Context, w http.ResponseWriter, r *http.Request) {
	method := r.Method
	for _, group := range e.routergroups {
		routerName := SubStringLast(r.URL.Path, "/"+group.name)
		node := group.treeNode.Get(routerName)
		if node != nil && node.isEnd {
			//路由匹配
			//ctx := &Context{
			//	W:      w,
			//	R:      r,
			//	engine: e,
			//}
			handle, ok := group.handleFuncMap[node.routerName][ANY]
			if ok {
				group.methodHandle(node.routerName, ANY, handle, ctx)
				return
			}
			//如果any不匹配则与对应的方法进行匹配
			handle, ok = group.handleFuncMap[node.routerName][method]
			if ok {
				group.methodHandle(node.routerName, method, handle, ctx)
				return
			}
			//路径一样，请求方式没有，返回405状态，
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "%s %s 请求方式不允许\n", r.RequestURI, method)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "%s  not found\n", r.RequestURI)
}

//实现handler接口中serveHTTP方法
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := e.pool.Get().(*Context)
	ctx.W = w
	ctx.R = r
	ctx.Logger = e.Logger
	e.httpRequestHandle(ctx, w, r)
	e.pool.Put(ctx)
}
func (e *Engine) RunTLS(addr, certFile, keyFile string) {
	err := http.ListenAndServeTLS(addr, certFile, keyFile, e.Handler())
	if err != nil {
		log.Fatal(err)
	}
}
func (e *Engine) Run(addr string) {
	//for _, group := range e.routergroups {
	//	//group:user key:get value:func
	//	for key, value := range group.handleFuncMap {
	//		http.HandleFunc("/"+group.name+key, value)
	//	}
	//}
	http.Handle("/", e)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

//引擎中间件   添加日志中间件
func (e *Engine) Use(middles ...MiddlewareFunc) {
	e.Middles = append(e.Middles, middles...)
}

func (e *Engine) Handler() http.Handler {
	return e
}
