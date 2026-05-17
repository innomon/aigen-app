package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/innomon/aigen-app/utils/ids"
)

const AuditLogNamespace = "aigen.core.descriptors.AuditLog"

type AuditService struct {
	dao relationdbdao.IPrimaryDao
}

func NewAuditService(dao relationdbdao.IPrimaryDao) *AuditService {
	return &AuditService{dao: dao}
}

func (s *AuditService) List(ctx context.Context, pagination datamodels.Pagination) ([]*descriptors.AuditLog, error) {
	recs, _, err := s.dao.List(ctx, AuditLogNamespace, nil, pagination, []datamodels.Sort{{Field: "createdAt", Order: datamodels.SortOrderDesc}})
	if err != nil {
		return nil, err
	}

	var results []*descriptors.AuditLog
	for _, r := range recs {
		var log descriptors.AuditLog
		data, _ := json.Marshal(r.Rec)
		json.Unmarshal(data, &log)
		results = append(results, &log)
	}
	return results, nil
}

func (s *AuditService) ById(ctx context.Context, id string) (*descriptors.AuditLog, error) {
	rec, err := s.dao.Get(ctx, AuditLogNamespace, id)
	if err != nil || rec == nil {
		return nil, fmt.Errorf("audit log not found")
	}

	var log descriptors.AuditLog
	data, _ := json.Marshal(rec.Rec)
	json.Unmarshal(data, &log)
	return &log, nil
}

func (s *AuditService) Log(ctx context.Context, log *descriptors.AuditLog) error {
	log.Id = ids.NewRandomID()
	log.CreatedAt = time.Now()

	rec := datamodels.RecJSON{
		Namespace: AuditLogNamespace,
		Key:       log.Id,
		Rec:       log,
		Tmstamp:   log.CreatedAt,
	}

	return s.dao.Save(ctx, rec)
}
