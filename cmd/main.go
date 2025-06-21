package main

import (
	"flag"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/javdb"
	"github.com/nekoimi/get-magnet/internal/server"
	"log"
	"os"
)

var cfg = config.Default()

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	flag.IntVar(&cfg.Port, "port", 8080, "http服务端口")
	flag.StringVar(&cfg.JwtSecret, "jwtSecret", "get-magnet", "jwt secret")
	flag.StringVar(&cfg.DB.Dns, "dsn", os.Getenv("DB_DSN"), "数据库连接参数")
	flag.StringVar(&cfg.Aria2Ops.JsonRpc, "jsonrpc", os.Getenv("ARIA2_JSONRPC"), "数据库连接参数")
	flag.StringVar(&cfg.Aria2Ops.Secret, "secret", os.Getenv("ARIA2_SECRET"), "数据库连接参数")

	flag.Parse()
}

func main() {
	s := server.Default(cfg)

	bus.Event().Publish(bus.SubmitTask.String(), crawler.NewStaticWorkerTask("https://javdb.com/censored?vft=2&vst=1", javdb.Handler()))

	s.Run()
}
