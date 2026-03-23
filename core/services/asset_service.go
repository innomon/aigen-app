package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/disintegration/imaging"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/infrastructure/filestore"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/innomon/aigen-cms/utils/datamodels"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

const UploadSessionTableName = "__uploadSessions"

type AssetService struct {
	dao       relationdbdao.IPrimaryDao
	fileStore filestore.IFileStore
	settings  *descriptors.SystemSettings
}

func NewAssetService(dao relationdbdao.IPrimaryDao, fileStore filestore.IFileStore, settings *descriptors.SystemSettings) *AssetService {
	return &AssetService{
		dao:       dao,
		fileStore: fileStore,
		settings:  settings,
	}
}

func (s *AssetService) ChunkStatus(ctx context.Context, userId, fileName string, fileSize int64) (*datamodels.ChunkStatus, error) {
	query, args, err := s.dao.GetBuilder().Select("path").From(UploadSessionTableName).
		Where(squirrel.Eq{"user_id": userId, "file_name": fileName, "file_size": fileSize}).Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	var path string
	err = s.dao.GetDb().QueryRowContext(ctx, query, args...).Scan(&path)
	if err != nil {
		// If not found, create new session
		now := time.Now()
		id, _ := gonanoid.New(12)
		ext := filepath.Ext(fileName)
		path = fmt.Sprintf("%s/%s%s", now.Format("2006-01"), id, ext)

		insertQuery, insertArgs, _ := s.dao.GetBuilder().Insert(UploadSessionTableName).
			Columns("user_id", "file_name", "file_size", "path").
			Values(userId, fileName, fileSize, path).ToSql()
		_, err = s.dao.GetDb().ExecContext(ctx, insertQuery, insertArgs...)
		if err != nil {
			return nil, err
		}
		return &datamodels.ChunkStatus{Path: path, ChunkCount: 0}, nil
	}

	chunks, err := s.fileStore.GetUploadedChunks(ctx, path)
	if err != nil {
		return nil, err
	}

	return &datamodels.ChunkStatus{Path: path, ChunkCount: len(chunks)}, nil
}

func (s *AssetService) UploadChunk(ctx context.Context, path string, chunkNumber int, reader io.Reader) error {
	_, err := s.fileStore.UploadChunk(ctx, path, chunkNumber, reader)
	return err
}

func (s *AssetService) CommitChunks(ctx context.Context, path, fileName string) (*descriptors.Asset, error) {
	err := s.fileStore.CommitChunks(ctx, path)
	if err != nil {
		return nil, err
	}

	// Get metadata for the committed file
	meta, err := s.fileStore.GetMetadata(ctx, path)
	if err != nil {
		return nil, err
	}

	asset := &descriptors.Asset{
		Path: path,
		Name: fileName,
		Size: meta.Size,
		Type: meta.ContentType,
		Url:  path,
	}

	savedAsset, err := s.Save(ctx, asset)
	if err != nil {
		return nil, err
	}

	// Delete session
	delQuery, delArgs, _ := s.dao.GetBuilder().Delete(UploadSessionTableName).Where(squirrel.Eq{"path": path}).ToSql()
	s.dao.GetDb().ExecContext(ctx, delQuery, delArgs...)

	return savedAsset, nil
}

func (s *AssetService) ProcessImage(reader io.Reader) (io.Reader, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	// Resize if necessary
	if img.Bounds().Dx() > s.settings.ImageCompression.MaxWidth {
		img = imaging.Resize(img, s.settings.ImageCompression.MaxWidth, 0, imaging.Lanczos)
	}

	// Compress
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, img, &jpeg.Options{Quality: s.settings.ImageCompression.Quality})
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (s *AssetService) Upload(ctx context.Context, path string, reader io.Reader) error {
	ext := filepath.Ext(path)
	contentType := mime.TypeByExtension(ext)
	if strings.HasPrefix(contentType, "image/") && ext != ".gif" && ext != ".svg" {
		processed, err := s.ProcessImage(reader)
		if err == nil {
			reader = processed
		}
	}
	return s.fileStore.Upload(ctx, path, reader)
}

