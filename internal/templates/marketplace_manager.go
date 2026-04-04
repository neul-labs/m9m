package templates

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type MarketplaceManager struct {
	mu              sync.RWMutex
	templateManager *TemplateManager
	repositories    map[string]*Repository
	cache           *MarketplaceCache
	httpClient      *http.Client
	updateScheduler *UpdateScheduler
	ratingSystem    *RatingSystem
	analytics       *MarketplaceAnalytics
	securityScanner *SecurityScanner
}

func NewMarketplaceManager(tm *TemplateManager) *MarketplaceManager {
	mm := &MarketplaceManager{
		templateManager: tm,
		repositories:    make(map[string]*Repository),
		cache:           NewMarketplaceCache(1 * time.Hour),
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		updateScheduler: NewUpdateScheduler(24 * time.Hour),
		ratingSystem:    NewRatingSystem(),
		analytics:       NewMarketplaceAnalytics(),
		securityScanner: NewSecurityScanner(),
	}

	mm.initializeDefaultRepositories()
	mm.startUpdateScheduler()

	return mm
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
	_, exists := mm.repositories[repoID]
	mm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("repository not found: %s", repoID)
	}

	return mm.syncRepository(repoID)
}
