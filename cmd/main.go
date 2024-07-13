package main

import (
	"flag"
	"github.com/nekoimi/get-magnet/cmd/server"
	"github.com/nekoimi/get-magnet/config"
	"log"
)

var cfg = config.Default()

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	flag.IntVar(&cfg.Port, "port", 8080, "http服务端口")
	flag.StringVar(&cfg.JwtSecret, "jwtSecret", "get-magnet", "jwt secret")
	flag.StringVar(&cfg.DB.Dns, "dsn", "", "数据库连接参数")
}

func main() {
	flag.Parse()
	server.New(cfg).Run()
}
