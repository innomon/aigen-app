package services

import (
	"context"
	"time"
	"fmt"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"encoding/json"
)

const NotificationNamespace = "aigen.core.descriptors.Notification"

type NotificationService struct {
	dao relationdbdao.IPrimaryDao
}

func NewNotificationService(dao relationdbdao.IPrimaryDao) *NotificationService {
	return &NotificationService{dao: dao}
}

func (s *NotificationService) List(ctx context.Context, userId string, pagination datamodels.Pagination) ([]*descriptors.Notification, error) {
	filters := []datamodels.Filter{
		{FieldName: "userId", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{userId}}}},
		{FieldName: "deleted", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{false}}}},
	}

	recs, _, err := s.dao.List(ctx, NotificationNamespace, filters, pagination, []datamodels.Sort{{Field: "createdAt", Order: datamodels.SortOrderDesc}})
	if err != nil {
		return nil, err
	}

	var results []*descriptors.Notification
	for _, r := range recs {
		var notif descriptors.Notification
		data, _ := json.Marshal(r.Rec)
		json.Unmarshal(data, &notif)
		results = append(results, &notif)
	}
	return results, nil
}

func (s *NotificationService) Send(ctx context.Context, n *descriptors.Notification) error {
	id, _ := gonanoid.New(12)
	now := time.Now()
	n.Id = id
	n.CreatedAt = now
	n.UpdatedAt = now

	rec := datamodels.RecJSON{
		Namespace: NotificationNamespace,
		Key:       id,
		Rec:       n,
		Tmstamp:   now,
	}

	return s.dao.Save(ctx, rec)
}

func (s *NotificationService) MarkAsRead(ctx context.Context, userId string, id string) error {
	rec, err := s.dao.Get(ctx, NotificationNamespace, id)
	if err != nil || rec == nil {
		return fmt.Errorf("notification not found")
	}

	var notif descriptors.Notification
	data, _ := json.Marshal(rec.Rec)
	json.Unmarshal(data, &notif)

	if notif.UserId != userId {
		return fmt.Errorf("access denied")
	}

	notif.IsRead = true
	notif.UpdatedAt = time.Now()
	rec.Rec = notif
	return s.dao.Save(ctx, *rec)
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userId string) error {
	filters := []datamodels.Filter{
		{FieldName: "userId", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{userId}}}},
		{FieldName: "isRead", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{false}}}},
	}

	recs, _, _ := s.dao.List(ctx, NotificationNamespace, filters, datamodels.Pagination{}, nil)
	for _, r := range recs {
		var notif descriptors.Notification
		data, _ := json.Marshal(r.Rec)
		json.Unmarshal(data, &notif)
		notif.IsRead = true
		notif.UpdatedAt = time.Now()
		r.Rec = notif
		s.dao.Save(ctx, r)
	}
	return nil
}
