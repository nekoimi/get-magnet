package main

import (
	"flag"
	"github.com/nekoimi/get-magnet/config"
	"github.com/nekoimi/get-magnet/server"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	signalChan = make(chan os.Signal, 1)
	cfg        = config.Default()
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	flag.IntVar(&cfg.Port, "port", 8080, "http服务端口")
	flag.IntVar(&cfg.Engine.WorkerNum, "worker", cfg.Engine.WorkerNum, "任务池worker数量")
	flag.StringVar(&cfg.DB.Dns, "dsn", "", "数据库连接参数")
	flag.StringVar(&cfg.Engine.Aria2.JsonRpc, "jsonrpc", "", "aria2服务jsonrpc连接地址")
	flag.StringVar(&cfg.Engine.Aria2.Secret, "secret", "", "aria2服务jsonrpc连接secret")

	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
}

func main() {
	flag.Parse()
	srv := server.New(cfg)

	go func() {
		for {
			select {
			case s := <-signalChan:
				switch s {
				case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
					srv.Stop()
					return
				default:
					log.Println("Ignore Signal: ", s)
				}
			}
		}
	}()

	srv.Run()
}
