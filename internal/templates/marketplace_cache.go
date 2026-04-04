package templates

import "time"

func NewMarketplaceCache(ttl time.Duration) *MarketplaceCache {
	return &MarketplaceCache{
		templates: make(map[string]*MarketplaceTemplate),
		queries:   make(map[string]*CachedQuery),
		ttl:       ttl,
	}
}

func (cache *MarketplaceCache) AddTemplate(template *MarketplaceTemplate) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.templates[template.ID] = template
}

func (cache *MarketplaceCache) GetCachedQuery(key string) *CachedQuery {
	cache.mu.RLock()
	query, exists := cache.queries[key]
	cache.mu.RUnlock()

	if exists {
		if time.Since(query.Timestamp) < cache.ttl {
			return query
		}

		cache.mu.Lock()
		delete(cache.queries, key)
		cache.mu.Unlock()
	}

	return nil
}

func (cache *MarketplaceCache) CacheQuery(key string, query *CachedQuery) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.queries[key] = query
}
