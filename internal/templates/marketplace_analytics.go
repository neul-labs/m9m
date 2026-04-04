package templates

func NewMarketplaceAnalytics() *MarketplaceAnalytics {
	return &MarketplaceAnalytics{
		searchQueries: make(map[string]int),
		downloads:     make(map[string]int64),
		errors:        make(map[string]int),
		trends: &TrendAnalysis{
			CategoryTrends:   make(map[string]float64),
			SeasonalPatterns: make(map[string][]int),
		},
	}
}

func (analytics *MarketplaceAnalytics) RecordSearch(query string) {
	analytics.mu.Lock()
	defer analytics.mu.Unlock()
	analytics.searchQueries[query]++
}

func (analytics *MarketplaceAnalytics) RecordDownload(templateID string) {
	analytics.mu.Lock()
	defer analytics.mu.Unlock()
	analytics.downloads[templateID]++
}
