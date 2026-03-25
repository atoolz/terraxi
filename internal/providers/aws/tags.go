package aws

import (
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ec2TagsToMap converts EC2-style tags to a string map.
func ec2TagsToMap(tags []ec2types.Tag) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		if t.Key != nil && t.Value != nil {
			m[*t.Key] = *t.Value
		}
	}
	return m
}

// nameFromEC2Tags extracts the "Name" tag value from EC2 tags.
func nameFromEC2Tags(tags []ec2types.Tag) string {
	for _, t := range tags {
		if t.Key != nil && *t.Key == "Name" && t.Value != nil {
			return *t.Value
		}
	}
	return ""
}

// s3TagsToMap converts S3-style tags to a string map.
func s3TagsToMap(tags []s3types.Tag) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		if t.Key != nil && t.Value != nil {
			m[*t.Key] = *t.Value
		}
	}
	return m
}

// iamTagsToMap converts IAM-style tags to a string map.
func iamTagsToMap(tags []iamtypes.Tag) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		if t.Key != nil && t.Value != nil {
			m[*t.Key] = *t.Value
		}
	}
	return m
}

// rdsTagsToMap converts RDS-style tags to a string map.
func rdsTagsToMap(tags []rdstypes.Tag) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		if t.Key != nil && t.Value != nil {
			m[*t.Key] = *t.Value
		}
	}
	return m
}

// nameFromMap extracts the "Name" key from a string map.
func nameFromMap(tags map[string]string) string {
	return tags["Name"]
}
