package templates

import (
	"fmt"
	"time"
)

func NewRatingSystem() *RatingSystem {
	return &RatingSystem{
		ratings: make(map[string][]*Rating),
	}
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
