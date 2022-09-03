package main

import (
	"errors"
	"fmt"
	"github.com/NBjjp/JpWebFrame"
	"github.com/NBjjp/JpWebFrame/jperror"
	"github.com/NBjjp/JpWebFrame/jppool"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type User struct {
	Name    string   `xml:"name" json:"name"`
	Age     int      `xml:"age" json:"age" validate:"required,max=50,min=18"`
	Address []string `xml:"address" json:"address"`
	Email   string   `json:"email" must:"mustType"`
}

func Log(handlerFunc frame.HandlerFunc) frame.HandlerFunc {
	return func(ctx *frame.Context) {
		fmt.Println("路由中间件")
		handlerFunc(ctx)
		fmt.Println("返回计时")
	}
}

func main() {
	//http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
	//	fmt.Fprintln(w, "hello,路由测试")
	//})
	//err := http.ListenAndServe(":8111", nil)
	//if err != nil {
	//	log.Fatal(err)
	//}
	engine := frame.Default()
	//测试日志中间件
	engine.Use(frame.Logging, frame.Recovery)
	//测试basic64认证
	fmt.Println(frame.BasicAuth("jjp", "123456"))
	auth := &frame.Accounts{
		Users: make(map[string]string),
	}
	auth.Users["jjp"] = "123456"
	engine.Use(auth.BasicAuth)
	g := engine.Group("user")
	//给user路由组添加通用中间件
	g.Use(func(handlerFunc frame.HandlerFunc) frame.HandlerFunc {
		return func(ctx *frame.Context) {
			fmt.Println("前置中间件")
			handlerFunc(ctx)
			fmt.Println("后置中间件")
		}
	})

	g.Get("/hello/get", func(ctx *frame.Context) {
		fmt.Println("传入的函数")
		fmt.Fprintln(ctx.W, "hello,路由测试")
	}, Log)
	g.Post("/hello/*/get", func(ctx *frame.Context) {
		fmt.Fprintln(ctx.W, "hello,路由测试")
	})
	g.Post("/test", func(ctx *frame.Context) {
		fmt.Fprintln(ctx.W, "路由分组测试")
	})

	g.Get("/html", func(ctx *frame.Context) {
		ctx.HTML(http.StatusOK, "<h1>模板映射<h1/>")
	})

	g.Get("/htmlTemplate", func(ctx *frame.Context) {
		err := ctx.HTMLTemplate("login.html", "", "tpl/login.html", "tpl/header.html")
		if err != nil {
			fmt.Println(err)
		}
	})

	g.Get("/htmlTemplateGlob", func(ctx *frame.Context) {
		err := ctx.HTMLTemplateGlob("login.html", "", "tpl/*.html")
		if err != nil {
			fmt.Println(err)
		}
	})
	order := engine.Group("order")
	order.Any("/gets", func(ctx *frame.Context) {
		fmt.Fprintln(ctx.W, "路由分组测试")
	})
	order.Get("/getss/:id", func(ctx *frame.Context) {
		fmt.Fprintln(ctx.W, "id路由分组测试")
	})
	//提前将模板加载到内存当中
	engine.LoadTemplate("tpl/*.html")
	//测试提前将模板加载到内存中
	g.Get("/testtemplate", func(ctx *frame.Context) {
		user := &User{
			Name: "测试JSON",
		}
		err := ctx.Template("login.html", user)
		if err != nil {
			fmt.Println(err)
		}
	})

	//测试输出JSON格式的模板
	g.Get("/json", func(ctx *frame.Context) {
		user := &User{
			Name: "测试JSON",
		}
		err := ctx.JSON(http.StatusOK, user)
		if err != nil {
			fmt.Println(err)
		}
	})

	//测试输出XML格式的模板
	g.Get("/xml", func(ctx *frame.Context) {
		user := &User{
			Name: "测试XML",
		}
		err := ctx.XML(http.StatusOK, user)
		if err != nil {
			fmt.Println(err)
		}
	})

	//测试下载文件
	g.Get("/download", func(ctx *frame.Context) {
		ctx.File("tpl/测试.xlsx")
	})
	//测试下载文件  指定下载文件的名称
	g.Get("/downloadname", func(ctx *frame.Context) {
		ctx.FileAttachment("tpl/测试.xlsx", "aaa.xlsx")
	})

	//从指定的文件系统下载文件
	g.Get("/downloadfs", func(ctx *frame.Context) {
		ctx.FileFromFS("测试.xlsx", http.Dir("tpl"))
	})

	//重定向
	g.Get("/re", func(ctx *frame.Context) {
		ctx.Redirect(http.StatusFound, "/user/testtemplate")
	})
	//测试string
	g.Get("/string", func(ctx *frame.Context) {
		ctx.String(http.StatusFound, "学习- %s -支持,%s", "string", "goweb框架")
	})
	//测试获取url参数   queryCache
	g.Get("/add", func(ctx *frame.Context) {
		ids, ok := ctx.GetQueryArray("ddd")
		fmt.Printf("id:%s,ok:%v \n", ids, ok)
	})
	//测试获取url中的map参数
	g.Get("/querymap", func(ctx *frame.Context) {
		m, _ := ctx.GetQueryMap("user")
		ctx.JSON(http.StatusOK, m)
	})
	//测试获取表单参数
	g.Post("/form", func(ctx *frame.Context) {
		m, _ := ctx.GetPostForm("name")
		ctx.JSON(http.StatusOK, m)
	})
	g.Post("/formarray", func(ctx *frame.Context) {
		m, _ := ctx.GetPostFormArray("name")
		ctx.JSON(http.StatusOK, m)
	})
	g.Post("/formmap", func(ctx *frame.Context) {
		m, _ := ctx.GetPostFormMap("user")
		ctx.JSON(http.StatusOK, m)
	})
	//测试文件获取
	g.Post("/file", func(ctx *frame.Context) {
		m, _ := ctx.GetPostFormMap("user")
		file := ctx.FormFile("file")
		src, err := file.Open()
		if err != nil {
			log.Println(err)
			return
		}
		dst, err := os.Create("./upload/" + file.Filename)
		io.Copy(dst, src)
		ctx.JSON(http.StatusOK, m)
	})
	//测试文件获取简化版
	g.Post("/filesave", func(ctx *frame.Context) {
		m, _ := ctx.GetPostFormMap("user")
		file := ctx.FormFile("file")
		ctx.SavaUploadFile(file, "./upload/"+file.Filename)
		ctx.JSON(http.StatusOK, m)
	})
	//测试文件获取MultipartForm
	g.Post("/savefile", func(ctx *frame.Context) {
		m, _ := ctx.GetPostFormMap("user")
		form, err := ctx.MultipartForm()
		if err != nil {
			fmt.Println(err)
			log.Println(err)
		}
		fileMap := form.File
		headers := fileMap["file"]

		for _, file := range headers {
			fmt.Println(file.Filename)
			ctx.SavaUploadFile(file, "./upload/"+file.Filename)
		}
		ctx.JSON(http.StatusOK, m)
	})
	//测试将参数转化为JSON格式
	g.Post("/jsonParam", func(ctx *frame.Context) {
		//user := &User{}
		user := make([]User, 0)
		ctx.DisallowUnknownFields = true
		ctx.IsValidate = true
		err := ctx.BindJSON(&user)
		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}
	})
	//测试将参数转化为xml格式
	g.Post("/xmlParam", func(ctx *frame.Context) {
		user := &User{}
		err := ctx.BindXML(&user)
		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}
	})
	//测试日志分级

	//json格式化
	//engine.Logger.Formatter = &jplog.JsonFormatter{
	//	TimeDisplay: true,
	//}

	engine.Logger.Level = 0
	//logger.Outs = append(logger.Outs, jplog.FileWriter("./log/log.log"))
	engine.Logger.SetLogPath("./log")
	engine.Logger.LogFileSize = 1 << 5
	g.Post("/loglevel", func(ctx *frame.Context) {
		user := &User{}
		//测试日志分级

		//ctx.Logger.WithFields(jplog.Fields{
		//	"name": "马神之路",
		//	"id":   1000,
		//}).Debug("我是debug日志")
		//ctx.Logger.Info("我是info日志")
		//ctx.Logger.Error("我是error日志")
		err := jperror.Default()
		a(&jperror.JpError{})
		err.Result(func(jpError *jperror.JpError) {
			log.Println(jpError.Error())
			ctx.JSON(http.StatusInternalServerError, user)
		})
		err1 := ctx.BindXML(&user)
		if err1 == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}
	})
	//测试协程池
	p, _ := jppool.NewPool(2)
	g.Post("/pool", func(ctx *frame.Context) {
		currentTime := time.Now().UnixMilli()
		var wg sync.WaitGroup
		wg.Add(5)
		p.Submit(func() {
			fmt.Println("1111111111")
			time.Sleep(2 * time.Second)
			wg.Done()
		})
		p.Submit(func() {
			fmt.Println("2222222222")
			time.Sleep(2 * time.Second)
			wg.Done()
		})
		p.Submit(func() {
			fmt.Println("3333333333")
			time.Sleep(2 * time.Second)
			wg.Done()
		})
		p.Submit(func() {
			fmt.Println("444444444444")
			time.Sleep(2 * time.Second)
			wg.Done()
		})
		p.Submit(func() {
			fmt.Println("5555555555555")
			time.Sleep(2 * time.Second)
			wg.Done()
		})
		wg.Wait()
		fmt.Println("time: %v \n", time.Now().UnixMilli()-currentTime)
		ctx.JSON(http.StatusOK, "success")
	})
	//engine.Run()
	engine.RunTLS(":8118", "key/server.pem", "key/server.key")
}

func a(jpError *jperror.JpError) {
	err := errors.New("sda")
	jpError.Put(err)
}
