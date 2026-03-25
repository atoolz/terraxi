package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/ahlert/terraxi/pkg/types"
)

type mockS3 struct {
	listBucketsFn       func(ctx context.Context, input *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	getBucketLocationFn func(ctx context.Context, input *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
	getBucketTaggingFn  func(ctx context.Context, input *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error)
}

func (m *mockS3) ListBuckets(ctx context.Context, input *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	return m.listBucketsFn(ctx, input, optFns...)
}
func (m *mockS3) GetBucketLocation(ctx context.Context, input *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
	return m.getBucketLocationFn(ctx, input, optFns...)
}
func (m *mockS3) GetBucketTagging(ctx context.Context, input *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
	return m.getBucketTaggingFn(ctx, input, optFns...)
}

func TestDiscoverS3Buckets_FiltersByRegion(t *testing.T) {
	p := NewWithClients("us-east-1", WithS3(&mockS3{
		listBucketsFn: func(_ context.Context, _ *s3.ListBucketsInput, _ ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
			return &s3.ListBucketsOutput{
				Buckets: []s3types.Bucket{
					{Name: ptr("bucket-east")},
					{Name: ptr("bucket-west")},
				},
			}, nil
		},
		getBucketLocationFn: func(_ context.Context, input *s3.GetBucketLocationInput, _ ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
			if *input.Bucket == "bucket-east" {
				return &s3.GetBucketLocationOutput{LocationConstraint: ""}, nil // us-east-1
			}
			return &s3.GetBucketLocationOutput{LocationConstraint: "eu-west-1"}, nil
		},
		getBucketTaggingFn: func(_ context.Context, _ *s3.GetBucketTaggingInput, _ ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
			return &s3.GetBucketTaggingOutput{}, nil
		},
	}))

	resources, err := discoverS3Buckets(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(resources))
	}
	if resources[0].ID != "bucket-east" {
		t.Errorf("expected bucket-east, got %s", resources[0].ID)
	}
	if resources[0].Region != "us-east-1" {
		t.Errorf("expected region us-east-1, got %s", resources[0].Region)
	}
}

func TestDiscoverS3Buckets_FiltersByTags(t *testing.T) {
	p := NewWithClients("us-east-1", WithS3(&mockS3{
		listBucketsFn: func(_ context.Context, _ *s3.ListBucketsInput, _ ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
			return &s3.ListBucketsOutput{
				Buckets: []s3types.Bucket{{Name: ptr("my-bucket")}},
			}, nil
		},
		getBucketLocationFn: func(_ context.Context, _ *s3.GetBucketLocationInput, _ ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
			return &s3.GetBucketLocationOutput{LocationConstraint: ""}, nil
		},
		getBucketTaggingFn: func(_ context.Context, _ *s3.GetBucketTaggingInput, _ ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
			return &s3.GetBucketTaggingOutput{
				TagSet: []s3types.Tag{{Key: ptr("env"), Value: ptr("staging")}},
			}, nil
		},
	}))

	// Filter for env=production should exclude the staging bucket
	filter := types.Filter{Tags: map[string]string{"env": "production"}}
	resources, err := discoverS3Buckets(context.Background(), p, filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 0 {
		t.Errorf("expected 0 resources (tag mismatch), got %d", len(resources))
	}
}
