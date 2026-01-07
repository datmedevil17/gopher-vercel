package services

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"deployment-platform/internal/config"
)

type S3Service struct {
	client *s3.S3
	bucket string
}

func NewS3Service(cfg *config.Config) *S3Service {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("auto"),
		Credentials: credentials.NewStaticCredentials(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Endpoint:    aws.String(cfg.S3Endpoint),
	}))

	return &S3Service{
		client: s3.New(sess),
		bucket: cfg.S3Bucket,
	}
}

func (s *S3Service) UploadFile(filePath, key string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = s.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   file,
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
	return s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
}
