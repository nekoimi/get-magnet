package drission_rod

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nekoimi/scrapio/internal/bean"
	"github.com/nekoimi/scrapio/internal/config"
	pb "github.com/nekoimi/scrapio/internal/drission_rod/grpc"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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
		return fmt.Errorf("DrissionRod gRPC 地址未配置")
	}
	d.grpcHost = cfg.Crawler.DrissionRodGrpcIp
	d.grpcPort = cfg.Crawler.DrissionRodGrpcPort
	log.Infof("DrissionRodGrpc服务：%s:%d", d.grpcHost, d.grpcPort)
	d.ctx = ctx

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, err := grpc.NewClient(
				fmt.Sprintf("%s:%d", d.grpcHost, d.grpcPort),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				log.Errorf("连接DrissionRodGrpc服务异常：%s", err.Error())
				continue
			}

			client := pb.NewPageFetchServiceClient(conn)
			probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			_, probeErr := client.Health(probeCtx, &pb.BrowserHealthRequest{ProtocolVersion: BrowserProtocolVersion})
			if status.Code(probeErr) == codes.Unimplemented {
				_, probeErr = client.Fetch(probeCtx, &pb.FetchRequest{Url: "https://www.baidu.com", Timeout: 30})
			}
			cancel()
			if probeErr != nil {
				_ = conn.Close()
				log.Errorf("测试连接DrissionRodGrpc服务异常：%s", probeErr.Error())
				continue
			}
			d.mux.Lock()
			d.client = client
			d.conn = conn
			d.mux.Unlock()
			return nil
		}
	}
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
