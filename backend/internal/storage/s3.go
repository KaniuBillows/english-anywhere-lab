package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config holds configuration for S3-compatible object storage.
type S3Config struct {
	Endpoint       string
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	ForcePathStyle bool
	PublicURL      string // optional: if set, GetURL returns publicURL/key instead of presigned
}

// S3Store stores objects in S3-compatible storage (AWS S3, Cloudflare R2, MinIO).
type S3Store struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

// NewS3Store creates a new S3Store.
func NewS3Store(cfg S3Config) (*S3Store, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.ForcePathStyle
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})

	return &S3Store{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: strings.TrimRight(cfg.PublicURL, "/"),
	}, nil
}

func (s *S3Store) Put(ctx context.Context, req PutRequest) (PutResult, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(req.ObjectKey),
		Body:        req.Reader,
		ContentType: aws.String(req.ContentType),
	}
	if req.SizeBytes > 0 {
		input.ContentLength = aws.Int64(req.SizeBytes)
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return PutResult{}, fmt.Errorf("s3 put object: %w", err)
	}
	return PutResult{ObjectKey: req.ObjectKey, SizeBytes: req.SizeBytes}, nil
}

func (s *S3Store) GetURL(ctx context.Context, objectKey string) (string, error) {
	if s.publicURL != "" {
		return s.publicURL + "/" + objectKey, nil
	}
	presigner := s3.NewPresignClient(s.client)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return "", fmt.Errorf("presign get object: %w", err)
	}
	return req.URL, nil
}

func (s *S3Store) Delete(ctx context.Context, objectKey string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("s3 delete object: %w", err)
	}
	return nil
}

func (s *S3Store) Stat(ctx context.Context, objectKey string) (*ObjectMeta, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, err
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return &ObjectMeta{
		Key:         objectKey,
		SizeBytes:   size,
		ContentType: ct,
	}, nil
}
