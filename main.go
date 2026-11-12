// Package main is the entry point for the Space UI web panel application.
// It initializes the database, web server, and handles command-line operations for managing the panel.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	_ "unsafe"

	"github.com/disp911/space-ui/v3/config"
	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/logger"
	"github.com/disp911/space-ui/v3/sub"
	"github.com/disp911/space-ui/v3/util/crypto"
	"github.com/disp911/space-ui/v3/util/sys"
	"github.com/disp911/space-ui/v3/web"
	"github.com/disp911/space-ui/v3/web/global"
	"github.com/disp911/space-ui/v3/web/service"

	"github.com/joho/godotenv"
	"github.com/op/go-logging"
)

// runWebServer initializes and starts the web server for the Space UI panel.
func runWebServer() {
	log.Printf("Starting %v %v", config.GetName(), config.GetVersion())

	switch config.GetLogLevel() {
	case config.Debug:
		logger.InitLogger(logging.DEBUG)
	case config.Info:
		logger.InitLogger(logging.INFO)
	case config.Notice:
		logger.InitLogger(logging.NOTICE)
	case config.Warning:
		logger.InitLogger(logging.WARNING)
	case config.Error:
		logger.InitLogger(logging.ERROR)
	default:
		log.Fatalf("Unknown log level: %v", config.GetLogLevel())
	}

	godotenv.Load()

	err := database.InitDB(config.GetDBPath())
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}

	// A failure here is not fatal: without the log database the panel runs
	// normally and only the Xray log viewer comes up empty.
	if err := database.InitLogDB(); err != nil {
		logger.Warning("Error initializing Xray log database:", err)
	}

	var server *web.Server
	server = web.NewServer()
	global.SetWebServer(server)
	err = server.Start()
	if err != nil {
		log.Fatalf("Error starting web server: %v", err)
		return
	}

	var subServer *sub.Server
	subServer = sub.NewServer()
	global.SetSubServer(subServer)
	err = subServer.Start()
	if err != nil {
		log.Fatalf("Error starting sub server: %v", err)
		return
	}

	sigCh := make(chan os.Signal, 1)
	// Trap shutdown signals
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM, sys.SIGUSR1)
	for {
		sig := <-sigCh

		switch sig {
		case syscall.SIGHUP:
			logger.Info("Received SIGHUP signal. Restarting servers...")

			err := server.Stop()
			if err != nil {
				logger.Debug("Error stopping web server:", err)
			}
			err = subServer.Stop()
			if err != nil {
				logger.Debug("Error stopping sub server:", err)
			}

			server = web.NewServer()
			global.SetWebServer(server)
			err = server.Start()
			if err != nil {
				log.Fatalf("Error restarting web server: %v", err)
				return
			}
			log.Println("Web server restarted successfully.")

			subServer = sub.NewServer()
			global.SetSubServer(subServer)
			err = subServer.Start()
			if err != nil {
				log.Fatalf("Error restarting sub server: %v", err)
				return
			}
			log.Println("Sub server restarted successfully.")
		case sys.SIGUSR1:
			logger.Info("Received USR1 signal, restarting xray-core...")
			err := server.RestartXray()
			if err != nil {
				logger.Error("Failed to restart xray-core:", err)
			}

		default:
			server.Stop()
			subServer.Stop()
			log.Println("Shutting down servers.")
			return
		}
	}
}

// resetSetting resets all panel settings to their default values.
func resetSetting() error {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println("Failed to initialize database:", err)
		return err
	}

	settingService := service.SettingService{}
	err = settingService.ResetSettings()
	if err != nil {
		fmt.Println("Failed to reset settings:", err)
		return err
	} else {
		fmt.Println("Settings successfully reset.")
	}
	return nil
}

