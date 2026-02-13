package s3

import (
	"fmt"

	"github.com/user/legisgraph/legal-parser/internal/domain"
)

// S3Source reads raw crawler output from an AWS S3 bucket.
// It implements domain.SourceProvider.
//
// TODO: Implement using the AWS SDK v2.
// This file exists to demonstrate that the domain is fully decoupled
// from the storage layer. Creating this adapter requires NO changes
// to internal/domain or internal/service.
type S3Source struct {
	bucket string
	prefix string
}

// NewS3Source creates a new S3Source.
func NewS3Source(bucket, prefix string) *S3Source {
	return &S3Source{bucket: bucket, prefix: prefix}
}

func (s *S3Source) ListDocuments() ([]domain.RawDocument, error) {
	return nil, fmt.Errorf("S3Source.ListDocuments: not implemented")
}

func (s *S3Source) GetContent(id string) ([]byte, error) {
	return nil, fmt.Errorf("S3Source.GetContent: not implemented")
}

func (s *S3Source) GetMetadata(id string) (*domain.CrawlerMetadata, error) {
	return nil, fmt.Errorf("S3Source.GetMetadata: not implemented")
}
