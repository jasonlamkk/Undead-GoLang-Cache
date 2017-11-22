package main

// -- start setup
import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"

	"jason/server/controller"
	"jason/server/model"
)

var (
	flagConfigFile = flag.String("c", "./configs/dev.toml", "TOML config file")
)

func main() {
	flag.Parse()

	fmt.Println("start server with config: ", *flagConfigFile)
	app := iris.New()
	app.Use(recover.New())
	app.Use(logger.New())

	conf := iris.TOML(*flagConfigFile) //panic if not exists

	var addrStr string
	if tmp, ok := conf.Other["MyAddress"]; ok {
		addrStr, ok = tmp.(string)
		if !ok {
			fmt.Println("[ConfigError]", *flagConfigFile, "Other.MyAddress must be string")
			return
		}
	} else {
		fmt.Println("[ConfigError]", *flagConfigFile, "Other.MyAddress not found")
		return
	}
	// -- end setup

	//routes
	addAllRoute(app)

	model.StartModelCluster(getPureIP(addrStr))

	//graceful stop
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch,
			// kill -SIGINT XXXX or Ctrl+c
			os.Interrupt,
			syscall.SIGINT, // register that too, it should be ok
			// os.Kill  is equivalent with the syscall.Kill
			os.Kill,
			syscall.SIGKILL, // register that too, it should be ok
			// kill -SIGTERM XXXX
			syscall.SIGTERM,
		)
		select {
		case <-ch:

			println("leave cluster...")
			model.StopModelCluster()

			println("shutdown...")
			time.Sleep(time.Second)
			timeout := 5 * time.Second
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			app.Shutdown(ctx)
		}
	}()

	// var config = iris.WithConfiguration(conf)
	// config.DisableInterruptHandler = true
	// start server
	app.Run(iris.Addr(addrStr), iris.WithoutInterruptHandler, iris.WithConfiguration(conf))

}

//addAllRoute is the entry point for all controller routes, which shall be package private
func addAllRoute(app *iris.Application) {

	app.Post("/route", controller.HttpRoutePost)

	app.Get("/route", controller.HttpRoutePost)

	// app.Get("/route/{token:string regexp()}", httpRouteGet)
}

func getPureIP(addr string) string {
	end := strings.Index(addr, ":")
	if end < 0 {
		return addr
	}
	return addr[:end]
}