// showSetting displays the current panel settings if show is true.
func showSetting(show bool) {
	if show {
		settingService := service.SettingService{}
		port, err := settingService.GetPort()
		if err != nil {
			fmt.Println("get current port failed, error info:", err)
		}

		webBasePath, err := settingService.GetBasePath()
		if err != nil {
			fmt.Println("get webBasePath failed, error info:", err)
		}

		certFile, err := settingService.GetCertFile()
		if err != nil {
			fmt.Println("get cert file failed, error info:", err)
		}
		keyFile, err := settingService.GetKeyFile()
		if err != nil {
			fmt.Println("get key file failed, error info:", err)
		}

		userService := service.UserService{}
		userModel, err := userService.GetFirstUser()
		if err != nil {
			fmt.Println("get current user info failed, error info:", err)
		}

		if userModel.Username == "" || userModel.Password == "" {
			fmt.Println("current username or password is empty")
		}

		fmt.Println("current panel settings as follows:")
		mode, _ := settingService.GetSiteMode()
		name, _ := settingService.GetSiteDomain()
		switch {
		case certFile != "" && keyFile != "":
			fmt.Println("Panel is secure with SSL: certificate files")
		case mode == service.SiteReality || name != "":
			fmt.Println("Panel is secure with SSL: a certificate it obtains for", name)
		default:
			fmt.Println("Warning: Panel is not secure with SSL")
		}

		hasDefaultCredential := func() bool {
			return userModel.Username == "admin" && crypto.CheckPasswordHash(userModel.Password, "admin")
		}()

		fmt.Println("hasDefaultCredential:", hasDefaultCredential)
		fmt.Println("port:", port)
		fmt.Println("webBasePath:", webBasePath)
	}
}

// updateSetting updates various panel settings including port, credentials, base path, listen IP, and two-factor authentication.
func updateSetting(port int, username string, password string, webBasePath string, listenIP string, resetTwoFactor bool) error {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println("Database initialization failed:", err)
		return err
	}

	settingService := service.SettingService{}
	userService := service.UserService{}

	if port > 0 {
		err := settingService.SetPort(port)
		if err != nil {
			fmt.Println("Failed to set port:", err)
		} else {
			fmt.Printf("Port set successfully: %v\n", port)
		}
	}

	if username != "" || password != "" {
		err := userService.UpdateFirstUser(username, password)
		if err != nil {
			fmt.Println("Failed to update username and password:", err)
		} else {
			fmt.Println("Username and password updated successfully")
		}
	}

	if webBasePath != "" {
		err := settingService.SetBasePath(webBasePath)
		if err != nil {
			fmt.Println("Failed to set base URI path:", err)
		} else {
			fmt.Println("Base URI path set successfully")
		}
	}

	if resetTwoFactor {
		err := settingService.SetTwoFactorEnable(false)

		if err != nil {
			fmt.Println("Failed to reset two-factor authentication:", err)
		} else {
			settingService.SetTwoFactorToken("")
			fmt.Println("Two-factor authentication reset successfully")
		}
	}

	if listenIP != "" {
		err := settingService.SetListen(listenIP)
		if err != nil {
			fmt.Println("Failed to set listen IP:", err)
		} else {
			fmt.Printf("listen %v set successfully", listenIP)
		}
	}

	return nil
}

// updateSite sets where the website on port 443 runs and for which domain,
// then prints what is set. Empty values are left as they are.
func updateSite(mode, domain string) error {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println("Database initialization failed:", err)
		return err
	}
	settingService := service.SettingService{}
	if domain != "" {
		if domain == "-" {
			domain = ""
		}
		if err := settingService.SetSiteDomain(domain); err != nil {
			fmt.Println("Failed to set the website domain:", err)
			return err
		}
	}
	if current, _ := settingService.GetSiteMode(); mode == "" && domain != "" && current == service.SiteReality {
		// The REALITY inbound takes the new domain as its server name.
		mode = current
	}
	if mode != "" {
		if mode == "port" {
			mode = service.SitePort
		}
		siteService := service.SiteService{}
		ib, err := siteService.SwitchMode(mode)
		if err != nil {
			fmt.Println("Failed to set the website mode:", err)
			return err
		}
		if ib != nil {
			printSiteInbound(&siteService, ib)
		}
		fmt.Println("Restart the panel to apply the mode: space-ui restart")
	}
	mode, _ = settingService.GetSiteMode()
	if mode == service.SitePort {
		mode = "port"
	}
	domain, _ = settingService.GetSiteDomain()
	fmt.Println("website mode:", mode)
	fmt.Println("website name:", domain)
	fmt.Println("website socket:", config.GetSiteSocket())
	return nil
}

