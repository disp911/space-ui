// Package web provides the main web server implementation for the 3x-ui panel,
// including HTTP/HTTPS serving, routing, the frontend, and background job scheduling.
package web

import (
	"context"
	"crypto/tls"
	"embed"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/disp911/spacex-ui/v2/config"
	"github.com/disp911/spacex-ui/v2/logger"
	"github.com/disp911/spacex-ui/v2/util/common"
	"github.com/disp911/spacex-ui/v2/web/controller"
	"github.com/disp911/spacex-ui/v2/web/job"
	"github.com/disp911/spacex-ui/v2/web/locale"
	"github.com/disp911/spacex-ui/v2/web/middleware"
	"github.com/disp911/spacex-ui/v2/web/service"
	"github.com/disp911/spacex-ui/v2/web/session"
	"github.com/disp911/spacex-ui/v2/web/site"
	"github.com/disp911/spacex-ui/v2/web/spa"
	"github.com/disp911/spacex-ui/v2/web/websocket"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

//go:embed translation/*
var i18nFS embed.FS

// appFS holds the v3 frontend that Vite builds from frontend/ into web/app.
// The directory keeps a .gitkeep, so the panel compiles before the first build.
//
//go:embed all:app
var appFS embed.FS

// AppFS returns the v3 frontend that Vite builds: web/app on disk in debug
// mode, the embedded copy otherwise. The subscription server serves its page
// from it too.
func AppFS() fs.FS {
	if config.IsDebug() {
		return os.DirFS("web/app")
	}
	app, _ := fs.Sub(appFS, "app")
	return app
}

// Server represents the main web server for the 3x-ui panel with controllers, services, and scheduled jobs.
type Server struct {
	httpServer *http.Server
	listener   net.Listener
	// decoyServer is the loopback site telemt shows WEB proxy requests that
	// carry no valid credentials.
	decoyServer *http.Server

	// The website behind REALITY and port 80 beside it, see startReality.
	siteServer     *http.Server
	redirectServer *http.Server
	siteTLS        *tls.Config
	// What answers on the panel's listeners, and the same without the
	// WEB proxies in front, see startReality and startOwnPort.
	siteHandler http.Handler
	siteFront   http.Handler
	// The certificate the panel obtains itself, and the hourly look at
	// the site from outside: under siteName, at siteAddr.
	siteCerts          *site.Certs
	siteCheck          *site.Checker
	siteName, siteAddr string

	index *controller.IndexController
	panel *controller.XUIController
	api   *controller.APIController
	ws    *controller.WebSocketController

	xrayService      service.XrayService
	settingService   service.SettingService
	telemtService    service.TelemtService
	customGeoService *service.CustomGeoService

	wsHub *websocket.Hub

	cron *cron.Cron

	ctx    context.Context
	cancel context.CancelFunc
}

// NewServer creates a new web server instance with a cancellable context.
func NewServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		ctx:    ctx,
		cancel: cancel,
	}
}

