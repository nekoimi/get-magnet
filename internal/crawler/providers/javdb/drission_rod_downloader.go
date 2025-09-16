package javdb

import (
	"bytes"
	"context"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"github.com/nekoimi/get-magnet/internal/drission_rod"
	pb "github.com/nekoimi/get-magnet/internal/drission_rod/grpc"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
)

type drissionRodDownloader struct {
	ctx         context.Context
	drissionRod *drission_rod.DrissionRod
}

func newDrissionRodDownloader(ctx context.Context, drissionRod *drission_rod.DrissionRod) download.Downloader {
	return &drissionRodDownloader{
		ctx:         ctx,
		drissionRod: drissionRod,
	}
}

func (s *drissionRodDownloader) SetCookies(u *url.URL, cookies []*http.Cookie) {
}

func (s *drissionRodDownloader) Download(rawUrl string) (selection *goquery.Selection, err error) {
	// 等待页面加载
	log.Debugf("等待页面 %s 加载...", rawUrl)

	resp, err := s.drissionRod.Client().FetchJavDB(s.ctx, &pb.FetchRequest{
		Url:     rawUrl,
		Timeout: 300,
	})
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, errors.New(resp.Error)
	}

	html := resp.GetHtml()

	log.Debugf("drissionRod页面 %s 加载: %s", rawUrl, html)

	doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(html))
	if err != nil {
		return nil, err
	}

	return doc.Selection, nil
}
