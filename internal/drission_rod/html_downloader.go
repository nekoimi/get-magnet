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
	"github.com/nekoimi/scrapio/internal/bean"
	"github.com/nekoimi/scrapio/internal/crawler/download"
	"github.com/nekoimi/scrapio/internal/db"
	"github.com/nekoimi/scrapio/internal/db/table"
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
	result, err := d.client.Execute(d.ctx, BrowserJob{URL: rawURL, Recipe: d.recipe, Timeout: 5 * time.Minute, Outputs: []string{"html", "screenshot"}, ClosePage: true})
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
			if len(result.Screenshot) > 0 && len(result.Screenshot) <= 10*1024*1024 {
				assetHash := sha256.Sum256(result.Screenshot)
				_, _ = db.Instance().Exec(`INSERT INTO document_assets (document_id, asset_type, content_type, content, content_hash, content_size, metadata) VALUES (?, 'screenshot', 'image/png', ?, ?, ?, '{}'::jsonb) ON CONFLICT (document_id, asset_type) DO UPDATE SET content = EXCLUDED.content, content_hash = EXCLUDED.content_hash, content_size = EXCLUDED.content_size`, document.Id, result.Screenshot, hex.EncodeToString(assetHash[:]), len(result.Screenshot))
			}
		}
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(result.HTML))
	if err != nil {
		return nil, err
	}
	return doc.Selection, nil
}
