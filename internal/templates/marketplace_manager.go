package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dipankar/n8n-go/internal/expressions"
)

type MarketplaceManager struct {
	mu                sync.RWMutex
	templateManager   *TemplateManager
	repositories      map[string]*Repository
	cache             *MarketplaceCache
	httpClient        *http.Client
	updateScheduler   *UpdateScheduler
	ratingSystem      *RatingSystem
	analytics         *MarketplaceAnalytics
	securityScanner   *SecurityScanner
}

type Repository struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	URL          string                 `json:"url"`
	Type         string                 `json:"type"` // "git", "http", "local"
	Enabled      bool                   `json:"enabled"`
	Authentication *RepositoryAuth      `json:"authentication,omitempty"`
	Metadata     *RepositoryMetadata    `json:"metadata"`
	Templates    []*MarketplaceTemplate `json:"templates"`
	LastSync     time.Time              `json:"last_sync"`
	SyncInterval time.Duration          `json:"sync_interval"`
}

type RepositoryAuth struct {
	Type     string            `json:"type"` // "none", "basic", "token", "oauth"
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Token    string            `json:"token,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

type RepositoryMetadata struct {
	Description  string   `json:"description"`
	Owner        string   `json:"owner"`
	Tags         []string `json:"tags"`
	Official     bool     `json:"official"`
	Verified     bool     `json:"verified"`
	TotalTemplates int    `json:"total_templates"`
	LastUpdated  time.Time `json:"last_updated"`
}

type MarketplaceTemplate struct {
	*WorkflowTemplate
	RepositoryID    string                     `json:"repository_id"`
	RemoteURL       string                     `json:"remote_url"`
	Checksum        string                     `json:"checksum"`
	Popularity      *PopularityMetrics         `json:"popularity"`
	Compatibility   *CompatibilityInfo         `json:"compatibility"`
	SecurityReport  *SecurityReport            `json:"security_report"`
	InstallStats    *InstallationStatistics    `json:"install_stats"`
	UpdateInfo      *UpdateInformation         `json:"update_info"`
}

type PopularityMetrics struct {
	Downloads       int64   `json:"downloads"`
	WeeklyDownloads int64   `json:"weekly_downloads"`
	Stars           int     `json:"stars"`
	Forks           int     `json:"forks"`
	Rating          float64 `json:"rating"`
	Reviews         int     `json:"reviews"`
	TrendingScore   float64 `json:"trending_score"`
}

type CompatibilityInfo struct {
	MinVersion       string   `json:"min_version"`
	MaxVersion       string   `json:"max_version"`
	TestedVersions   []string `json:"tested_versions"`
	CompatibleNodes  []string `json:"compatible_nodes"`
	RequiredFeatures []string `json:"required_features"`
	Warnings         []string `json:"warnings"`
}

type SecurityReport struct {
	ScanDate        time.Time        `json:"scan_date"`
	SecurityLevel   string           `json:"security_level"` // "safe", "warning", "danger"
	Vulnerabilities []*Vulnerability `json:"vulnerabilities"`
	CodeQuality     *CodeQuality     `json:"code_quality"`
	TrustScore      float64          `json:"trust_score"`
}

type Vulnerability struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Mitigation  string `json:"mitigation,omitempty"`
}

type CodeQuality struct {
	Score           float64  `json:"score"`
	Issues          []string `json:"issues"`
	BestPractices   []string `json:"best_practices"`
	ComplexityScore float64  `json:"complexity_score"`
}

type InstallationStatistics struct {
	TotalInstalls     int64             `json:"total_installs"`
	RecentInstalls    int64             `json:"recent_installs"`
	SuccessRate       float64           `json:"success_rate"`
	FailureReasons    map[string]int    `json:"failure_reasons"`
	PopularParameters map[string]string `json:"popular_parameters"`
}

type UpdateInformation struct {
	Available       bool      `json:"available"`
	LatestVersion   string    `json:"latest_version"`
	CurrentVersion  string    `json:"current_version"`
	ChangeLog       []string  `json:"change_log"`
	BreakingChanges bool      `json:"breaking_changes"`
	UpdateDate      time.Time `json:"update_date"`
}

type MarketplaceCache struct {
	mu        sync.RWMutex
	templates map[string]*MarketplaceTemplate
	queries   map[string]*CachedQuery
	ttl       time.Duration
}

type CachedQuery struct {
	Query     string                   `json:"query"`
	Results   []*MarketplaceTemplate   `json:"results"`
	Timestamp time.Time                `json:"timestamp"`
	Filters   map[string]interface{}   `json:"filters"`
}

type UpdateScheduler struct {
	ticker   *time.Ticker
	stopChan chan bool
	interval time.Duration
}

type RatingSystem struct {
	mu      sync.RWMutex
	ratings map[string][]*Rating
}

type Rating struct {
	UserID     string    `json:"user_id"`
	TemplateID string    `json:"template_id"`
	Score      int       `json:"score"` // 1-5
	Comment    string    `json:"comment,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	Helpful    int       `json:"helpful"`
}

