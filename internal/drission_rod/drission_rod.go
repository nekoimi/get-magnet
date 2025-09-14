package drission_rod

import (
	"context"
	"fmt"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/config"
	pb "github.com/nekoimi/get-magnet/internal/drission_rod/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DrissionRod struct {
	client pb.PageFetchServiceClient
}

func NewDrissionRod() *DrissionRod {
	return &DrissionRod{}
}

func (d *DrissionRod) Client() pb.PageFetchServiceClient {
	return d.client
}

func (d *DrissionRod) Name() string {
	return "DrissionRodGrpc"
}

func (d *DrissionRod) Start(ctx context.Context) error {
	cfg := bean.PtrFromContext[config.Config](ctx)
	ip := cfg.Crawler.DrissionRodGrpcIp
	port := cfg.Crawler.DrissionRodGrpcPort
	// 连接服务器
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", ip, port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	d.client = pb.NewPageFetchServiceClient(conn)
	return nil
}

func (d *DrissionRod) Stop(ctx context.Context) error {
	return nil
}
