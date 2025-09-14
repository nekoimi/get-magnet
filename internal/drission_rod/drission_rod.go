package drission_rod

import (
	"context"
	"fmt"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/config"
	pb "github.com/nekoimi/get-magnet/internal/drission_rod/grpc"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sync"
	"time"
)

type DrissionRod struct {
	ctx        context.Context
	mux        sync.Mutex
	grpcHost   string
	grpcPort   int
	client     pb.PageFetchServiceClient
	connTicker *time.Ticker
}

func NewDrissionRod() *DrissionRod {
	return &DrissionRod{
		mux:        sync.Mutex{},
		connTicker: time.NewTicker(5 * time.Second),
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
	d.mux.Lock()
	defer d.mux.Unlock()

	cfg := bean.PtrFromContext[config.Config](ctx)
	d.grpcHost = cfg.Crawler.DrissionRodGrpcIp
	d.grpcPort = cfg.Crawler.DrissionRodGrpcPort
	log.Infof("DrissionRodGrpc服务：%s:%d", d.grpcHost, d.grpcPort)
	d.ctx = ctx

	for {
		select {
		case <-d.ctx.Done():
			return nil
		case <-d.connTicker.C:
			// 连接服务器
			conn, err := grpc.NewClient(
				fmt.Sprintf("%s:%d", d.grpcHost, d.grpcPort),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				log.Errorf("连接DrissionRodGrpc服务异常：%s", err.Error())
				continue
			}

			d.client = pb.NewPageFetchServiceClient(conn)
			// test
			_, err = d.client.Fetch(d.ctx, &pb.FetchRequest{
				Url:     "https://www.baidu.com",
				Timeout: 0,
			})
			if err != nil {
				log.Errorf("测试连接DrissionRodGrpc服务异常：%s", err.Error())
				continue
			}

			return nil
		}
	}
}

func (d *DrissionRod) Stop(ctx context.Context) error {
	d.connTicker.Stop()
	return nil
}
