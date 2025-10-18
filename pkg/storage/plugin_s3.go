package storage

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Provider реализует хранилище Amazon S3 и S3-совместимые сервисы
type S3Provider struct {
	client   *s3.Client
	bucket   string
	prefix   string
	endpoint string
}

// NewS3Provider создает провайдер для S3 хранилища
// Поддерживает AWS S3 и S3-совместимые сервисы
// Формат URI:
//
//	AWS S3: s3://access_key:secret_key@bucket-name/prefix
//	Кастомные: s3://access_key:secret_key@endpoint.com/bucket/prefix
func NewS3Provider(s3URI string) (*S3Provider, error) {
	parsed, err := url.Parse(s3URI)
	if err != nil {
		return nil, fmt.Errorf("invalid S3 URI: %v", err)
	}

	// Извлекаем credentials из URI
	var accessKey, secretKey string
	if parsed.User != nil {
		accessKey = parsed.User.Username()
		secretKey, _ = parsed.User.Password()
	}

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("S3 URI must include credentials: s3://access_key:secret_key@endpoint/bucket/prefix")
	}

	var bucket, prefix, endpoint string
	host := parsed.Host
	path := strings.TrimPrefix(parsed.Path, "/")

	// Определяем тип URI:
	// Если host содержит точки, то это endpoint (например storage.yandexcloud.net)
	// Иначе это bucket name (AWS стиль)
	if strings.Contains(host, ".") {
		// Кастомный endpoint: s3://key:secret@storage.yandexcloud.net/bucket/prefix
		endpoint = host
		pathParts := strings.SplitN(path, "/", 2)
		if len(pathParts) == 0 || pathParts[0] == "" {
			return nil, fmt.Errorf("S3 URI with custom endpoint must include bucket name in path")
		}
		bucket = pathParts[0]
		if len(pathParts) > 1 {
			prefix = pathParts[1]
		}
	} else {
		// AWS S3 стиль: s3://key:secret@bucket-name/prefix
		bucket = host
		prefix = path
	}

	// Создаем конфигурацию с credentials из URI
	configOptions := []func(*config.LoadOptions) error{
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     accessKey,
				SecretAccessKey: secretKey,
			}, nil
		})),
		config.WithRegion("us-east-1"), // дефолтный регион
	}

	// Для кастомных endpoints добавляем endpoint resolver
	if endpoint != "" {
		configOptions = append(configOptions, config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               "https://" + endpoint,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			},
		)))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), configOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %v", err)
	}

	client := s3.NewFromConfig(cfg)

	return &S3Provider{
		client:   client,
		bucket:   bucket,
		prefix:   prefix,
		endpoint: endpoint,
	}, nil
}

// EnsureExists проверяет доступность S3 бакета
func (sp *S3Provider) EnsureExists() error {
	_, err := sp.client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(sp.bucket),
	})
	if err != nil {
		return fmt.Errorf("S3 bucket %s is not accessible: %v", sp.bucket, err)
	}
	return nil
}

// GenerateKey генерирует ключ для нового объекта S3
func (sp *S3Provider) GenerateKey(prefix string) string {
	timestamp := time.Now().UTC().Format("2006-01-02_15-04")
	filename := fmt.Sprintf("%s_%s.tar.gz", prefix, timestamp)
	if sp.prefix != "" {
		return sp.prefix + "/" + filename
	}
	return filename
}

// Upload загружает данные в S3
func (sp *S3Provider) Upload(key string, data []byte) error {
	_, err := sp.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(sp.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %v", err)
	}

	// Формируем URL для логов
	var logURL string
	if sp.endpoint != "" {
		logURL = fmt.Sprintf("s3://%s/%s/%s", sp.endpoint, sp.bucket, key)
	} else {
		logURL = fmt.Sprintf("s3://%s/%s", sp.bucket, key)
	}
	log.Printf("Successfully uploaded backup to S3: %s", logURL)
	return nil
}

// Delete удаляет объект из S3
func (sp *S3Provider) Delete(key string) error {
	_, err := sp.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(sp.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %v", err)
	}
	return nil
}

// List возвращает список объектов S3 с заданным префиксом
func (sp *S3Provider) List(prefix string) ([]ObjectInfo, error) {
	searchPrefix := prefix
	if sp.prefix != "" {
		searchPrefix = sp.prefix + "/" + prefix
	}

	result, err := sp.client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(sp.bucket),
		Prefix: aws.String(searchPrefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %v", err)
	}

	var objects []ObjectInfo
	for _, obj := range result.Contents {
		if obj.Key != nil && obj.Size != nil && obj.LastModified != nil {
			objects = append(objects, ObjectInfo{
				Key:          *obj.Key,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			})
		}
	}
	return objects, nil
}
