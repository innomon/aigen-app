package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/innomon/aigen-cms/utils/datamodels"
	"github.com/mitchellh/mapstructure"
)

type AuditService struct {
	dao relationdbdao.IPrimaryDao
}

func NewAuditService(dao relationdbdao.IPrimaryDao) *AuditService {
	return &AuditService{dao: dao}
}

func (s *AuditService) List(ctx context.Context, pagination datamodels.Pagination) ([]*descriptors.AuditLog, error) {
	sb := s.dao.GetBuilder().Select("*").From(descriptors.AuditLogTableName).OrderBy("id DESC")

	// Apply pagination
	if pagination.Limit != nil {
		// ...
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*descriptors.AuditLog
	for rows.Next() {
		log, err := s.scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, log)
	}

	return results, nil
}

func (s *AuditService) ById(ctx context.Context, id int64) (*descriptors.AuditLog, error) {
	query, args, err := s.dao.GetBuilder().Select("*").From(descriptors.AuditLogTableName).
		Where(squirrel.Eq{"id": id}).Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	return s.scanAuditLog(rows)
}

func (s *AuditService) Log(ctx context.Context, l *descriptors.AuditLog) error {
	payloadJSON, _ := json.Marshal(l.Payload)
	now := time.Now()
	l.CreatedAt = now

	data := datamodels.Record{
		"user_id":      l.UserId,
		"user_name":    l.UserName,
		"action":       string(l.Action),
		"entity_name":  l.EntityName,
		"record_id":    l.RecordId,
		"record_label": l.RecordLabel,
		"payload":      string(payloadJSON),
		"created_at":   l.CreatedAt,
	}

	query, args, err := s.dao.GetBuilder().Insert(descriptors.AuditLogTableName).
		SetMap(data).ToSql()
	if err != nil {
		return err
	}

	_, err = s.dao.GetDb().ExecContext(ctx, query, args...)
	return err
}

func (s *AuditService) scanAuditLog(scanner interface {
	Scan(dest ...interface{}) error
	Columns() ([]string, error)
}) (*descriptors.AuditLog, error) {
	cols, _ := scanner.Columns()
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range cols {
		valuePtrs[i] = &values[i]
	}

	if err := scanner.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	record := make(map[string]interface{})
	for i, col := range cols {
		val := values[i]
		if b, ok := val.([]byte); ok {
			record[col] = string(b)
		} else {
			record[col] = val
		}
	}

	var l descriptors.AuditLog
	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &l,
		TagName:  "mapstructure",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			func(f interface{}, t interface{}) (interface{}, error) {
				if f == nil {
					return nil, nil
				}
				// Handle string to Payload (map[string]interface{})
				if str, ok := f.(string); ok {
					if fmt.Sprintf("%T", t) == "map[string]interface {}" {
						var payload map[string]interface{}
						if err := json.Unmarshal([]byte(str), &payload); err != nil {
							return nil, err
						}
						return payload, nil
					}
				}
				return f, nil
			},
		),
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(record); err != nil {
		return nil, err
	}

	return &l, nil
}
