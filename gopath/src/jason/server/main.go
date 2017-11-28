package main

// -- start setup
import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"

	"jason/server/cluster"
	"jason/server/configstore"
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

	if tmp, ok := conf.Other["UseProcessors"]; ok {
		if numUseProcessors, ok := tmp.(int64); ok {
			fmt.Println("Limited number of processor to : ", numUseProcessors)
			runtime.GOMAXPROCS(int(numUseProcessors))
		}
	}

	var exIP, mapKey string
	var portShift int64

	if tmp, ok := conf.Other["GoogleMapApiKey"]; ok {
		mapKey, ok = tmp.(string)
		if !ok {
			panic("Other.GoogleMapApiKey is not a string")
		}
	} else {
		panic("Other.GoogleMapApiKey is required")
	}

	model.InitGoogleMapWithApiKey(mapKey)

	if tmp, ok := conf.Other["MyPortShift"]; ok {
		portShift, ok = tmp.(int64)
		if !ok {
			fmt.Println("[ConfigError]", *flagConfigFile, "Other.MyPortShift must be int")
		}
	} else {
		portShift = 0 //optional , default 0
	}

	if tmp, ok := conf.Other["MyAddress"]; ok {
		exIP, ok = tmp.(string)
		if !ok {
			fmt.Println("[ConfigError]", *flagConfigFile, "Other.MyAddress must be string")
			return
		}
	} else {
		fmt.Println("[ConfigError]", *flagConfigFile, "Other.MyAddress not found")
		return
	}

	configstore.GetAddressStore().SetAddress(exIP, int(portShift))

	if tmp, ok := conf.Other["ClusterAcceptPattern"]; ok {
		ptn, ok := tmp.(string)
		// fmt.Println("ClusterAcceptPattern", ptn)
		if ok {
			configstore.SetClusterAcceptPattern(ptn)
		}
	}
	// -- end setup

	//graceful stop logic
	clusterCtx, clusterStopper := context.WithCancel(context.Background()) //stop later

	serverCtx, serverStopper := context.WithCancel(clusterCtx) //stop earlier
	//start supporting tasks

	model.StartBgTask(serverCtx)
	defer model.StopBgTask()

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
			serverStopper()
			println("shutdown...")
			wait := 3 * time.Second                                           //wait 3 seconds for cluster
			stopCtx, cancelStopCtx := context.WithTimeout(clusterCtx, 2*wait) //wait max 6 seconds for cluster to rebalance
			go cluster.PublishRebalance(stopCtx)                              //fire rebalance
			time.Sleep(wait)                                                  //wait 3 seconds, as incoming websocket depend on web server, server shutdown will harm rebalance
			//leave cluster
			clusterStopper()
			defer cancelStopCtx()
			app.Shutdown(stopCtx)
		}
	}()

	//start controller routes
	app.Post("/route", controller.HTTPRoutePost)
	app.Get(controller.PathGetRoutePrefix+"{token:string regexp(^[a-f0-9]{8,8}-[a-f0-9]{4,4}-[a-f0-9]{4,4}-[a-f0-9]{4,4}-[a-f0-9]{12,12})}", controller.HTTPRouteGet)

	app.Get(cluster.RouteClusterSocket, cluster.GetWsHandler(clusterCtx))
	//end controller routes

	//invoke SetRebalanceInjector
	cluster.SetRebalanceInjector(model.GetRouteRequestStoreForInject())

	if tmp, ok := conf.Other["JoinCluster"]; ok {
		// fmt.Println("init peers:", tmp)
		if peers, ok := tmp.([]interface{}); ok {
			cluster.JoinCluster(clusterCtx, peers)
		}
	}

	// start server
	serverAddr := configstore.GetAddressStore().GetServerAddress()
	fmt.Println("------------------------------------------")
	fmt.Println("-Starting server on:", serverAddr, "-")
	fmt.Println("------------------------------------------")
	app.Run(iris.Addr(serverAddr), iris.WithoutInterruptHandler, iris.WithConfiguration(conf))
	clusterStopper() //ensure stop, if not called ; no drawback
}

// func getPureIP(addr string) string {
// 	end := strings.Index(addr, ":")
// 	if end < 0 {
// 		return addr
// 	}
// 	return addr[:end]
// }
