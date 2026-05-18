package filestore

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3FileStore struct {
	client *s3.Client
	bucket string
	region string
}

func NewS3FileStore(ctx context.Context, bucket, region, accessKey, secretKey, endpoint string) (*S3FileStore, error) {
	var opts []func(*config.LoadOptions) error
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	if accessKey != "" && secretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

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
		CreatedAt:   *head.LastModified,
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

func (s *S3FileStore) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	return err
}

func (s *S3FileStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}

		var objects []types.ObjectIdentifier
		for _, obj := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{Key: obj.Key})
		}

		if len(objects) > 0 {
			_, err = s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(s.bucket),
				Delete: &types.Delete{Objects: objects},
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *S3FileStore) List(ctx context.Context, prefix string) ([]string, error) {
	var files []string
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, obj := range page.Contents {
			files = append(files, *obj.Key)
		}
	}

	return files, nil
}

func (s *S3FileStore) PurgeExpired(ctx context.Context, prefix string, ttlSeconds int) (int, error) {
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	now := time.Now()
	expiryDuration := time.Duration(ttlSeconds) * time.Second

	totalDeleted := 0
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return totalDeleted, err
		}

		var batch []types.ObjectIdentifier
		for _, obj := range page.Contents {
			if now.Sub(*obj.LastModified) > expiryDuration {
				batch = append(batch, types.ObjectIdentifier{Key: obj.Key})
			}
		}

		if len(batch) > 0 {
			_, err = s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(s.bucket),
				Delete: &types.Delete{Objects: batch},
			})
			if err != nil {
				return totalDeleted, err
			}
			totalDeleted += len(batch)
		}
	}

	return totalDeleted, nil
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
