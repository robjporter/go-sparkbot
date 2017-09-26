package app

import (
	"runtime"
	"sync"
	"time"

	stdContext "context"

	"../localtunnelme"

	"github.com/Sirupsen/logrus"
	"github.com/betacraft/yaag/irisyaag"
	"github.com/betacraft/yaag/yaag"
	prometheusMiddleware "github.com/iris-contrib/middleware/prometheus"
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
	"github.com/robjporter/go-utils/filesystem/config"
	"github.com/robjporter/go-utils/filesystem/path"
	"github.com/robjporter/go-utils/go/as"
)

var _init_ctx sync.Once
var _instance *Application

func init() {
	a := New()
	a.conf = config.New()
	a.Log = logrus.New()
	a.Server = iris.New()
	a.Tunnel = localtunnelme.NewTunnel()
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
	appName := "Default"
	a.conf.Set("application.name", appName)
	a.conf.Set("application.version", "1")
	a.conf.Set("application.debug", false)
	a.conf.Set("server.port", 9090)
	a.conf.Set("server.timeout", 5)
	a.conf.Set("server.localtunnel.name", appName)
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
		timeout := time.Duration(a.conf.GetInt("server.timeout")) * time.Second
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
		defer cancel()
		// close all hosts
		a.Log.Info("Shutting down server....")
		a.Stop()
		a.Log.Info("Wait ", a.conf.GetInt("server.timeout"), " seconds and check your terminal again")
		time.Sleep(time.Duration(a.conf.GetInt("server.timeout")) * time.Second)
		a.Server.Shutdown(ctx)
		a.Log.Info("Applciation has been shutdown successfully.")
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
	a.addRoutes()
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
func (a Application) createLocalTunnelMe() bool {
	a.Log.Info("Initialising LocalTunnel.Me config....")
	url, err := a.Tunnel.GetUrl(a.conf.GetString("server.localtunnel.name"))
	if err != nil {
		a.Log.Error(err)
		return false
	}
	a.Log.Info("LOCAL TUNNEL URL:", url)
	a.conf.Set("server.localtunnel.url", url)
	go func() {
		err := a.Tunnel.CreateTunnel(a.conf.GetInt("server.port"))
		if err != nil {
			a.Log.Error(err)
		}
	}()
	return true
}

func (a Application) Stop() {
	a.Tunnel.StopTunnel()
	a.Log.Info("LocalTunnelMe service has been stopped successfully....")
}

func (a Application) Run() {
	var serverConfig iris.Configuration
	a.createLocalTunnelMe()
	//deleteWebHooks()
	//registerWebHook()
	serverConfig.Charset = a.conf.GetString("server.config.charset")
	serverConfig.DisableAutoFireStatusCode = a.conf.GetBool("server.config.disableautofirestatuscode")
	serverConfig.DisableBodyConsumptionOnUnmarshal = a.conf.GetBool("server.config.disablebodyconsumptiononunmarshal")
	serverConfig.DisableInterruptHandler = a.conf.GetBool("server.config.disableinterrupthandler")
	serverConfig.DisablePathCorrection = a.conf.GetBool("server.config.disablepathcorrection")
	serverConfig.DisableStartupLog = a.conf.GetBool("server.config.disablestartuplog")
	serverConfig.DisableVersionChecker = a.conf.GetBool("server.config.disableversionchecker")
	serverConfig.EnableOptimizations = a.conf.GetBool("server.config.enableoptimizations")
	serverConfig.EnablePathEscape = a.conf.GetBool("server.config.enablepathescape")
	serverConfig.FireMethodNotAllowed = a.conf.GetBool("server.config.firemethodnotallowed")
	serverConfig.TimeFormat = a.conf.GetString("server.config.timeformat")

	a.Server.Run(iris.Addr(":"+as.ToString(a.conf.GetInt("server.port"))),
		iris.WithConfiguration(serverConfig),
		iris.WithoutServerError(iris.ErrServerClosed),
	)
}
