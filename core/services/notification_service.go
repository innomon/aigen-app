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
	}
	recs, _, err := s.dao.List(ctx, NotificationNamespace, filters, pagination, []datamodels.Sort{{Field: "createdAt", Order: datamodels.SortOrderDesc}})
	if err != nil {
		return nil, err
	}

	var results []*descriptors.Notification
	for _, r := range recs {
		var n descriptors.Notification
		data, _ := json.Marshal(r.Rec)
		json.Unmarshal(data, &n)
		results = append(results, &n)
	}
	return results, nil
}

func (s *NotificationService) Send(ctx context.Context, n *descriptors.Notification) error {
	n.Id = ids.NewRandomID()
	n.CreatedAt = time.Now()
	n.UpdatedAt = time.Now()

	rec := datamodels.RecJSON{
		Namespace: NotificationNamespace,
		Key:       n.Id,
		Rec:       n,
		Tmstamp:   n.CreatedAt,
	}

	return s.dao.Save(ctx, rec)
}

func (s *NotificationService) MarkAsRead(ctx context.Context, userId string, id string) error {
	rec, err := s.dao.Get(ctx, NotificationNamespace, id)
	if err != nil || rec == nil {
		return fmt.Errorf("notification not found")
	}

	var n descriptors.Notification
	data, _ := json.Marshal(rec.Rec)
	json.Unmarshal(data, &n)

	if n.UserId != userId {
		return fmt.Errorf("access denied")
	}

	n.IsRead = true
	n.UpdatedAt = time.Now()
	rec.Rec = n
	rec.Tmstamp = n.UpdatedAt
	return s.dao.Save(ctx, *rec)
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userId string) error {
	filters := []datamodels.Filter{
		{FieldName: "userId", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{userId}}}},
		{FieldName: "isRead", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{false}}}},
	}

	recs, _, err := s.dao.List(ctx, NotificationNamespace, filters, datamodels.Pagination{}, nil)
	if err != nil {
		return err
	}

	for _, r := range recs {
		var n descriptors.Notification
		data, _ := json.Marshal(r.Rec)
		json.Unmarshal(data, &n)
		n.IsRead = true
		n.UpdatedAt = time.Now()
		r.Rec = n
		r.Tmstamp = n.UpdatedAt
		s.dao.Save(ctx, r)
	}

	return nil
}
