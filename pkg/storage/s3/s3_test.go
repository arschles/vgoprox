package s3

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gomods/athens/internal/testutil"
	"github.com/gomods/athens/internal/testutil/testconfig"
	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/storage/compliance"
)

func TestBackend(t *testing.T) {
	backend := getStorage(t)
	compliance.RunTests(t, backend, backend.clear)
}

func BenchmarkBackend(b *testing.B) {
	backend := getStorage(b)
	compliance.RunBenchmarks(b, backend, backend.clear)
}

func (s *Storage) clear() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	objects, err := s.s3API.ListObjectsWithContext(ctx, &s3.ListObjectsInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		return err
	}

	for _, o := range objects.Contents {
		delParams := &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    o.Key,
		}

		_, err := s.s3API.DeleteObjectWithContext(ctx, delParams)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) createBucket() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	if _, err := s.s3API.CreateBucketWithContext(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.bucket)}); err != nil {
		aerr, ok := err.(awserr.Error)
		if !ok {
			return err
		}

		switch aerr.Code() {
		case s3.ErrCodeBucketAlreadyOwnedByYou:
			return nil
		case s3.ErrCodeBucketAlreadyExists:
			return nil
		default:
			return aerr
		}
	}

	return s.s3API.WaitUntilBucketExistsWithContext(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
}

func getStorage(t testing.TB) *Storage {
	t.Helper()
	testutil.CheckTestDependencies(t, testutil.TestDependencyMinio)
	url := testconfig.LoadTestConfig(t).Storage.Minio.Endpoint

	options := func(conf *aws.Config) {
		conf.Endpoint = aws.String(url)
		conf.DisableSSL = aws.Bool(true)
	}
	backend, err := New(
		&config.S3Config{
			Key:    "minio",
			Secret: "minio123",
			Bucket: "gomodsaws",
			Region: "us-west-1",
			ForcePathStyle: true,
		},
		config.GetTimeoutDuration(300),
		options,
	)

	if err != nil {
		t.Fatal(err)
	}

	if err = backend.createBucket(); err != nil {
		t.Fatal(err)
	}

	return backend
}