type MarketplaceAnalytics struct {
	mu           sync.RWMutex
	searchQueries map[string]int
	downloads     map[string]int64
	errors        map[string]int
	trends        *TrendAnalysis
}

type TrendAnalysis struct {
	TopTemplates     []string          `json:"top_templates"`
	TrendingKeywords []string          `json:"trending_keywords"`
	CategoryTrends   map[string]float64 `json:"category_trends"`
	SeasonalPatterns map[string][]int   `json:"seasonal_patterns"`
}

type SecurityScanner struct {
	rules           map[string]*SecurityRule
	whitelist       map[string]bool
	blacklist       map[string]bool
	evaluator       *expressions.GojaExpressionEvaluator
}

type SecurityRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Pattern     string `json:"pattern"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

func NewMarketplaceManager(tm *TemplateManager) *MarketplaceManager {
	mm := &MarketplaceManager{
		templateManager:   tm,
		repositories:      make(map[string]*Repository),
		cache:            NewMarketplaceCache(1 * time.Hour),
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		updateScheduler:  NewUpdateScheduler(24 * time.Hour),
		ratingSystem:     NewRatingSystem(),
		analytics:        NewMarketplaceAnalytics(),
		securityScanner:  NewSecurityScanner(),
	}

	mm.initializeDefaultRepositories()
	mm.startUpdateScheduler()

	return mm
}

func NewMarketplaceCache(ttl time.Duration) *MarketplaceCache {
	return &MarketplaceCache{
		templates: make(map[string]*MarketplaceTemplate),
		queries:   make(map[string]*CachedQuery),
		ttl:       ttl,
	}
}

func NewUpdateScheduler(interval time.Duration) *UpdateScheduler {
	return &UpdateScheduler{
		interval: interval,
		stopChan: make(chan bool),
	}
}

func NewRatingSystem() *RatingSystem {
	return &RatingSystem{
		ratings: make(map[string][]*Rating),
	}
}

func NewMarketplaceAnalytics() *MarketplaceAnalytics {
	return &MarketplaceAnalytics{
		searchQueries: make(map[string]int),
		downloads:     make(map[string]int64),
		errors:        make(map[string]int),
		trends:        &TrendAnalysis{
			CategoryTrends:   make(map[string]float64),
			SeasonalPatterns: make(map[string][]int),
		},
	}
}

func NewSecurityScanner() *SecurityScanner {
	scanner := &SecurityScanner{
		rules:     make(map[string]*SecurityRule),
		whitelist: make(map[string]bool),
		blacklist: make(map[string]bool),
		evaluator: expressions.NewGojaExpressionEvaluator(),
	}

	scanner.initializeSecurityRules()
	return scanner
}

func (mm *MarketplaceManager) AddRepository(repo *Repository) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.repositories[repo.ID]; exists {
		return fmt.Errorf("repository already exists: %s", repo.ID)
	}

	mm.repositories[repo.ID] = repo

	if repo.Enabled {
		go mm.syncRepository(repo.ID)
	}

	return nil
}

func (mm *MarketplaceManager) SyncRepository(repoID string) error {
	mm.mu.RLock()
	repo, exists := mm.repositories[repoID]
	mm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("repository not found: %s", repoID)
	}

	return mm.syncRepository(repoID)
}

func (mm *MarketplaceManager) syncRepository(repoID string) error {
	repo := mm.repositories[repoID]

	switch repo.Type {
	case "http":
		return mm.syncHTTPRepository(repo)
	case "git":
		return mm.syncGitRepository(repo)
	case "local":
		return mm.syncLocalRepository(repo)
	default:
		return fmt.Errorf("unsupported repository type: %s", repo.Type)
	}
}

func (mm *MarketplaceManager) syncHTTPRepository(repo *Repository) error {
	req, err := http.NewRequest("GET", repo.URL, nil)
	if err != nil {
		return err
	}

	if repo.Authentication != nil {
		mm.addAuthentication(req, repo.Authentication)
	}

	resp, err := mm.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var templates []*MarketplaceTemplate
	if err := json.Unmarshal(body, &templates); err != nil {
		return err
	}

	mm.mu.Lock()
	repo.Templates = templates
	repo.LastSync = time.Now()
	mm.mu.Unlock()

	for _, template := range templates {
		template.RepositoryID = repo.ID
		mm.processMarketplaceTemplate(template)
	}

	return nil
}

func (mm *MarketplaceManager) syncGitRepository(repo *Repository) error {
	return fmt.Errorf("git repository sync not implemented yet")
}

func (mm *MarketplaceManager) syncLocalRepository(repo *Repository) error {
	return fmt.Errorf("local repository sync not implemented yet")
}

func (mm *MarketplaceManager) addAuthentication(req *http.Request, auth *RepositoryAuth) {
	switch auth.Type {
	case "basic":
		req.SetBasicAuth(auth.Username, auth.Password)
	case "token":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	}

	for key, value := range auth.Headers {
		req.Header.Set(key, value)
	}
}

func (mm *MarketplaceManager) processMarketplaceTemplate(template *MarketplaceTemplate) {
	template.Checksum = mm.calculateChecksum(template)

	if template.SecurityReport == nil {
		template.SecurityReport = mm.securityScanner.ScanTemplate(template)
	}

	if template.Popularity == nil {
		template.Popularity = &PopularityMetrics{}
	}

	mm.cache.AddTemplate(template)
}

func (mm *MarketplaceManager) calculateChecksum(template *MarketplaceTemplate) string {
	data, _ := json.Marshal(template.WorkflowTemplate)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

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

	cachedQuery := &CachedQuery{
		Query:     query,
		Results:   results,
		Timestamp: time.Now(),
		Filters:   filters,
	}
	mm.cache.CacheQuery(cacheKey, cachedQuery)

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

func (mm *MarketplaceManager) RateTemplate(templateID, userID string, score int, comment string) error {
	return mm.ratingSystem.AddRating(&Rating{
		UserID:     userID,
		TemplateID: templateID,
		Score:      score,
		Comment:    comment,
		CreatedAt:  time.Now(),
	})
}

func (mm *MarketplaceManager) GetTemplateRating(templateID string) (float64, int, error) {
	return mm.ratingSystem.GetAverageRating(templateID)
}

func (mm *MarketplaceManager) startUpdateScheduler() {
	mm.updateScheduler.Start(func() {
		mm.syncAllRepositories()
	})
}

func (mm *MarketplaceManager) syncAllRepositories() {
	mm.mu.RLock()
	repos := make([]*Repository, 0, len(mm.repositories))
	for _, repo := range mm.repositories {
		if repo.Enabled {
			repos = append(repos, repo)
		}
	}
	mm.mu.RUnlock()

	for _, repo := range repos {
		if time.Since(repo.LastSync) >= repo.SyncInterval {
			go mm.syncRepository(repo.ID)
		}
	}
}

func (mm *MarketplaceManager) initializeDefaultRepositories() {
	officialRepo := &Repository{
		ID:   "official",
		Name: "Official n8n-go Templates",
		URL:  "https://marketplace.n8n-go.dev/api/templates",
		Type: "http",
		Enabled: true,
		Metadata: &RepositoryMetadata{
			Description: "Official templates maintained by the n8n-go team",
			Owner:       "n8n-go",
			Official:    true,
			Verified:    true,
		},
		SyncInterval: 6 * time.Hour,
	}

	mm.repositories[officialRepo.ID] = officialRepo
}

func (cache *MarketplaceCache) AddTemplate(template *MarketplaceTemplate) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.templates[template.ID] = template
}

func (cache *MarketplaceCache) GetCachedQuery(key string) *CachedQuery {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if query, exists := cache.queries[key]; exists {
		if time.Since(query.Timestamp) < cache.ttl {
			return query
		}
		delete(cache.queries, key)
	}

	return nil
}

func (cache *MarketplaceCache) CacheQuery(key string, query *CachedQuery) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.queries[key] = query
}

func (scheduler *UpdateScheduler) Start(callback func()) {
	scheduler.ticker = time.NewTicker(scheduler.interval)
	go func() {
		for {
			select {
			case <-scheduler.ticker.C:
				callback()
			case <-scheduler.stopChan:
				return
			}
		}
	}()
}

func (scheduler *UpdateScheduler) Stop() {
	if scheduler.ticker != nil {
		scheduler.ticker.Stop()
	}
	scheduler.stopChan <- true
}

func (rs *RatingSystem) AddRating(rating *Rating) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rating.Score < 1 || rating.Score > 5 {
		return fmt.Errorf("rating score must be between 1 and 5")
	}

	rs.ratings[rating.TemplateID] = append(rs.ratings[rating.TemplateID], rating)
	return nil
}

func (rs *RatingSystem) GetAverageRating(templateID string) (float64, int, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	ratings, exists := rs.ratings[templateID]
	if !exists || len(ratings) == 0 {
		return 0, 0, nil
	}

	total := 0
	for _, rating := range ratings {
		total += rating.Score
	}

	average := float64(total) / float64(len(ratings))
	return average, len(ratings), nil
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

func (scanner *SecurityScanner) ScanTemplate(template *MarketplaceTemplate) *SecurityReport {
	report := &SecurityReport{
		ScanDate:        time.Now(),
		SecurityLevel:   "safe",
		Vulnerabilities: []*Vulnerability{},
		CodeQuality:     &CodeQuality{Score: 100.0},
		TrustScore:      1.0,
	}

	templateJSON, _ := json.Marshal(template.WorkflowTemplate)
	content := string(templateJSON)

	for _, rule := range scanner.rules {
		if scanner.matchesPattern(content, rule.Pattern) {
			vulnerability := &Vulnerability{
				ID:          rule.ID,
				Type:        rule.Name,
				Severity:    rule.Severity,
				Description: rule.Description,
				Mitigation:  rule.Remediation,
			}
			report.Vulnerabilities = append(report.Vulnerabilities, vulnerability)

			if rule.Severity == "high" || rule.Severity == "critical" {
				report.SecurityLevel = "danger"
				report.TrustScore *= 0.5
			} else if rule.Severity == "medium" {
				if report.SecurityLevel == "safe" {
					report.SecurityLevel = "warning"
				}
				report.TrustScore *= 0.8
			}
		}
	}

	return report
}

func (scanner *SecurityScanner) matchesPattern(content, pattern string) bool {
	return strings.Contains(strings.ToLower(content), strings.ToLower(pattern))
}

func (scanner *SecurityScanner) initializeSecurityRules() {
	scanner.rules["hardcoded_credentials"] = &SecurityRule{
		ID:          "hardcoded_credentials",
		Name:        "Hardcoded Credentials",
		Pattern:     "password|secret|key|token",
		Severity:    "high",
		Description: "Template may contain hardcoded credentials",
		Remediation: "Use parameter substitution for sensitive values",
	}

	scanner.rules["external_scripts"] = &SecurityRule{
		ID:          "external_scripts",
		Name:        "External Scripts",
		Pattern:     "eval|function|script",
		Severity:    "medium",
		Description: "Template contains executable code",
		Remediation: "Review code execution for security risks",
	}
}