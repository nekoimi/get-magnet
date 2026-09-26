package workflow

type Template struct {
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	RecordType  string     `json:"record_type"`
	Definition  Definition `json:"definition"`
}

// Templates provide editable rules; URLs are examples and never fetched here.
func Templates() []Template {
	field := func(name, selector string) any {
		return map[string]any{"name": name, "selector": selector, "required": name == "url"}
	}
	makeDefinition := func(url, contentType string, fields []any) Definition {
		return Definition{Persistence: "records", Trigger: Trigger{Type: "manual", URL: url, Fetch: FetchOptions{Mode: "http"}}, Nodes: []Node{
			{Name: "extract", Type: "extract", Config: map[string]any{"content_type": contentType, "fields": fields}},
			{Name: "clean", Type: "transform", Config: map[string]any{"operations": []any{map[string]any{"op": "trim", "field": "title"}}}},
			{Name: "check", Type: "validate", Config: map[string]any{"fields": []any{map[string]any{"name": "url", "required": true, "type": "string"}}}},
		}}
	}
	page := makeDefinition("https://example.org/article", "html", []any{map[string]any{"name": "url", "selector": "link[rel=canonical]", "attribute": "href", "required": true}, field("title", "h1"), field("body", "article")})
	api := makeDefinition("https://example.org/api/articles", "json", []any{field("url", "$.url"), field("title", "$.title"), field("body", "$.body")})
	api.Nodes[0].Config["items_path"] = "$.items"
	list := makeDefinition("https://example.org/articles", "html", []any{map[string]any{"name": "url", "selector": "link[rel=canonical]", "attribute": "href", "required": true}, field("title", "h1"), field("body", "article")})
	list.Listing = &ListingOptions{DetailSelector: "a.article-link[href]", MaxPages: 1, MaxEmptyPages: 1}
	paginated := list
	paginated.Listing = &ListingOptions{DetailSelector: "a.article-link[href]", NextSelector: "a[rel=next][href]", MaxPages: 10, MaxEmptyPages: 2}
	return []Template{
		{Code: "single_page", Name: "单页文章", Description: "提取页面的绝对 canonical URL、标题与正文，写入文章数据集", RecordType: "article", Definition: page},
		{Code: "json_api", Name: "JSON 文章接口", Description: "从 $.items 数组逐对象提取字段，整批写入文章数据集", RecordType: "article", Definition: api},
		{Code: "list_detail", Name: "列表到详情", Description: "从单个列表页发现详情，详情提取后写入数据集", RecordType: "article", Definition: list},
		{Code: "paginated_list", Name: "分页列表", Description: "按下一页链接翻页，发现详情并写入数据集", RecordType: "article", Definition: paginated},
	}
}
