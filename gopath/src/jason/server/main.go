package main

// -- start setup
import (
	"flag"
	"fmt"

	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
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

	// start server
	app.Run(iris.Addr(addrStr), iris.WithConfiguration(conf))
}
