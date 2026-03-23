package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/innomon/aigen-cms/utils/datamodels"
	"github.com/mitchellh/mapstructure"
	"github.com/oklog/ulid/v2"
)

type CommentService struct {
	dao relationdbdao.IPrimaryDao
}

func NewCommentService(dao relationdbdao.IPrimaryDao) *CommentService {
	return &CommentService{dao: dao}
}

func (s *CommentService) List(ctx context.Context, entityName string, recordId int64, pagination datamodels.Pagination) ([]*descriptors.Comment, error) {
	sb := s.dao.GetBuilder().Select("*").From(descriptors.CommentTableName).
		Where(squirrel.Eq{"entity_name": entityName, "record_id": recordId, "parent": nil, "deleted": false}).
		OrderBy("id DESC")

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*descriptors.Comment
	for rows.Next() {
		comment, err := s.scanComment(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, comment)
	}

	return results, nil
}

func (s *CommentService) Single(ctx context.Context, id string) (*descriptors.Comment, error) {
	query, args, err := s.dao.GetBuilder().Select("*").From(descriptors.CommentTableName).
		Where(squirrel.Eq{"id": id, "deleted": false}).Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("comment not found")
	}

	return s.scanComment(rows)
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

	data := datamodels.Record{
		"id":          comment.Id,
		"entity_name": comment.EntityName,
		"record_id":   comment.RecordId,
		"created_by":  comment.CreatedBy,
		"content":     comment.Content,
		"parent":      comment.Parent,
		"mention":     comment.Mention,
		"updated_at":  comment.UpdatedAt,
	}
	if comment.CreatedAt.IsZero() {
		data["created_at"] = now
	} else {
		data["created_at"] = comment.CreatedAt
	}

	keyFields := []string{"id"}
	_, err := s.dao.UpdateOnConflict(ctx, descriptors.CommentTableName, data, keyFields)
	if err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *CommentService) Delete(ctx context.Context, userId, id string) error {
	query, args, err := s.dao.GetBuilder().Update(descriptors.CommentTableName).
		Set("deleted", true).
		Where(squirrel.Eq{"id": id, "created_by": userId}).ToSql()
	if err != nil {
		return err
	}

	_, err = s.dao.GetDb().ExecContext(ctx, query, args...)
	return err
}

func (s *CommentService) scanComment(scanner interface {
	Scan(dest ...interface{}) error
	Columns() ([]string, error)
}) (*descriptors.Comment, error) {
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

	var comment descriptors.Comment
	if err := mapstructure.Decode(record, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}
