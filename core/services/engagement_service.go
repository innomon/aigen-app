package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
)

const (
	EngagementStatusNamespace = "aigen.core.descriptors.EngagementStatus"
	EngagementCountNamespace  = "aigen.core.descriptors.EngagementCount"
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
	for _, status := range buffer {
		key := fmt.Sprintf("%s_%v_%s_%v", status.EntityName, status.RecordId, status.EngagementType, status.UserId)
		
		existing, _ := s.dao.Get(ctx, EngagementStatusNamespace, key)
		inserted := existing == nil

		rec := datamodels.RecJSON{
			Namespace: EngagementStatusNamespace,
			Key:       key,
			Rec:       status,
			Tmstamp:   time.Now(),
		}
		if err := s.dao.Save(ctx, rec); err != nil {
			log.Printf("Failed to save engagement status: %v", err)
			continue
		}

		delta := int64(1)
		if !status.IsActive {
			delta = -1
		}

		if status.EngagementType != "visit" || inserted {
			s.flushCounts(ctx, status, delta)
		}
	}
}

func (s *EngagementService) flushCounts(ctx context.Context, status *descriptors.EngagementStatus, delta int64) {
	key := fmt.Sprintf("%s_%v_%s", status.EntityName, status.RecordId, status.EngagementType)
	
	rec, err := s.dao.Get(ctx, EngagementCountNamespace, key)
	if err != nil {
		log.Printf("Failed to get engagement counts: %v", err)
		return
	}

	count := int64(0)
	if rec != nil {
		if c, ok := rec.Rec.(float64); ok {
			count = int64(c)
		} else if c, ok := rec.Rec.(int64); ok {
			count = c
		}
	}

	count += delta
	s.dao.Save(ctx, datamodels.RecJSON{
		Namespace: EngagementCountNamespace,
		Key:       key,
		Rec:       count,
		Tmstamp:   time.Now(),
	})
}