// initRouter initializes Gin, registers middleware, controllers and the
// frontend and returns the configured engine.
func (s *Server) initRouter() (*gin.Engine, error) {
	if config.IsDebug() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.DefaultWriter = io.Discard
		gin.DefaultErrorWriter = io.Discard
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.Default()

	webDomain, err := s.settingService.GetWebDomain()
	if err != nil {
		return nil, err
	}

	if webDomain != "" {
		engine.Use(middleware.DomainValidatorMiddleware(webDomain))
	}

	secret, err := s.settingService.GetSecret()
	if err != nil {
		return nil, err
	}

	basePath, err := s.settingService.GetBasePath()
	if err != nil {
		return nil, err
	}
	engine.Use(gzip.Gzip(gzip.DefaultCompression))

	store := cookie.NewStore(secret)
	// Configure default session cookie options, including expiration (MaxAge)
	sessionOptions := sessions.Options{
		Path:     basePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if sessionMaxAge, err := s.settingService.GetSessionMaxAge(); err == nil && sessionMaxAge > 0 {
		sessionOptions.MaxAge = sessionMaxAge * 60 // minutes -> seconds
	}
	store.Options(sessionOptions)
	engine.Use(sessions.Sessions("3x-ui", store))
	engine.Use(func(c *gin.Context) {
		c.Set("base_path", basePath)
	})

	// init i18n
	err = locale.InitLocalizer(i18nFS)
	if err != nil {
		return nil, err
	}

	// Server messages in the request's language, see controller.I18nWeb.
	engine.Use(locale.LocalizerMiddleware())

	// Apply the redirect middleware (`/xui` to `/panel`)
	engine.Use(middleware.RedirectMiddleware(basePath))

	g := engine.Group(basePath)

	s.index = controller.NewIndexController(g)
	s.panel = controller.NewXUIController(g)
	s.api = controller.NewAPIController(g, s.customGeoService)

	// The panel's frontend, served from the base path: its files, and for any
	// other page index.html, so the app's own routes survive a reload.
	app := spa.Handler(AppFS(), func(c *gin.Context) spa.Config {
		return spa.Config{Base: basePath, Mount: basePath, LoggedIn: session.IsLogin(c), Version: config.GetVersion()}
	})

	// Initialize WebSocket hub
	s.wsHub = websocket.NewHub()
	go s.wsHub.Run()

	// Initialize WebSocket controller
	s.ws = controller.NewWebSocketController(s.wsHub)
	// Register WebSocket route with basePath (g already has basePath prefix)
	g.GET("/ws", s.ws.HandleWebSocket)

	// Chrome DevTools endpoint for debugging web apps
	engine.GET("/.well-known/appspecific/com.chrome.devtools.json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	// Under the base path a request no route took is the frontend's: a page
	// of the app or its files; the server's own endpoints under panel/ stay
	// theirs. Paths outside it are the website's (siteFront) and do not
	// reach the panel.
	engine.NoRoute(frontendOrDecoy(basePath, app, func(c *gin.Context) {
		c.AbortWithStatus(http.StatusNotFound)
	}))

	return engine, nil
}

// frontendOrDecoy answers a GET the routes did not take: under the base path
// with the frontend (app), except for the server's endpoints under panel/,
// and anywhere else with the decoy page. Other methods get 404.
func frontendOrDecoy(basePath string, app, decoy gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		if rest, ok := strings.CutPrefix(c.Request.URL.Path, basePath); ok && !strings.HasPrefix(rest, "panel/") {
			c.Params = append(c.Params, gin.Param{Key: "path", Value: "/" + rest})
			app(c)
			return
		}
		decoy(c)
	}
}

// startWebDecoy opens the loopback decoy site of the WEB proxies and tells
// the telemt service whether the panel can front them. Without the decoy
// site the WEB proxies stay off; the rest of the panel is unaffected.
func (s *Server) startWebDecoy(panel http.Handler, httpsOn443 bool) {
	domain, err := s.settingService.GetWebDomain()
	if err != nil {
		logger.Warning("WEB proxy decoy:", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		logger.Warning("WEB proxy decoy:", err)
		service.SetTelemtWebFront(0, false)
		return
	}
	s.decoyServer = &http.Server{
		Handler:           webDecoyHandler{panel: panel, domain: domain, route: service.TelemtWebRoute},
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		_ = s.decoyServer.Serve(listener)
	}()
	service.SetTelemtWebFront(listener.Addr().(*net.TCPAddr).Port, httpsOn443)
}

// startTask schedules background jobs (Xray checks, traffic jobs, cron
// jobs) which the panel relies on for periodic maintenance and monitoring.
func (s *Server) startTask() {
	s.customGeoService.EnsureOnStartup()
	err := s.xrayService.RestartXray(true)
	if err != nil {
		logger.Warning("start xray failed:", err)
	}
	// Check whether xray is running every second
	s.cron.AddJob("@every 1s", job.NewCheckXrayRunningJob())

	// Run the telemt proxies behind mtproto inbounds, keep them in sync with
	// the database and record their traffic
	s.telemtService.Sync()
	s.cron.AddJob("@every 10s", job.NewTelemtJob())

	// Check if xray needs to be restarted every 30 seconds
	s.cron.AddFunc("@every 30s", func() {
		if s.xrayService.IsNeedRestartAndSetFalse() {
			err := s.xrayService.RestartXray(false)
			if err != nil {
				logger.Error("restart xray failed:", err)
			}
		}
	})

	go func() {
		time.Sleep(time.Second * 5)
		// Statistics every 10 seconds, start the delay for 5 seconds for the first time, and staggered with the time to restart xray
		s.cron.AddJob("@every 10s", job.NewXrayTrafficJob())
	}()

	// Look at the website from outside every hour, the first time once a
	// certificate has had a moment to come
	if s.siteCheck != nil {
		time.AfterFunc(2*time.Minute, s.checkSite)
		s.cron.AddFunc("@hourly", s.checkSite)
	}

	// check client ips from log file every 10 sec
	s.cron.AddJob("@every 10s", job.NewCheckClientIpJob())

	// check client ips from log file every day
	s.cron.AddJob("@daily", job.NewClearLogsJob())

	// ingest Xray access log lines into the log database every 10 sec
	s.cron.AddJob("@every 10s", job.NewXrayLogIngestJob())

	// Inbound traffic reset jobs
	// Run every hour
	s.cron.AddJob("@hourly", job.NewPeriodicTrafficResetJob("hourly"))
	// Run once a day, midnight
	s.cron.AddJob("@daily", job.NewPeriodicTrafficResetJob("daily"))
	// Run once a week, midnight between Sat/Sun
	s.cron.AddJob("@weekly", job.NewPeriodicTrafficResetJob("weekly"))
	// Run once a month, midnight, first of month
	s.cron.AddJob("@monthly", job.NewPeriodicTrafficResetJob("monthly"))

	// LDAP sync scheduling
	if ldapEnabled, _ := s.settingService.GetLdapEnable(); ldapEnabled {
		runtime, err := s.settingService.GetLdapSyncCron()
		if err != nil || runtime == "" {
			runtime = "@every 1m"
		}
		j := job.NewLdapSyncJob()
		// job has zero-value services with method receivers that read settings on demand
		s.cron.AddJob(runtime, j)
	}

}

// Start initializes and starts the web server with configured settings, routes, and background jobs.
func (s *Server) Start() (err error) {
	// This is an anonymous function, no function name
	defer func() {
		if err != nil {
			s.Stop()
		}
	}()

	loc, err := s.settingService.GetTimeLocation()
	if err != nil {
		return err
	}
	s.cron = cron.New(cron.WithLocation(loc), cron.WithSeconds())
	s.cron.Start()

	s.customGeoService = service.NewCustomGeoService()

	engine, err := s.initRouter()
	if err != nil {
		return err
	}

	// Behind REALITY the panel shares port 443 with its website and opens
	// no port of its own; should REALITY's side fail to start, the panel
	// opens on its own port instead.
	ownPort, httpsOn443 := true, false
	if mode, _ := s.settingService.GetSiteMode(); mode == service.SiteReality {
		if err := s.startReality(engine); err != nil {
			logger.Error("Website behind REALITY:", err, "- the panel opens on its own port")
		} else {
			ownPort, httpsOn443 = false, true
		}
	}
	if ownPort {
		listener, on443, err := s.startOwnPort(engine)
		httpsOn443 = on443
		if err != nil {
			return err
		}
		s.listener = listener
		s.httpServer = caddyServer(s.siteHandler)
		go func() {
			s.httpServer.Serve(listener)
		}()
	}

	s.startWebDecoy(s.siteFront, httpsOn443)
	s.startTask()

	return nil
}

// Stop gracefully shuts down the web server, stops Xray and cron jobs.
func (s *Server) Stop() error {
	s.cancel()
	s.xrayService.StopXray()
	if s.cron != nil {
		s.cron.Stop()
	}
	s.telemtService.StopAll()
	if s.decoyServer != nil {
		_ = s.decoyServer.Close()
	}
	s.stopSite()
	service.SetTelemtWebFront(0, false)
	// Gracefully stop WebSocket hub
	if s.wsHub != nil {
		s.wsHub.Stop()
	}
	var err1 error
	var err2 error
	if s.httpServer != nil {
		err1 = s.httpServer.Shutdown(s.ctx)
	}
	if s.listener != nil {
		err2 = s.listener.Close()
	}
	return common.Combine(err1, err2)
}

// GetCtx returns the server's context for cancellation and deadline management.
func (s *Server) GetCtx() context.Context {
	return s.ctx
}

// GetCron returns the server's cron scheduler instance.
func (s *Server) GetCron() *cron.Cron {
	return s.cron
}

// GetWSHub returns the WebSocket hub instance.
func (s *Server) GetWSHub() any {
	return s.wsHub
}

func (s *Server) RestartXray() error {
	return s.xrayService.RestartXray(true)
}
