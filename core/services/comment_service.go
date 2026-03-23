package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/oklog/ulid/v2"
	"encoding/json"
)

const CommentNamespace = "aigen.core.descriptors.Comment"

type CommentService struct {
	dao relationdbdao.IPrimaryDao
}

func NewCommentService(dao relationdbdao.IPrimaryDao) *CommentService {
	return &CommentService{dao: dao}
}

func (s *CommentService) List(ctx context.Context, entityName string, recordId int64, pagination datamodels.Pagination) ([]*descriptors.Comment, error) {
	filters := []datamodels.Filter{
		{FieldName: "entity_name", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{entityName}}}},
		{FieldName: "record_id", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{recordId}}}},
		{FieldName: "parent", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{nil}}}},
		{FieldName: "deleted", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{false}}}},
	}

	recs, _, err := s.dao.List(ctx, CommentNamespace, filters, pagination, []datamodels.Sort{{Field: "id", Order: datamodels.SortOrderDesc}})
	if err != nil {
		return nil, err
	}

	var results []*descriptors.Comment
	for _, r := range recs {
		var comment descriptors.Comment
		data, _ := json.Marshal(r.Rec)
		json.Unmarshal(data, &comment)
		results = append(results, &comment)
	}
	return results, nil
}

func (s *CommentService) Single(ctx context.Context, id string) (*descriptors.Comment, error) {
	rec, err := s.dao.Get(ctx, CommentNamespace, id)
	if err != nil || rec == nil {
		return nil, fmt.Errorf("comment not found")
	}

	var comment descriptors.Comment
	data, _ := json.Marshal(rec.Rec)
	json.Unmarshal(data, &comment)
	return &comment, nil
}

func (s *CommentService) Save(ctx context.Context, comment *descriptors.Comment) (*descriptors.Comment, error) {
	now := time.Now()
	if comment.Id == "" {
		t := time.Now()
		entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
		id := ulid.MustNew(ulid.Timestamp(t), entropy)
		comment.Id = fmt.Sprintf("%s_%d_%s", comment.EntityName, comment.RecordId, id.String())
		comment.CreatedAt = now
	}
	comment.UpdatedAt = now

	rec := datamodels.RecJSON{
		Namespace: CommentNamespace,
		Key:       comment.Id,
		Rec:       comment,
		Tmstamp:   now,
	}

	if err := s.dao.Save(ctx, rec); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *CommentService) Delete(ctx context.Context, userId, id string) error {
	comment, err := s.Single(ctx, id)
	if err != nil {
		return err
	}
	if comment.CreatedBy != userId {
		return fmt.Errorf("access denied")
	}
	comment.Deleted = true
	_, err = s.Save(ctx, comment)
	return err
}
