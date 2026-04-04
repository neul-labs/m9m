package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

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
		ID:      "official",
		Name:    "Official m9m Templates",
		URL:     "https://marketplace.m9m.dev/api/templates",
		Type:    "http",
		Enabled: true,
		Metadata: &RepositoryMetadata{
			Description: "Official templates maintained by the m9m team",
			Owner:       "m9m",
			Official:    true,
			Verified:    true,
		},
		SyncInterval: 6 * time.Hour,
	}

	mm.repositories[officialRepo.ID] = officialRepo
}

func NewUpdateScheduler(interval time.Duration) *UpdateScheduler {
	return &UpdateScheduler{
		interval: interval,
		stopChan: make(chan bool),
	}
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
