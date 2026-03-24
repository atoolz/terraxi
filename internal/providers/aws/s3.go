package aws

import (
	"context"

	"github.com/ahlert/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_s3_bucket", discoverS3Buckets)
}

func discoverS3Buckets(_ context.Context, _ *Provider, _ types.Filter) ([]types.Resource, error) {
	// TODO: Implement using aws-sdk-go-v2
	// 1. Call s3.ListBuckets
	// 2. For each bucket, get location to match region filter
	// 3. Get tags via s3.GetBucketTagging
	// 4. Filter by tags if specified
	// 5. Return list of Resource{Type: "aws_s3_bucket", ID: bucketName, ...}
	return nil, nil
}
