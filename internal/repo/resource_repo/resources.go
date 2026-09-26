package resource_repo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/nekoimi/scrapio/internal/db"
	"github.com/nekoimi/scrapio/internal/db/table"
	"github.com/nekoimi/scrapio/internal/repo/record_repo"
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
		if _, err := record_repo.ImportLegacyResource(resource.Id); err != nil {
			return resource, fmt.Errorf("sync legacy resource %d: %w", resource.Id, err)
		}
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
	if _, err := record_repo.ImportLegacyResource(resource.Id); err != nil {
		return resource, fmt.Errorf("sync legacy resource %d: %w", resource.Id, err)
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

func ListAllSources() ([]table.Source, error) {
	var sources []table.Source
	err := db.Instance().OrderBy("name ASC").Find(&sources)
	return sources, err
}

func SaveSource(source *table.Source) error {
	if source == nil || strings.TrimSpace(source.Code) == "" || strings.TrimSpace(source.Name) == "" {
		return errors.New("source code and name are required")
	}
	source.Code = normalizeSourceCode(source.Code)
	if source.Config == "" {
		source.Config = "{}"
	}
	source.CreatedAt, source.UpdatedAt = time.Now(), time.Now()
	if _, err := db.Instance().InsertOne(source); err != nil {
		return err
	}
	return nil
}

func UpdateSource(source *table.Source) error {
	if source == nil || source.Id <= 0 {
		return errors.New("source id is required")
	}
	if strings.TrimSpace(source.Name) == "" {
		return errors.New("source name is required")
	}
	if source.Config == "" {
		source.Config = "{}"
	}
	source.UpdatedAt = time.Now()
	_, err := db.Instance().ID(source.Id).Cols("name", "config", "updated_at").Update(source)
	return err
}

func SetSourceEnabled(id int64, enabled bool) error {
	if id <= 0 {
		return errors.New("source id is required")
	}
	_, err := db.Instance().ID(id).Cols("enabled", "updated_at").Update(&table.Source{Enabled: enabled, UpdatedAt: time.Now()})
	return err
}

func GetSource(id int64) (*table.Source, bool) {
	source := new(table.Source)
	has, err := db.Instance().ID(id).Get(source)
	if err != nil || !has {
		return nil, false
	}
	return source, true
}

// ResolveSourceID returns an existing source id or creates a stable source
// entry for a user/provider supplied code.
func ResolveSourceID(sourceID *int64, sourceCode, sourceName string) (int64, error) {
	if sourceID != nil && *sourceID > 0 {
		var source table.Source
		has, err := db.Instance().ID(*sourceID).Get(&source)
		if err != nil {
			return 0, err
		}
		if !has {
			return 0, errors.New("source not found")
		}
		return *sourceID, nil
	}
	if strings.TrimSpace(sourceName) == "" {
		sourceName = sourceCode
	}
	return ensureSource(normalizeSourceCode(sourceCode), sourceName)
}

func Update(resource *table.Resource, links []LinkInput) error {
	if resource == nil || resource.Id <= 0 {
		return errors.New("resource id is required")
	}
	if !table.IsValidResourceStatus(table.ResourceStatus(resource.Status)) {
		return errors.New("invalid resource status")
	}
	current, exists := GetByID(resource.Id)
	if !exists {
		return errors.New("resource not found")
	}
	if !table.CanTransitionResourceStatus(table.ResourceStatus(current.Status), table.ResourceStatus(resource.Status)) {
		return fmt.Errorf("invalid resource status transition: %s -> %s", current.Status, resource.Status)
	}
	if strings.TrimSpace(resource.CanonicalKey) == "" {
		return errors.New("canonical key is required")
	}
	resource.CanonicalKey = fitCanonicalKey(normalizeCanonicalKey(resource.CanonicalKey))
	if strings.TrimSpace(resource.Attributes) == "" {
		resource.Attributes = "{}"
	}
	resource.UpdatedAt = time.Now()
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	if _, err := s.ID(resource.Id).Cols("resource_type", "source_id", "source_url", "canonical_key", "title", "status", "attributes", "updated_at").Update(resource); err != nil {
		_ = s.Rollback()
		return err
	}
	if links != nil {
		if _, err := s.Where("resource_id = ?", resource.Id).Delete(new(table.ResourceLink)); err != nil {
			_ = s.Rollback()
			return err
		}
		if err := insertLinks(s, resource.Id, links); err != nil {
			_ = s.Rollback()
			return err
		}
	}
	if err := s.Commit(); err != nil {
		return err
	}
	return RecordEvent(resource.Id, "updated", "资源已更新", "{}")
}

type LinkInput struct {
	Link      string `json:"link"`
	Name      string `json:"name,omitempty"`
	Priority  int    `json:"priority,omitempty"`
	IsOptimal bool   `json:"is_optimal,omitempty"`
}

func Delete(id int64) error {
	if id <= 0 {
		return errors.New("resource id is required")
	}
	_, err := db.Instance().ID(id).Delete(new(table.Resource))
	return err
}

func ReplaceLinks(resourceID int64, links []LinkInput) error {
	if resourceID <= 0 {
		return errors.New("resource id is required")
	}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	if _, err := s.Where("resource_id = ?", resourceID).Delete(new(table.ResourceLink)); err != nil {
		_ = s.Rollback()
		return err
	}
	if err := insertLinks(s, resourceID, links); err != nil {
		_ = s.Rollback()
		return err
	}
	if err := s.Commit(); err != nil {
		return err
	}
	return RecordEvent(resourceID, "links_updated", "资源链接已更新", "{}")
}

func BatchDelete(ids []int64) error {
	clean := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id > 0 {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				clean = append(clean, id)
			}
		}
	}
	if len(clean) == 0 {
		return errors.New("resource ids are required")
	}
	_, err := db.Instance().In("id", clean).Delete(new(table.Resource))
	return err
}

