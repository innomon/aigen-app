package filestore

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3FileStore struct {
	client *s3.Client
	bucket string
	region string
}

func NewS3FileStore(ctx context.Context, bucket, region string) (*S3FileStore, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	return &S3FileStore{
		client: client,
		bucket: bucket,
		region: region,
	}, nil
}

func (s *S3FileStore) Upload(ctx context.Context, path string, reader io.Reader) error {
	uploader := manager.NewUploader(s.client)
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   reader,
	})
	return err
}

func (s *S3FileStore) UploadLocal(ctx context.Context, localPath, destPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return s.Upload(ctx, destPath, file)
}

func (s *S3FileStore) GetMetadata(ctx context.Context, path string) (*FileMetadata, error) {
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, err
	}

	return &FileMetadata{
		Size:        *head.ContentLength,
		ContentType: *head.ContentType,
	}, nil
}

func (s *S3FileStore) GetUrl(path string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, path)
}

func (s *S3FileStore) Download(ctx context.Context, path string, writer io.Writer) error {
	downloader := manager.NewDownloader(s.client)
	_, err := downloader.Download(ctx, fakeWriterAt{writer}, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	return err
}

func (s *S3FileStore) DownloadToLocal(ctx context.Context, path, localPath string) error {
	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return s.Download(ctx, path, file)
}

func (s *S3FileStore) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	return err
}

func (s *S3FileStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	// Need to list and delete
	return fmt.Errorf("DeleteByPrefix not implemented for S3")
}

func (s *S3FileStore) GetUploadedChunks(ctx context.Context, path string) ([]string, error) {
	return nil, fmt.Errorf("Chunked upload not implemented for S3")
}

func (s *S3FileStore) UploadChunk(ctx context.Context, path string, chunkNumber int, reader io.Reader) (string, error) {
	return "", fmt.Errorf("Chunked upload not implemented for S3")
}

func (s *S3FileStore) CommitChunks(ctx context.Context, path string) error {
	return fmt.Errorf("Chunked upload not implemented for S3")
}

type fakeWriterAt struct {
	w io.Writer
}

func (fw fakeWriterAt) WriteAt(p []byte, off int64) (n int, err error) {
	// This is a simplified version, real S3 downloader needs proper WriteAt
	return fw.w.Write(p)
}