func (s *AssetService) Save(ctx context.Context, asset *descriptors.Asset) (*descriptors.Asset, error) {
	metadataJSON, _ := json.Marshal(asset.Metadata)
	now := time.Now()
	asset.CreatedAt = now
	asset.UpdatedAt = now

	sb := s.dao.GetBuilder().Insert(descriptors.AssetTableName).
		Columns("path", "url", "name", "title", "size", "type", "metadata", "created_at", "updated_at", "created_by").
		Values(asset.Path, asset.Url, asset.Name, asset.Title, asset.Size, asset.Type, string(metadataJSON), asset.CreatedAt, asset.UpdatedAt, asset.CreatedBy)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	var newId int64
	// Fix: avoid comparing s.dao.GetBuilder().PlaceholderFormat(squirrel.Dollar) == squirrel.Dollar
	// Just use a simpler check or rely on the SQL string returned by squirrel.
	if strings.Contains(query, "$1") {
		// Postgres
		err = s.dao.GetDb().QueryRowContext(ctx, query+" RETURNING id", args...).Scan(&newId)
	} else {
		// SQLite
		res, err := s.dao.GetDb().ExecContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		newId, err = res.LastInsertId()
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	asset.Id = newId
	return asset, nil
}

func (s *AssetService) GetAssetByPath(ctx context.Context, path string) (*descriptors.Asset, error) {
	query, args, err := s.dao.GetBuilder().Select("*").From(descriptors.AssetTableName).
		Where(squirrel.Eq{"path": path}).Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	row := s.dao.GetDb().QueryRowContext(ctx, query, args...)
	// Reuse scanning logic (can be improved with a helper)
	var id int64
	var p, url, name, title, assetType, metadataStr, createdBy string
	var size int64
	var createdAt, updatedAt time.Time

	if err := row.Scan(&id, &p, &url, &name, &title, &size, &assetType, &metadataStr, &createdAt, &updatedAt, &createdBy); err != nil {
		return nil, err
	}

	var metadata map[string]interface{}
	json.Unmarshal([]byte(metadataStr), &metadata)

	return &descriptors.Asset{
		Id:        id,
		Path:      p,
		Url:       url,
		Name:      name,
		Title:     title,
		Size:      size,
		Type:      assetType,
		Metadata:  metadata,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		CreatedBy: createdBy,
	}, nil
}

func (s *AssetService) UpdateAssetsLinks(ctx context.Context, oldAssetIds []int64, newAssetPaths []string, entityName string, recordId int64) error {
	// 1. Get IDs for new paths
	var newAssetIds []int64
	for _, path := range newAssetPaths {
		asset, err := s.GetAssetByPath(ctx, path)
		if err == nil && asset != nil {
			newAssetIds = append(newAssetIds, asset.Id)
		}
	}

	// 2. Diff
	oldSet := make(map[int64]bool)
	for _, id := range oldAssetIds {
		oldSet[id] = true
	}

	newSet := make(map[int64]bool)
	for _, id := range newAssetIds {
		newSet[id] = true
	}

	var toAdd []int64
	for id := range newSet {
		if !oldSet[id] {
			toAdd = append(toAdd, id)
		}
	}

	var toDel []int64
	for id := range oldSet {
		if !newSet[id] {
			toDel = append(toDel, id)
		}
	}

	tx, err := s.dao.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 3. Delete old links
	if len(toDel) > 0 {
		query, args, _ := s.dao.GetBuilder().Delete(descriptors.AssetLinkTableName).
			Where(squirrel.Eq{"entity_name": entityName, "record_id": recordId, "asset_id": toDel}).ToSql()
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}

	// 4. Add new links
	for _, id := range toAdd {
		query, args, _ := s.dao.GetBuilder().Insert(descriptors.AssetLinkTableName).
			Columns("entity_name", "record_id", "asset_id", "created_at", "updated_at").
			Values(entityName, recordId, id, time.Now(), time.Now()).ToSql()
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}
