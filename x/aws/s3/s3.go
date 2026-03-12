package s3

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3setlock "github.com/mashiike/s3-setlock"
)

func GetS3Client() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	return s3.NewFromConfig(cfg), nil
}

func BucketPath(fragments ...string) string {
	var val string

	val = strings.TrimPrefix(strings.Join(fragments, "/"), "/")

	return val
}

func NewLock(uri string, s3Client *s3.Client) (*s3setlock.Locker, error) {
	lockPath, err := url.JoinPath(uri, ".lock")
	if err != nil {
		return nil, err
	}

	locker, err := s3setlock.New(
		lockPath,
		s3setlock.WithClient(s3Client),
		s3setlock.WithContext(context.Background()),
		s3setlock.WithDelay(true),
		s3setlock.WithLeaseDuration(30*time.Second),
	)

	return locker, err
}
