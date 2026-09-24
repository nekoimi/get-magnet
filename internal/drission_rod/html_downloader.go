package drission_rod

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
)

type HTMLDownloader struct {
	ctx    context.Context
	client *DrissionRod
	recipe string
}

func NewHTMLDownloader(ctx context.Context, recipe string) download.Downloader {
	return &HTMLDownloader{ctx: ctx, client: bean.PtrFromContext[DrissionRod](ctx), recipe: recipe}
}

func (d *HTMLDownloader) SetCookies(_ *url.URL, _ []*http.Cookie) {}

func (d *HTMLDownloader) Download(rawURL string) (*goquery.Selection, error) {
	if d.client == nil {
		return nil, errors.New("drission-rod client is unavailable")
	}
	result, err := d.client.Execute(d.ctx, BrowserJob{URL: rawURL, Recipe: d.recipe, Timeout: 5 * time.Minute, Outputs: []string{"html"}, ClosePage: true})
	if err != nil {
		return nil, err
	}
	if result.HTML == "" {
		return nil, errors.New("browser returned empty html")
	}
	if db.Instance() != nil {
		hash := sha256.Sum256([]byte(result.HTML))
		document := &table.Document{DocumentType: "html", Content: result.HTML, ContentHash: hex.EncodeToString(hash[:]), ContentSize: int64(len(result.HTML)), Metadata: "{}", CreatedAt: time.Now()}
		if _, err := db.Instance().InsertOne(document); err == nil {
			result.DocumentID = document.Id
		}
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(result.HTML))
	if err != nil {
		return nil, err
	}
	return doc.Selection, nil
}