// lookupSite prints the addresses a name resolves to, one a line, so the
// installer can tell whether a domain points at this server.
func lookupSite(name string) error {
	addrs, err := net.LookupIP(name)
	if err != nil {
		fmt.Println("Failed to look up", name+":", err)
		return err
	}
	for _, a := range addrs {
		fmt.Println(a)
	}
	return nil
}

// selfSteal puts the website behind a REALITY inbound of its own on port
// 443, which passes whoever is not a client on to the website, and prints
// how its first client connects.
func selfSteal(domain string) error {
	return updateSite(service.SiteReality, domain)
}

func printSiteInbound(siteService *service.SiteService, ib *model.Inbound) {
	fmt.Printf("REALITY inbound id %d on port %d, target %s\n", ib.Id, ib.Port, config.GetSiteSocket())
	link, sub, err := siteService.FirstClientLinks(ib)
	if err != nil {
		fmt.Println("Failed to read the inbound's first client:", err)
		return
	}
	if link != "" {
		fmt.Println("client link:", link)
	}
	if sub != "" {
		fmt.Println("subscription:", sub)
	}
}

// updateCert updates the SSL certificate files for the panel.
func updateCert(publicKey string, privateKey string) {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	if (privateKey != "" && publicKey != "") || (privateKey == "" && publicKey == "") {
		settingService := service.SettingService{}
		err = settingService.SetCertFile(publicKey)
		if err != nil {
			fmt.Println("set certificate public key failed:", err)
		} else {
			fmt.Println("set certificate public key success")
		}

		err = settingService.SetKeyFile(privateKey)
		if err != nil {
			fmt.Println("set certificate private key failed:", err)
		} else {
			fmt.Println("set certificate private key success")
		}

		err = settingService.SetSubCertFile(publicKey)
		if err != nil {
			fmt.Println("set certificate for subscription public key failed:", err)
		} else {
			fmt.Println("set certificate for subscription public key success")
		}

		err = settingService.SetSubKeyFile(privateKey)
		if err != nil {
			fmt.Println("set certificate for subscription private key failed:", err)
		} else {
			fmt.Println("set certificate for subscription private key success")
		}
	} else {
		fmt.Println("both public and private key should be entered.")
	}
}

// GetCertificate displays the current SSL certificate settings if getCert is true.
func GetCertificate(getCert bool) {
	if getCert {
		settingService := service.SettingService{}
		certFile, err := settingService.GetCertFile()
		if err != nil {
			fmt.Println("get cert file failed, error info:", err)
		}
		keyFile, err := settingService.GetKeyFile()
		if err != nil {
			fmt.Println("get key file failed, error info:", err)
		}

		fmt.Println("cert:", certFile)
		fmt.Println("key:", keyFile)
	}
}

// GetListenIP displays the current panel listen IP address if getListen is true.
func GetListenIP(getListen bool) {
	if getListen {

		settingService := service.SettingService{}
		ListenIP, err := settingService.GetListen()
		if err != nil {
			log.Printf("Failed to retrieve listen IP: %v", err)
			return
		}

		fmt.Println("listenIP:", ListenIP)
	}
}

// migrateDb performs database migration operations for the Space UI panel.
func migrateDb() {
	inboundService := service.InboundService{}

	err := database.InitDB(config.GetDBPath())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Start migrating database...")
	inboundService.MigrateDB()
	fmt.Println("Migration done!")
}

