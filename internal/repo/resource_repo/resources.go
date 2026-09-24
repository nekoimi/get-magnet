package resource_repo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	log "github.com/sirupsen/logrus"
	"xorm.io/xorm"
)

// PageFilter contains the v2 resource filters shared by API and background
// jobs. It deliberately has no delivery/download fields.
type PageFilter struct {
	PageNum      int
	PageSize     int
	Keyword      string
	ResourceType string
	SourceID     *int64
	Status       string
	CreatedStart string
	CreatedEnd   string
}

func Save(resource *table.Resource) error {
	if resource == nil {
		return errors.New("resource is nil")
	}
	if resource.ResourceType == "" {
		resource.ResourceType = "magnet"
	}
	if resource.Status == "" {
		resource.Status = string(table.ResourceStatusDiscovered)
	}
	if !table.IsValidResourceStatus(table.ResourceStatus(resource.Status)) {
		return errors.New("invalid resource status")
	}
	if strings.TrimSpace(resource.Attributes) == "" {
		resource.Attributes = "{}"
	}
	if _, err := db.Instance().InsertOne(resource); err != nil {
		log.Errorf("保存资源异常: %s", err.Error())
		return err
	}
	return nil
}

// SaveCollected persists a provider result in the v2 resource model. It is
// idempotent by source, resource type and canonical key and records a created
// or duplicate event without involving delivery/download services.
func SaveCollected(origin, title, number, actress, rawURLHost, rawURLPath string, links []string, optimalLink string) (*table.Resource, error) {
	code := normalizeSourceCode(origin)
	sourceID, err := ensureSource(code, origin)
	if err != nil {
		return nil, err
	}
	canonicalKey := normalizeCanonicalKey(number)
	sourceURL := buildSourceURL(rawURLHost, rawURLPath)
	if canonicalKey == "" {
		canonicalKey = sourceURL
	}
	if canonicalKey == "" {
		return nil, errors.New("resource canonical key is empty")
	}
	canonicalKey = fitCanonicalKey(canonicalKey)

	resource := &table.Resource{}
	has, err := db.Instance().Where("source_id = ? AND resource_type = ? AND canonical_key = ?", sourceID, "magnet", canonicalKey).Get(resource)
	if err != nil {
		return nil, err
	}
	if has {
		if err := appendLinks(resource.Id, links, optimalLink); err != nil {
			return resource, err
		}
		_ = RecordEvent(resource.Id, "duplicate", "采集结果已存在", fmt.Sprintf(`{"origin":%q}`, origin))
		return resource, nil
	}

	status := table.ResourceStatusCollected
	if strings.TrimSpace(title) == "" || countValidLinks(links, optimalLink) == 0 {
		status = table.ResourceStatusInvalid
	}
	resource = &table.Resource{
		ResourceType: "magnet",
		SourceId:     sourceID,
		SourceURL:    sourceURL,
		CanonicalKey: canonicalKey,
		Title:        strings.TrimSpace(title),
		Status:       string(status),
		Attributes:   attributesJSON(actress),
		FirstSeenAt:  time.Now(),
		LastSeenAt:   time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := Save(resource); err != nil {
		return nil, err
	}
	if err := appendLinks(resource.Id, links, optimalLink); err != nil {
		return resource, err
	}
	if err := RecordEvent(resource.Id, "created", "资源已采集", "{}"); err != nil {
		return resource, err
	}
	return resource, nil
}

func ExistsBySourceURL(origin, sourceURL string) (bool, error) {
	code := normalizeSourceCode(origin)
	sourceID, err := findSourceID(code)
	if err != nil || sourceID == 0 {
		return false, err
	}
	return db.Instance().Where("source_id = ? AND source_url = ?", sourceID, sourceURL).Exist(new(table.Resource))
}

func ExistsByCanonicalKey(origin, canonicalKey string) (bool, error) {
	sourceID, err := findSourceID(normalizeSourceCode(origin))
	if err != nil || sourceID == 0 {
		return false, err
	}
	return db.Instance().Where("source_id = ? AND resource_type = ? AND canonical_key = ?", sourceID, "magnet", normalizeCanonicalKey(canonicalKey)).Exist(new(table.Resource))
}

func ListSources() ([]table.Source, error) {
	var sources []table.Source
	err := db.Instance().Where("enabled = ?", true).OrderBy("name ASC").Find(&sources)
	return sources, err
}

func ensureSource(code, origin string) (int64, error) {
	name := strings.TrimSpace(origin)
	if name == "" {
		name = "Legacy"
	}
	if _, err := db.Instance().Exec("INSERT INTO sources (code, name) VALUES (?, ?) ON CONFLICT (code) DO UPDATE SET updated_at = NOW()", code, name); err != nil {
		return 0, err
	}
	return findSourceID(code)
}

func findSourceID(code string) (int64, error) {
	var source table.Source
	has, err := db.Instance().Where("code = ?", code).Get(&source)
	if err != nil {
		return 0, err
	}
	if !has {
		return 0, nil
	}
	return source.Id, nil
}

func GetByID(id int64) (*table.Resource, bool) {
	resource := new(table.Resource)
	has, err := db.Instance().ID(id).Get(resource)
	if err != nil {
		log.Errorf("查询资源异常: %d - %s", id, err.Error())
		return nil, false
	}
	return resource, has
}

func PageList(filter PageFilter) ([]table.Resource, int64, error) {
	if filter.PageNum <= 0 {
		filter.PageNum = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	count := db.Instance().NewSession()
	defer count.Close()
	applyFilter(count, filter)
	total, err := count.Count(new(table.Resource))
	if err != nil {
		return nil, 0, err
	}
	listSession := db.Instance().NewSession()
	defer listSession.Close()
	applyFilter(listSession, filter)
	var list []table.Resource
	if err := listSession.OrderBy("created_at DESC").Limit(filter.PageSize, (filter.PageNum-1)*filter.PageSize).Find(&list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func CountAll() (int64, error) {
	return db.Instance().Count(new(table.Resource))
}

func CountCreatedSince(since time.Time) (int64, error) {
	return db.Instance().Where("created_at >= ?", since).Count(new(table.Resource))
}

func CountByStatus() (map[string]int64, error) {
	type statusCount struct {
		Status string `xorm:"status"`
		Total  int64  `xorm:"total"`
	}
	var rows []statusCount
	if err := db.Instance().Table(new(table.Resource)).Select("status, COUNT(*) AS total").GroupBy("status").Find(&rows); err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[row.Status] = row.Total
	}
	return result, nil
}

func applyFilter(session *xorm.Session, filter PageFilter) {
	session.Where("1 = 1")
	if strings.TrimSpace(filter.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(filter.Keyword) + "%"
		session.And("(title ILIKE ? OR canonical_key ILIKE ?)", keyword, keyword)
	}
	if filter.ResourceType != "" {
		session.And("resource_type = ?", filter.ResourceType)
	}
	if filter.SourceID != nil {
		session.And("source_id = ?", *filter.SourceID)
	}
	if filter.Status != "" {
		session.And("status = ?", filter.Status)
	}
	if filter.CreatedStart != "" {
		session.And("created_at >= ?", filter.CreatedStart)
	}
	if filter.CreatedEnd != "" {
		session.And("created_at <= ?", filter.CreatedEnd)
	}
}

func ListLinks(resourceID int64) ([]table.ResourceLink, error) {
	var links []table.ResourceLink
	err := db.Instance().Where("resource_id = ?", resourceID).OrderBy("priority ASC, id ASC").Find(&links)
	return links, err
}

func ListEvents(resourceID int64, limit int) ([]table.ResourceEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	var events []table.ResourceEvent
	err := db.Instance().Where("resource_id = ?", resourceID).OrderBy("created_at DESC").Limit(limit).Find(&events)
	return events, err
}

func RecordEvent(resourceID int64, eventType, message string, data string) error {
	if resourceID <= 0 || strings.TrimSpace(eventType) == "" {
		return errors.New("resource id and event type are required")
	}
	if strings.TrimSpace(data) == "" {
		data = "{}"
	}
	_, err := db.Instance().InsertOne(&table.ResourceEvent{
		ResourceId: resourceID,
		EventType:  eventType,
		Message:    message,
		Data:       data,
		CreatedAt:  time.Now(),
	})
	return err
}

func appendLinks(resourceID int64, links []string, optimalLink string) error {
	seen := make(map[string]struct{}, len(links)+1)
	if strings.TrimSpace(optimalLink) != "" {
		links = append(links, optimalLink)
	}
	for priority, link := range links {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		if _, ok := seen[link]; ok {
			continue
		}
		seen[link] = struct{}{}
		_, err := db.Instance().Exec(`INSERT INTO resource_links
 (resource_id, link_type, link, priority, is_optimal, metadata)
 VALUES (?, ?, ?, ?, ?, '{}'::jsonb)
 ON CONFLICT (resource_id, link_type, link) DO UPDATE SET is_optimal = resource_links.is_optimal OR EXCLUDED.is_optimal`,
			resourceID, linkType(link), link, priority, link == strings.TrimSpace(optimalLink))
		if err != nil {
			return err
		}
	}
	return nil
}

func countValidLinks(links []string, optimal string) int {
	count := 0
	seen := map[string]struct{}{}
	for _, link := range append(append([]string{}, links...), optimal) {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		if _, ok := seen[link]; ok {
			continue
		}
		seen[link] = struct{}{}
		if linkType(link) == "magnet" && strings.Contains(strings.ToLower(link), "xt=") {
			count++
		} else if parsed, err := url.Parse(link); err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" {
			count++
		}
	}
	return count
}

func linkType(link string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(link)), "magnet:") {
		return "magnet"
	}
	return "url"
}

