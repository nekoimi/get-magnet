package main

import (
	"flag"
	"github.com/nekoimi/get-magnet/config"
	"github.com/nekoimi/get-magnet/server"
	"log"
)

var cfg = config.Default()

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	flag.IntVar(&cfg.Port, "port", 8080, "http服务端口")
	flag.StringVar(&cfg.DB.Dns, "dsn", "", "数据库连接参数")
}

func main() {
	flag.Parse()

	server.New(cfg).Run()
}
