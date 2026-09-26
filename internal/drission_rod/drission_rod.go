package drission_rod

import (
	"context"
	"fmt"
	"sync"

	"github.com/nekoimi/scrapio/internal/bean"
	"github.com/nekoimi/scrapio/internal/config"
	pb "github.com/nekoimi/scrapio/internal/drission_rod/grpc"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DrissionRod struct {
	ctx      context.Context
	mux      sync.Mutex
	grpcHost string
	grpcPort int
	client   pb.PageFetchServiceClient
	conn     *grpc.ClientConn
}

func NewDrissionRod() *DrissionRod {
	return &DrissionRod{
		mux: sync.Mutex{},
	}
}

func (d *DrissionRod) Client() pb.PageFetchServiceClient {
	d.mux.Lock()
	defer d.mux.Unlock()

	return d.client
}

func (d *DrissionRod) Name() string {
	return "DrissionRodGrpc"
}

func (d *DrissionRod) Start(ctx context.Context) error {
	cfg := bean.PtrFromContext[config.Config](ctx)
	if cfg == nil || cfg.Crawler == nil || cfg.Crawler.DrissionRodGrpcIp == "" || cfg.Crawler.DrissionRodGrpcPort <= 0 {
		log.Warn("scrapio-browser 未配置；浏览器任务需要连接后才能执行")
		return nil
	}
	d.grpcHost = cfg.Crawler.DrissionRodGrpcIp
	d.grpcPort = cfg.Crawler.DrissionRodGrpcPort
	log.Infof("DrissionRodGrpc服务：%s:%d", d.grpcHost, d.grpcPort)
	d.ctx = ctx

	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", d.grpcHost, d.grpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	d.mux.Lock()
	d.client = pb.NewPageFetchServiceClient(conn)
	d.conn = conn
	d.mux.Unlock()
	return nil
}

func (d *DrissionRod) Stop(ctx context.Context) error {
	d.mux.Lock()
	conn := d.conn
	d.conn = nil
	d.client = nil
	d.mux.Unlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}
