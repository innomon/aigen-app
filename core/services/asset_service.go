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

	"github.com/disintegration/imaging"
	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/filestore"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

const (
	AssetNamespace         = "aigen.core.descriptors.Asset"
	AssetLinkNamespace     = "aigen.core.descriptors.AssetLink"
	UploadSessionNamespace = "aigen.core.services.UploadSession"
)

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
	filters := []datamodels.Filter{
		{FieldName: "userId", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{userId}}}},
		{FieldName: "fileName", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{fileName}}}},
		{FieldName: "fileSize", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{fileSize}}}},
	}
	recs, _, err := s.dao.List(ctx, UploadSessionNamespace, filters, datamodels.Pagination{}, nil)
	if err != nil || len(recs) == 0 {
		now := time.Now()
		id, _ := gonanoid.New(12)
		ext := filepath.Ext(fileName)
		path := fmt.Sprintf("%s/%s%s", now.Format("2006-01"), id, ext)

		session := datamodels.UploadSession{
			UserId:   userId,
			FileName: fileName,
			FileSize: fileSize,
			Path:     path,
		}
		s.dao.Save(ctx, datamodels.RecJSON{
			Namespace: UploadSessionNamespace,
			Key:       path,
			Rec:       session,
		})
		return &datamodels.ChunkStatus{Path: path, ChunkCount: 0}, nil
	}

	path := recs[0].Key
	chunks, _ := s.fileStore.GetUploadedChunks(ctx, path)
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

	s.dao.Delete(ctx, UploadSessionNamespace, path)
	return savedAsset, nil
}

func (s *AssetService) ProcessImage(reader io.Reader) (io.Reader, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	if img.Bounds().Dx() > s.settings.ImageCompression.MaxWidth {
		img = imaging.Resize(img, s.settings.ImageCompression.MaxWidth, 0, imaging.Lanczos)
	}

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
	now := time.Now()
	asset.CreatedAt = now
	asset.UpdatedAt = now

	if asset.Id == 0 {
		asset.Id = time.Now().UnixNano()
	}

	rec := datamodels.RecJSON{
		Namespace: AssetNamespace,
		Key:       asset.Path,
		Rec:       asset,
		Tmstamp:   now,
	}

	err := s.dao.Save(ctx, rec)
	return asset, err
}

func (s *AssetService) GetAssetByPath(ctx context.Context, path string) (*descriptors.Asset, error) {
	rec, err := s.dao.Get(ctx, AssetNamespace, path)
	if err != nil || rec == nil {
		return nil, err
	}
	// Simplified scanning logic: Rec is already interface{}, mapstructure or type assertion needed
	var asset descriptors.Asset
	data, _ := json.Marshal(rec.Rec)
	json.Unmarshal(data, &asset)
	return &asset, nil
}

func (s *AssetService) UpdateAssetsLinks(ctx context.Context, oldAssetIds []int64, newAssetPaths []string, entityName string, recordId int64) error {
	// Simplified implementation for the pivot
	for _, path := range newAssetPaths {
		link := descriptors.AssetLink{
			EntityName: entityName,
			RecordId:   recordId,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		asset, _ := s.GetAssetByPath(ctx, path)
		if asset != nil {
			link.AssetId = asset.Id
		}
		key := fmt.Sprintf("%s_%v_%s", entityName, recordId, path)
		s.dao.Save(ctx, datamodels.RecJSON{
			Namespace: AssetLinkNamespace,
			Key:       key,
			Rec:       link,
		})
	}
	return nil
}
