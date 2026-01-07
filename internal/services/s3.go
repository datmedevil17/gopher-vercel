package services

import (
	"context"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"deployment-platform/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Service struct {
	client *s3.Client
	bucket string
}

func NewS3Service(cfg *config.Config) *S3Service {
	creds := credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")

	s3Config := aws.Config{
		Region:      "auto",
		Credentials: creds,
		EndpointResolver: aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           cfg.S3Endpoint,
				SigningRegion: "auto",
			}, nil
		}),
	}

	return &S3Service{
		client: s3.NewFromConfig(s3Config),
		bucket: cfg.S3Bucket,
	}
}

func (s *S3Service) UploadFile(filePath, key string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})

	return err
}

func (s *S3Service) UploadDirectory(dirPath, prefix string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		key := filepath.Join(prefix, relPath)
		key = strings.ReplaceAll(key, "\\", "/")

		return s.UploadFile(path, key)
	})
}

func (s *S3Service) GetObject(key string) (*s3.GetObjectOutput, error) {
	return s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
}
