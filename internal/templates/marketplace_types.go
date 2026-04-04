package templates

import (
	"net/http"
	"sync"
	"time"

	"github.com/neul-labs/m9m/internal/expressions"
)

type Repository struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	URL            string                 `json:"url"`
	Type           string                 `json:"type"`
	Enabled        bool                   `json:"enabled"`
	Authentication *RepositoryAuth        `json:"authentication,omitempty"`
	Metadata       *RepositoryMetadata    `json:"metadata"`
	Templates      []*MarketplaceTemplate `json:"templates"`
	LastSync       time.Time              `json:"last_sync"`
	SyncInterval   time.Duration          `json:"sync_interval"`
}

type RepositoryAuth struct {
	Type     string            `json:"type"`
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Token    string            `json:"token,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

type RepositoryMetadata struct {
	Description    string    `json:"description"`
	Owner          string    `json:"owner"`
	Tags           []string  `json:"tags"`
	Official       bool      `json:"official"`
	Verified       bool      `json:"verified"`
	TotalTemplates int       `json:"total_templates"`
	LastUpdated    time.Time `json:"last_updated"`
}

type MarketplaceTemplate struct {
	*WorkflowTemplate
	RepositoryID   string                  `json:"repository_id"`
	RemoteURL      string                  `json:"remote_url"`
	Checksum       string                  `json:"checksum"`
	Popularity     *PopularityMetrics      `json:"popularity"`
	Compatibility  *CompatibilityInfo      `json:"compatibility"`
	SecurityReport *SecurityReport         `json:"security_report"`
	InstallStats   *InstallationStatistics `json:"install_stats"`
	UpdateInfo     *UpdateInformation      `json:"update_info"`
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
	SecurityLevel   string           `json:"security_level"`
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
	Query     string                 `json:"query"`
	Results   []*MarketplaceTemplate `json:"results"`
	Timestamp time.Time              `json:"timestamp"`
	Filters   map[string]interface{} `json:"filters"`
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
	Score      int       `json:"score"`
	Comment    string    `json:"comment,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	Helpful    int       `json:"helpful"`
}

type MarketplaceAnalytics struct {
	mu            sync.RWMutex
	searchQueries map[string]int
	downloads     map[string]int64
	errors        map[string]int
	trends        *TrendAnalysis
}

type TrendAnalysis struct {
	TopTemplates     []string           `json:"top_templates"`
	TrendingKeywords []string           `json:"trending_keywords"`
	CategoryTrends   map[string]float64 `json:"category_trends"`
	SeasonalPatterns map[string][]int   `json:"seasonal_patterns"`
}

type SecurityScanner struct {
	rules     map[string]*SecurityRule
	whitelist map[string]bool
	blacklist map[string]bool
	evaluator *expressions.GojaExpressionEvaluator
}

type SecurityRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Pattern     string `json:"pattern"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

var _ = http.MethodGet