func MarkStatus(id int64, status, message string) error {
	resource, exists := GetByID(id)
	if !exists {
		return errors.New("resource not found")
	}
	to := table.ResourceStatus(status)
	if !table.CanTransitionResourceStatus(table.ResourceStatus(resource.Status), to) {
		return fmt.Errorf("invalid resource status transition: %s -> %s", resource.Status, status)
	}
	if _, err := db.Instance().ID(id).Cols("status", "updated_at").Update(&table.Resource{Status: status, UpdatedAt: time.Now()}); err != nil {
		return err
	}
	return RecordEvent(id, "status_changed", message, fmt.Sprintf(`{"from":%q,"to":%q}`, resource.Status, status))
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

// MergeAttributes applies a provider-owned patch to resource.attributes and
// records the corresponding event atomically. Delivery metadata is kept in
// attributes so it remains extensible without adding provider columns.
func MergeAttributes(resourceID int64, patch map[string]any, eventType, message string) error {
	if db.Instance() == nil {
		return errors.New("database is not initialized")
	}
	if resourceID <= 0 || len(patch) == 0 {
		return errors.New("resource id and attributes patch are required")
	}
	if strings.TrimSpace(eventType) == "" {
		return errors.New("event type is required")
	}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	resource := new(table.Resource)
	has, err := s.ID(resourceID).Get(resource)
	if err != nil {
		_ = s.Rollback()
		return err
	}
	if !has {
		_ = s.Rollback()
		return errors.New("resource not found")
	}
	attributes := map[string]any{}
	if strings.TrimSpace(resource.Attributes) != "" {
		if err := json.Unmarshal([]byte(resource.Attributes), &attributes); err != nil {
			_ = s.Rollback()
			return fmt.Errorf("resource attributes are invalid: %w", err)
		}
	}
	if attributes == nil {
		attributes = map[string]any{}
	}
	mergeAttributeMap(attributes, patch)
	encoded, err := json.Marshal(attributes)
	if err != nil {
		_ = s.Rollback()
		return err
	}
	eventData, err := json.Marshal(patch)
	if err != nil {
		_ = s.Rollback()
		return err
	}
	now := time.Now()
	if _, err := s.ID(resourceID).Cols("attributes", "updated_at", "last_seen_at").Update(&table.Resource{
		Attributes: string(encoded), UpdatedAt: now, LastSeenAt: now,
	}); err != nil {
		_ = s.Rollback()
		return err
	}
	if _, err := s.Insert(&table.ResourceEvent{
		ResourceId: resourceID, EventType: eventType, Message: message,
		Data: string(eventData), CreatedAt: now,
	}); err != nil {
		_ = s.Rollback()
		return err
	}
	return s.Commit()
}

func mergeAttributeMap(target, patch map[string]any) {
	for key, value := range patch {
		if nested, ok := value.(map[string]any); ok {
			if current, ok := target[key].(map[string]any); ok {
				mergeAttributeMap(current, nested)
				continue
			}
		}
		target[key] = value
	}
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

func insertLinks(s *xorm.Session, resourceID int64, links []LinkInput) error {
	seen := make(map[string]struct{}, len(links))
	for i, input := range links {
		link := strings.TrimSpace(input.Link)
		if link == "" {
			continue
		}
		if _, ok := seen[link]; ok {
			continue
		}
		seen[link] = struct{}{}
		priority := input.Priority
		if priority == 0 {
			priority = i
		}
		if _, err := s.Insert(&table.ResourceLink{ResourceId: resourceID, LinkType: linkType(link), Link: link, Name: input.Name, Priority: priority, IsOptimal: input.IsOptimal, Metadata: "{}", CreatedAt: time.Now()}); err != nil {
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