// main is the entry point of the Space UI application.
// It parses command-line arguments to run the web server, migrate database, or update settings.
func main() {
	if len(os.Args) < 2 {
		runWebServer()
		return
	}

	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "show version")

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)

	settingCmd := flag.NewFlagSet("setting", flag.ExitOnError)
	var port int
	var username string
	var password string
	var webBasePath string
	var listenIP string
	var getListen bool
	var webCertFile string
	var webKeyFile string
	var reset bool
	var show bool
	var getCert bool
	var resetTwoFactor bool
	settingCmd.BoolVar(&reset, "reset", false, "Reset all settings")
	settingCmd.BoolVar(&show, "show", false, "Display current settings")
	settingCmd.IntVar(&port, "port", 0, "Set panel port number")
	settingCmd.StringVar(&username, "username", "", "Set login username")
	settingCmd.StringVar(&password, "password", "", "Set login password")
	settingCmd.StringVar(&webBasePath, "webBasePath", "", "Set base path for Panel")
	settingCmd.StringVar(&listenIP, "listenIP", "", "set panel listenIP IP")
	settingCmd.BoolVar(&resetTwoFactor, "resetTwoFactor", false, "Reset two-factor authentication settings")
	settingCmd.BoolVar(&getListen, "getListen", false, "Display current panel listenIP IP")
	settingCmd.BoolVar(&getCert, "getCert", false, "Display current certificate settings")
	settingCmd.StringVar(&webCertFile, "webCert", "", "Set path to public key file for panel")
	settingCmd.StringVar(&webKeyFile, "webCertKey", "", "Set path to private key file for panel")

	siteCmd := flag.NewFlagSet("site", flag.ExitOnError)
	var siteMode, siteDomain, siteLookup string
	siteCmd.StringVar(&siteMode, "mode", "", "Where the panel and its website answer: reality (443 behind REALITY) or port (the panel's own port)")
	siteCmd.StringVar(&siteDomain, "domain", "", "Set the domain or IP address the panel gets its certificate for, - to clear it")
	siteCmd.StringVar(&siteLookup, "lookup", "", "Print the addresses a domain resolves to")

	selfStealCmd := flag.NewFlagSet("selfsteal", flag.ExitOnError)
	var stealDomain string
	selfStealCmd.StringVar(&stealDomain, "domain", "", "Set the website domain")

	oldUsage := flag.Usage
	flag.Usage = func() {
		oldUsage()
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("    run            run web panel")
		fmt.Println("    migrate        migrate the database")
		fmt.Println("    setting        set settings")
		fmt.Println("    site           set where the panel and its website answer")
		fmt.Println("    selfsteal      put the website behind a REALITY inbound on port 443")
	}

	flag.Parse()
	if showVersion {
		fmt.Println(config.GetVersion())
		return
	}

	switch os.Args[1] {
	case "run":
		err := runCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		runWebServer()
	case "migrate":
		migrateDb()
	case "setting":
		err := settingCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		if reset {
			if err = resetSetting(); err != nil {
				return
			}
		} else {
			if err = updateSetting(port, username, password, webBasePath, listenIP, resetTwoFactor); err != nil {
				return
			}
		}
		if show {
			showSetting(show)
		}
		if getListen {
			GetListenIP(getListen)
		}
		if getCert {
			GetCertificate(getCert)
		}
	case "site":
		if err := siteCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println(err)
			return
		}
		if siteLookup != "" {
			if lookupSite(siteLookup) != nil {
				os.Exit(1)
			}
			return
		}
		if updateSite(siteMode, siteDomain) != nil {
			os.Exit(1)
		}
	case "selfsteal":
		if err := selfStealCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println(err)
			return
		}
		if selfSteal(stealDomain) != nil {
			os.Exit(1)
		}
	case "cert":
		err := settingCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		if reset {
			updateCert("", "")
		} else {
			updateCert(webCertFile, webKeyFile)
		}
	default:
		fmt.Println("Invalid subcommands")
		fmt.Println()
		runCmd.Usage()
		fmt.Println()
		settingCmd.Usage()
	}
}
