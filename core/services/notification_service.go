package services

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/innomon/aigen-cms/utils/datamodels"
	"github.com/mitchellh/mapstructure"
)

type NotificationService struct {
	dao relationdbdao.IPrimaryDao
}

func NewNotificationService(dao relationdbdao.IPrimaryDao) *NotificationService {
	return &NotificationService{dao: dao}
}

func (s *NotificationService) List(ctx context.Context, userId string, pagination datamodels.Pagination) ([]*descriptors.Notification, error) {
	sb := s.dao.GetBuilder().Select("*").From(descriptors.NotificationTableName).
		Where(squirrel.Eq{"user_id": userId, "deleted": false}).
		OrderBy("id DESC")

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

	var results []*descriptors.Notification
	for rows.Next() {
		notif, err := s.scanNotification(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, notif)
	}

	return results, nil
}

func (s *NotificationService) Send(ctx context.Context, n *descriptors.Notification) error {
	now := time.Now()
	n.CreatedAt = now
	n.UpdatedAt = now

	data := datamodels.Record{
		"user_id":      n.UserId,
		"sender_id":    n.SenderId,
		"action_type":  n.ActionType,
		"message_type": n.MessageType,
		"message":      n.Message,
		"url":          n.Url,
		"is_read":      n.IsRead,
		"created_at":   n.CreatedAt,
		"updated_at":   n.UpdatedAt,
	}

	query, args, err := s.dao.GetBuilder().Insert(descriptors.NotificationTableName).
		SetMap(data).ToSql()
	if err != nil {
		return err
	}

	_, err = s.dao.GetDb().ExecContext(ctx, query, args...)
	return err
}

func (s *NotificationService) MarkAsRead(ctx context.Context, userId string, id int64) error {
	query, args, err := s.dao.GetBuilder().Update(descriptors.NotificationTableName).
		Set("is_read", true).
		Where(squirrel.Eq{"id": id, "user_id": userId}).ToSql()
	if err != nil {
		return err
	}

	_, err = s.dao.GetDb().ExecContext(ctx, query, args...)
	return err
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userId string) error {
	query, args, err := s.dao.GetBuilder().Update(descriptors.NotificationTableName).
		Set("is_read", true).
		Where(squirrel.Eq{"user_id": userId, "is_read": false}).ToSql()
	if err != nil {
		return err
	}

	_, err = s.dao.GetDb().ExecContext(ctx, query, args...)
	return err
}

func (s *NotificationService) scanNotification(scanner interface {
	Scan(dest ...interface{}) error
	Columns() ([]string, error)
}) (*descriptors.Notification, error) {
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

	var n descriptors.Notification
	if err := mapstructure.Decode(record, &n); err != nil {
		return nil, err
	}
	return &n, nil
}
