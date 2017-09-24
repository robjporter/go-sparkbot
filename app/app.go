package app

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	stdContext "context"

	"github.com/Sirupsen/logrus"
	"github.com/betacraft/yaag/irisyaag"
	"github.com/betacraft/yaag/yaag"
	prometheusMiddleware "github.com/iris-contrib/middleware/prometheus"
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
	"github.com/robjporter/go-utils/filesystem/config"
	"github.com/robjporter/go-utils/filesystem/path"
)

var _init_ctx sync.Once
var _instance *Application

func init() {
	a := New()
	a.conf = config.New()
	a.Log = logrus.New()
	a.Server = iris.New()
	numCPU := runtime.NumCPU()
	a.Log.Info("Initialising application...")
	a.setDefaultsConfig()
	a.conf.Set("runtime.nunmcpu", numCPU)
	a.conf.Set("runtime.numgoroutine", runtime.NumGoroutine())
	a.conf.Set("runtime.version", runtime.Version())
	runtime.GOMAXPROCS(numCPU)
	a.setServerConfig()
	a.Log.Info("Application initialised successfully...")
}

// Set Config defaults before anything has a chance to be set
func (a Application) setDefaultsConfig() {
	a.conf.Set("application.name", "Default")
	a.conf.Set("application.version", "1")
	a.conf.Set("application.debug", false)
	a.conf.Set("server.port", 9090)
	a.conf.Set("server.config.charset", "UTF-8")
	a.conf.Set("server.config.disableautofirestatuscode", false)
	a.conf.Set("server.config.disablebodyconsumptiononunmarshal", false)
	a.conf.Set("server.config.disableinterrupthandler", true)
	a.conf.Set("server.config.disablepathcorrection", false)
	a.conf.Set("server.config.disablestartuplog", false)
	a.conf.Set("server.config.disableversionchecker", false)
	a.conf.Set("server.config.enableoptimizations", true)
	a.conf.Set("server.config.enablepathescape", true)
	a.conf.Set("server.config.firemethodnotallowed", false)
	a.conf.Set("server.config.timeformat", "Mon, 02 Jan 2006 15:04:05 GMT")
}

func (a Application) setServerConfig() {
	a.Server.Use(recover.New())
	a.Server.Use(logger.New())
	a.Server.Favicon("./static/favicons/favicon.ico")
	a.Server.RegisterView(iris.HTML("./templates", ".html").Reload(a.conf.GetBool("application.debug")))
	m := prometheusMiddleware.New("serviceName", 300, 1200, 5000)
	a.Server.Use(m.ServeHTTP)
	iris.RegisterOnInterrupt(func() {
		timeout := 5 * time.Second
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
		defer cancel()
		// close all hosts
		fmt.Println("Shutting down server....")
		a.Log.Info("Wait 10 seconds and check your terminal again")
		a.Server.Shutdown(ctx)
	})
	yaag.Init(&yaag.Config{ // <- IMPORTANT, init the middleware.
		On:       true,
		DocTitle: a.conf.GetString("application.name"),
		DocPath:  a.conf.GetString("application.name") + "-apidoc.html",
		BaseUrls: map[string]string{"Production": "", "Staging": ""},
	})

	if a.conf.GetBool("application.debug") {
		a.Server.Use(irisyaag.New()) // <- IMPORTANT, register the middleware.
	}
}

// GET
func (a Application) GetName() string    { return a.conf.GetString("application.name") }
func (a Application) GetVersion() string { return a.conf.GetString("application.version") }
func (a Application) GetServerPort() int { return a.conf.GetInt("server.port") }

//SET
func (a Application) SetName(data string)    { a.conf.Set("application.name", data) }
func (a Application) SetVersion(data string) { a.conf.Set("application.version", data) }
func (a Application) SetServerPort(data int) { a.conf.Set("server.port", data) }

// Core
func New() *Application {
	_init_ctx.Do(func() { _instance = new(Application) })
	return _instance
}
func (a Application) LoadConfig(data string) {
	if data != "" {
		ok, err := path.IsExist(data)
		if ok {
			a.conf.ReadFiles(data)
		} else {
			panic(err)
		}
	}
	if a.conf.GetBool("application.debug") {
		a.Log.Level = logrus.DebugLevel
		a.Log.Debug("Debug Logging has been initialised...")
	} else {
		a.Log.Level = logrus.InfoLevel
		a.Log.Info("Info Logging has been initialised...")
	}
}
func (a Application) Run() {
	//app.Run(iris.Addr(":9090"), iris.WithConfiguration(serverConfig), iris.WithoutServerError(iris.ErrServerClosed))
}
