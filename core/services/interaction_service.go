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

type InteractionService struct {
	dao relationdbdao.IPrimaryDao
}

func NewInteractionService(dao relationdbdao.IPrimaryDao) *InteractionService {
	return &InteractionService{dao: dao}
}

func (s *InteractionService) Log(ctx context.Context, i *descriptors.Interaction) error {
	if i.Id == "" {
		i.Id = ids.NewRandomID()
	}
	if i.CreatedAt.IsZero() {
		i.CreatedAt = time.Now()
	}

	rec := datamodels.RecJSON{
		Namespace: descriptors.InteractionNamespace,
		Key:       i.Id,
		Rec:       i,
		Tmstamp:   i.CreatedAt,
	}

	return s.dao.Save(ctx, rec)
}

func (s *InteractionService) GetHistory(ctx context.Context, identifier string, limit int) ([]*descriptors.Interaction, error) {
	filters := []datamodels.Filter{
		{
			FieldName: "identifier",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{identifier}},
			},
		},
	}

	limitStr := fmt.Sprintf("%d", limit)
	pagination := datamodels.Pagination{Limit: &limitStr}
	sorts := []datamodels.Sort{{Field: "createdAt", Order: datamodels.SortOrderDesc}}

	recs, _, err := s.dao.List(ctx, descriptors.InteractionNamespace, filters, pagination, sorts)
	if err != nil {
		return nil, err
	}

	var results []*descriptors.Interaction
	for _, r := range recs {
		var interaction descriptors.Interaction
		data, _ := json.Marshal(r.Rec)
		json.Unmarshal(data, &interaction)
		results = append(results, &interaction)
	}
	return results, nil
}

func (s *InteractionService) UpdateStatus(ctx context.Context, id string, status string, errStr string) error {
	rec, err := s.dao.Get(ctx, descriptors.InteractionNamespace, id)
	if err != nil || rec == nil {
		return fmt.Errorf("interaction %s not found: %w", id, err)
	}

	var i descriptors.Interaction
	data, _ := json.Marshal(rec.Rec)
	json.Unmarshal(data, &i)

	i.Status = status
	if errStr != "" {
		if i.Metadata == nil {
			i.Metadata = make(map[string]interface{})
		}
		i.Metadata["error"] = errStr
	}

	rec.Rec = i
	rec.Tmstamp = time.Now()
	return s.dao.Save(ctx, *rec)
}

func (s *InteractionService) GetPendingOutbound(ctx context.Context, channel descriptors.ChannelType) ([]*descriptors.Interaction, error) {
	filters := []datamodels.Filter{
		{
			FieldName: "channelType",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{channel}},
			},
		},
		{
			FieldName: "direction",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{"outbound"}},
			},
		},
		{
			FieldName: "status",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{"pending"}},
			},
		},
	}

	recs, _, err := s.dao.List(ctx, descriptors.InteractionNamespace, filters, datamodels.Pagination{}, nil)
	if err != nil {
		return nil, err
	}

	var results []*descriptors.Interaction
	for _, r := range recs {
		var interaction descriptors.Interaction
		data, _ := json.Marshal(r.Rec)
		json.Unmarshal(data, &interaction)
		results = append(results, &interaction)
	}
	return results, nil
}
