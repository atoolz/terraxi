package aws

import (
	"testing"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func ptr(s string) *string { return &s }

func TestEC2TagsToMap(t *testing.T) {
	tags := []ec2types.Tag{
		{Key: ptr("Name"), Value: ptr("web-server")},
		{Key: ptr("env"), Value: ptr("production")},
	}
	m := ec2TagsToMap(tags)
	if m["Name"] != "web-server" {
		t.Errorf("expected Name=web-server, got %q", m["Name"])
	}
	if m["env"] != "production" {
		t.Errorf("expected env=production, got %q", m["env"])
	}
}

func TestEC2TagsToMap_Empty(t *testing.T) {
	m := ec2TagsToMap(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestEC2TagsToMap_NilKeyValue(t *testing.T) {
	tags := []ec2types.Tag{
		{Key: nil, Value: ptr("val")},
		{Key: ptr("key"), Value: nil},
		{Key: ptr("good"), Value: ptr("ok")},
	}
	m := ec2TagsToMap(tags)
	if len(m) != 1 || m["good"] != "ok" {
		t.Errorf("expected only good=ok, got %v", m)
	}
}

func TestNameFromEC2Tags(t *testing.T) {
	tags := []ec2types.Tag{
		{Key: ptr("env"), Value: ptr("prod")},
		{Key: ptr("Name"), Value: ptr("my-instance")},
	}
	name := nameFromEC2Tags(tags)
	if name != "my-instance" {
		t.Errorf("expected my-instance, got %q", name)
	}
}

func TestNameFromEC2Tags_Missing(t *testing.T) {
	tags := []ec2types.Tag{
		{Key: ptr("env"), Value: ptr("prod")},
	}
	if nameFromEC2Tags(tags) != "" {
		t.Error("expected empty string when Name tag is missing")
	}
}

func TestS3TagsToMap(t *testing.T) {
	tags := []s3types.Tag{
		{Key: ptr("env"), Value: ptr("staging")},
	}
	m := s3TagsToMap(tags)
	if m["env"] != "staging" {
		t.Errorf("expected env=staging, got %q", m["env"])
	}
}

func TestNameFromMap(t *testing.T) {
	m := map[string]string{"Name": "test", "env": "prod"}
	if nameFromMap(m) != "test" {
		t.Errorf("expected test, got %q", nameFromMap(m))
	}
	if nameFromMap(map[string]string{}) != "" {
		t.Error("expected empty string for empty map")
	}
}
