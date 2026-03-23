package services

import (
	"context"
	"log"
	"time"

	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type EngagementService struct {
	dao relationdbdao.IPrimaryDao
	ch  chan *descriptors.EngagementStatus
}

func NewEngagementService(dao relationdbdao.IPrimaryDao) *EngagementService {
	s := &EngagementService{
		dao: dao,
		ch:  make(chan *descriptors.EngagementStatus, 100),
	}
	go s.startFlushWorker()
	return s
}

func (s *EngagementService) Track(ctx context.Context, status *descriptors.EngagementStatus) error {
	s.ch <- status
	return nil
}

func (s *EngagementService) startFlushWorker() {
	ticker := time.NewTicker(5 * time.Second)
	var buffer []*descriptors.EngagementStatus

	for {
		select {
		case status := <-s.ch:
			buffer = append(buffer, status)
			if len(buffer) >= 50 {
				s.flush(buffer)
				buffer = nil
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				s.flush(buffer)
				buffer = nil
			}
		}
	}
}

func (s *EngagementService) flush(buffer []*descriptors.EngagementStatus) {
	ctx := context.Background()
	
	// Group by status
	for _, status := range buffer {
		data := datamodels.Record{
			"entity_name":     status.EntityName,
			"record_id":       status.RecordId,
			"engagement_type": status.EngagementType,
			"user_id":         status.UserId,
			"is_active":       status.IsActive,
			"title":           status.Title,
			"url":             status.Url,
			"image":           status.Image,
			"subtitle":        status.Subtitle,
			"updated_at":      time.Now(),
		}
		
		keyFields := []string{"entity_name", "record_id", "engagement_type", "user_id"}
		inserted, err := s.dao.UpdateOnConflict(ctx, descriptors.EngagementStatusTableName, data, keyFields)
		if err != nil {
			log.Printf("Failed to flush engagement status: %v", err)
			continue
		}

		// Update counts
		delta := int64(1)
		if !status.IsActive {
			delta = -1
		}
		
		// If it was an update (not inserted) and isActive changed, we need to adjust count.
		// Actually, C# implementation usually increments on every track if it's a "visit", 
		// but for "like" it's more about status.
		// Simplified count update:
		if status.EngagementType != "visit" || inserted {
			s.flushCounts(ctx, status, delta)
		}
	}
}

func (s *EngagementService) flushCounts(ctx context.Context, status *descriptors.EngagementStatus, delta int64) {
	keyConditions := datamodels.Record{
		"entity_name":     status.EntityName,
		"record_id":       status.RecordId,
		"engagement_type": status.EngagementType,
	}
	
	_, err := s.dao.Increase(ctx, descriptors.EngagementCountTableName, keyConditions, "count", 0, delta)
	if err != nil {
		log.Printf("Failed to flush engagement counts: %v", err)
	}
}
