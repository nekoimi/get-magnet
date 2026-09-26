package crawlpolicy

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Budget counts distinct durable tasks/pages, not retry attempts.
type Budget struct {
	MaxDiscoveredPerPage int      `json:"max_discovered_per_page"`
	MaxTasks             int      `json:"max_tasks"`
	MaxPages             int      `json:"max_pages"`
	MaxDepth             int      `json:"max_depth"`
	MaxDurationSeconds   int      `json:"max_duration_seconds"`
	MaxDomains           int      `json:"max_domains"`
	AllowedDomains       []string `json:"allowed_domains"`
}

func Defaults(entryURL string) Budget {
	u, _ := url.Parse(entryURL)
	domains := []string{}
	if u != nil && u.Hostname() != "" {
		domains = append(domains, strings.ToLower(u.Hostname()))
	}
	return Budget{100, 1000, 500, 100, 1800, 1, domains}
}

func (b Budget) Validate(entryURL string) error {
	for _, field := range []struct {
		name       string
		value, max int
	}{
		{"max_discovered_per_page", b.MaxDiscoveredPerPage, 1000}, {"max_tasks", b.MaxTasks, 10000},
		{"max_pages", b.MaxPages, 10000}, {"max_depth", b.MaxDepth, 1000},
		{"max_duration_seconds", b.MaxDurationSeconds, 86400}, {"max_domains", b.MaxDomains, 32},
	} {
		if field.value < 1 || field.value > field.max {
			return fmt.Errorf("budget.%s: require 1–%d", field.name, field.max)
		}
	}
	if len(b.AllowedDomains) == 0 || len(b.AllowedDomains) > 32 {
		return fmt.Errorf("budget.allowed_domains: require 1–32 exact hostnames")
	}
	seen := map[string]bool{}
	for _, host := range b.AllowedDomains {
		parseHost := host
		if net.ParseIP(host) != nil && strings.Contains(host, ":") {
			parseHost = "[" + host + "]"
		}
		u, err := url.Parse("https://" + parseHost)
		if err != nil || u.Hostname() != host || host != strings.ToLower(host) || u.Port() != "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || strings.ContainsAny(host, "/* \t\n") || seen[host] {
			return fmt.Errorf("budget.allowed_domains: use unique lowercase hostnames without ports or wildcards")
		}
		seen[host] = true
	}
	if !b.Allows(entryURL) {
		return fmt.Errorf("budget.allowed_domains: entry hostname must be allowed")
	}
	return nil
}

func (b Budget) Allows(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return false
	}
	for _, host := range b.AllowedDomains {
		if strings.EqualFold(u.Hostname(), host) {
			return true
		}
	}
	return false
}

func FromDefinition(raw string) (Budget, string, error) {
	var d struct {
		Trigger struct {
			URL string `json:"url"`
		} `json:"trigger"`
		Budget *Budget `json:"budget"`
	}
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return Budget{}, "", err
	}
	b := Defaults(d.Trigger.URL)
	if d.Budget != nil {
		b = *d.Budget
	}
	return b, d.Trigger.URL, b.Validate(d.Trigger.URL)
}
