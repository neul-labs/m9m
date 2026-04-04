package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (mm *MarketplaceManager) SearchMarketplace(query string, filters map[string]interface{}) ([]*MarketplaceTemplate, error) {
	mm.analytics.RecordSearch(query)

	cacheKey := mm.generateCacheKey(query, filters)
	if cached := mm.cache.GetCachedQuery(cacheKey); cached != nil {
		return cached.Results, nil
	}

	var results []*MarketplaceTemplate
	queryLower := strings.ToLower(query)

	mm.mu.RLock()
	for _, repo := range mm.repositories {
		if !repo.Enabled {
			continue
		}

		for _, template := range repo.Templates {
			if mm.matchesMarketplaceQuery(template, queryLower) && mm.matchesMarketplaceFilters(template, filters) {
				results = append(results, template)
			}
		}
	}
	mm.mu.RUnlock()

	mm.sortSearchResults(results, filters)

	mm.cache.CacheQuery(cacheKey, &CachedQuery{
		Query:     query,
		Results:   results,
		Timestamp: time.Now(),
		Filters:   filters,
	})

	return results, nil
}

func (mm *MarketplaceManager) matchesMarketplaceQuery(template *MarketplaceTemplate, query string) bool {
	if query == "" {
		return true
	}

	searchableText := strings.ToLower(fmt.Sprintf("%s %s %s %s %s",
		template.Name,
		template.Description,
		template.Author,
		strings.Join(template.Tags, " "),
		strings.Join(template.Metadata.Keywords, " ")))

	return strings.Contains(searchableText, query)
}

func (mm *MarketplaceManager) matchesMarketplaceFilters(template *MarketplaceTemplate, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "security_level":
			if template.SecurityReport.SecurityLevel != value.(string) {
				return false
			}
		case "min_rating":
			if template.Popularity.Rating < value.(float64) {
				return false
			}
		case "verified_only":
			if value.(bool) {
				repo := mm.repositories[template.RepositoryID]
				if !repo.Metadata.Verified {
					return false
				}
			}
		case "official_only":
			if value.(bool) {
				repo := mm.repositories[template.RepositoryID]
				if !repo.Metadata.Official {
					return false
				}
			}
		case "compatible_version":
			version := value.(string)
			if !mm.isVersionCompatible(template.Compatibility, version) {
				return false
			}
		}
	}

	return true
}

func (mm *MarketplaceManager) isVersionCompatible(compat *CompatibilityInfo, version string) bool {
	if compat == nil {
		return true
	}

	return true
}

func (mm *MarketplaceManager) sortSearchResults(results []*MarketplaceTemplate, filters map[string]interface{}) {
	sortBy := "relevance"
	if sortFilter, exists := filters["sort_by"]; exists {
		sortBy = sortFilter.(string)
	}

	switch sortBy {
	case "popularity":
		sort.Slice(results, func(i, j int) bool {
			return results[i].Popularity.TrendingScore > results[j].Popularity.TrendingScore
		})
	case "rating":
		sort.Slice(results, func(i, j int) bool {
			return results[i].Popularity.Rating > results[j].Popularity.Rating
		})
	case "downloads":
		sort.Slice(results, func(i, j int) bool {
			return results[i].Popularity.Downloads > results[j].Popularity.Downloads
		})
	case "newest":
		sort.Slice(results, func(i, j int) bool {
			return results[i].CreatedAt.After(results[j].CreatedAt)
		})
	case "updated":
		sort.Slice(results, func(i, j int) bool {
			return results[i].UpdatedAt.After(results[j].UpdatedAt)
		})
	}
}

func (mm *MarketplaceManager) generateCacheKey(query string, filters map[string]interface{}) string {
	data := fmt.Sprintf("%s_%v", query, filters)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

func (mm *MarketplaceManager) GetTrendingTemplates(limit int) ([]*MarketplaceTemplate, error) {
	var templates []*MarketplaceTemplate

	mm.mu.RLock()
	for _, repo := range mm.repositories {
		if !repo.Enabled {
			continue
		}
		templates = append(templates, repo.Templates...)
	}
	mm.mu.RUnlock()

	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Popularity.TrendingScore > templates[j].Popularity.TrendingScore
	})

	if len(templates) > limit {
		templates = templates[:limit]
	}

	return templates, nil
}

func (mm *MarketplaceManager) GetFeaturedTemplates() ([]*MarketplaceTemplate, error) {
	var featured []*MarketplaceTemplate

	mm.mu.RLock()
	for _, repo := range mm.repositories {
		if !repo.Metadata.Official {
			continue
		}

		for _, template := range repo.Templates {
			if template.Popularity.Rating >= 4.5 && template.Popularity.Downloads > 1000 {
				featured = append(featured, template)
			}
		}
	}
	mm.mu.RUnlock()

	return featured, nil
}
