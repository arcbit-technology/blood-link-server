package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	config "github.com/mdhasib01/go-rest-starter/config"
	"github.com/mdhasib01/go-rest-starter/controller"
	dao "github.com/mdhasib01/go-rest-starter/dao"
	"github.com/mdhasib01/go-rest-starter/pkg/geoip"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
	notifier "github.com/mdhasib01/go-rest-starter/pkg/notifications"
	"github.com/mdhasib01/go-rest-starter/rest"

	_ "github.com/mdhasib01/go-rest-starter/docs"

	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	initserver()
}

func init() {
	binaryPath, err := os.Executable()

	if err != nil {
		log.Fatal(err.Error())
	}

	path := filepath.Dir(binaryPath)

	system := runtime.GOOS

	currentDir, _ := os.Getwd()

	if system == "windows" {
		path = currentDir
	}

	err = config.InitConfig(path)
	if err != nil {
		log.Fatal(err.Error() + "failed to init config")
	}

	// initializing logger
	logger.GetLogger()

	err = dao.InitDatabase(config.Param.ConnString)
	if err != nil {

		log.Println(err.Error() + " failed to init database")
	}

	err = config.InitAuthProviders()
	if err != nil {

		log.Println(err.Error() + " failed to init auth providers")
	}

	err = config.PingServices()
	if err != nil {

		log.Println(err.Error() + " failed to ping services")
	}
	// set email service
	notifier.ES = notifier.NewEmailService()

	geoip.InitGeoIP()

	// allocate the websocket manager
	controller.M = controller.NewManager()

	// start cron job
	err = controller.StartCronJob()
	if err != nil {
		log.Println(err.Error() + " failed to start cron job")
	}

	// err = controller.InitSettings()
	// if err != nil {
	// 	log.Println(err.Error() + " failed to init settings")
	// }
}

func initserver() {

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"POST", "PUT", "GET", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	logger.GetLogger().LogInfo(fmt.Sprintf("Go Server is listening on port: %s", config.Param.ServerPort), nil)

	handler := rest.InitializeRouter()
	handler.Router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	server := &http.Server{
		Addr:    ":" + config.Param.ServerPort,
		Handler: c.Handler(handler),
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Println(err)
	}

}