func attributesJSON(actress string) string {
	actress = strings.TrimSpace(actress)
	if actress == "" {
		return "{}"
	}
	return fmt.Sprintf(`{"actress":%q}`, actress)
}

func normalizeSourceCode(origin string) string {
	code := strings.ToLower(strings.TrimSpace(origin))
	if code == "" {
		return "legacy"
	}
	if len(code) > 64 {
		hash := sha256.Sum256([]byte(code))
		return code[:55] + "-" + hex.EncodeToString(hash[:])[:8]
	}
	return code
}

func normalizeCanonicalKey(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToUpper(r)
	}, strings.TrimSpace(value))
}

func fitCanonicalKey(value string) string {
	if len(value) <= 512 {
		return value
	}
	hash := sha256.Sum256([]byte(value))
	return value[:495] + "-" + hex.EncodeToString(hash[:])[:16]
}

func buildSourceURL(host, path string) string {
	host, path = strings.TrimSpace(host), strings.TrimSpace(path)
	if host == "" {
		return path
	}
	if parsed, err := url.Parse(host); err == nil && parsed.Scheme != "" {
		return strings.TrimRight(host, "/") + "/" + strings.TrimLeft(path, "/")
	}
	return "https://" + strings.TrimRight(host, "/") + "/" + strings.TrimLeft(path, "/")
}
