package aws

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/atoolz/terraxi/internal/discovery"
	"github.com/atoolz/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_s3_bucket", discoverS3Buckets)
}

func discoverS3Buckets(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	out, err := p.s3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		if isAccessDenied(err) {
			return nil, fmt.Errorf("insufficient permissions for s3:ListAllMyBuckets: %w", err)
		}
		return nil, fmt.Errorf("listing S3 buckets: %w", err)
	}

	var resources []types.Resource
	for _, b := range out.Buckets {
		if b.Name == nil {
			continue
		}
		name := *b.Name

		// S3 is global: filter by region via GetBucketLocation
		loc, err := p.s3.GetBucketLocation(ctx, &s3.GetBucketLocationInput{Bucket: &name})
		if err != nil {
			if isNotFound(err) || isAccessDenied(err) {
				slog.Debug("Skipping bucket", "bucket", name, "reason", err)
				continue
			}
			return nil, fmt.Errorf("getting location for bucket %s: %w", name, err)
		}

		// AWS returns empty string for us-east-1
		bucketRegion := string(loc.LocationConstraint)
		if bucketRegion == "" {
			bucketRegion = "us-east-1"
		}
		if bucketRegion != p.region {
			continue
		}

		// Fetch tags (NoSuchTagSet is not an error)
		tags := map[string]string{}
		tagOut, err := p.s3.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: &name})
		if err != nil {
			if !isNotFound(err) && !isAccessDenied(err) {
				slog.Warn("Failed to get tags for bucket", "bucket", name, "error", err)
			}
		} else {
			tags = s3TagsToMap(tagOut.TagSet)
		}

		if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
			continue
		}

		resources = append(resources, types.Resource{
			Type:   "aws_s3_bucket",
			ID:     name,
			Name:   name,
			Region: bucketRegion,
			Tags:   tags,
		})
	}

	slog.Debug("S3 discovery complete", "count", len(resources))
	return resources, nil
}
