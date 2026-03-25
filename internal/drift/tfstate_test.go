package drift

import (
	"testing"
)

func TestParseState_ValidV4(t *testing.T) {
	data := []byte(`{
  "version": 4,
  "resources": [
    {
      "mode": "managed",
      "type": "aws_vpc",
      "name": "main",
      "instances": [
        {
          "attributes": {
            "id": "vpc-abc123",
            "cidr_block": "10.0.0.0/16"
          }
        }
      ]
    },
    {
      "mode": "data",
      "type": "aws_ami",
      "name": "ubuntu",
      "instances": [
        {
          "attributes": { "id": "ami-12345" }
        }
      ]
    }
  ]
}`)

	resources, err := ParseState(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only managed resources (data sources skipped)
	if len(resources) != 1 {
		t.Fatalf("expected 1 managed resource, got %d", len(resources))
	}
	if resources[0].Type != "aws_vpc" || resources[0].ID != "vpc-abc123" {
		t.Errorf("unexpected resource: %+v", resources[0])
	}
	if resources[0].Address != "aws_vpc.main" {
		t.Errorf("expected address aws_vpc.main, got %s", resources[0].Address)
	}
}

func TestParseState_UnsupportedVersion(t *testing.T) {
	data := []byte(`{"version": 3, "resources": []}`)
	_, err := ParseState(data)
	if err == nil {
		t.Fatal("expected error for version 3")
	}
}

func TestParseState_EmptyState(t *testing.T) {
	data := []byte(`{"version": 4, "resources": []}`)
	resources, err := ParseState(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(resources))
	}
}

func TestParseState_CountInstances(t *testing.T) {
	data := []byte(`{
  "version": 4,
  "resources": [
    {
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "instances": [
        {
          "index_key": 0,
          "attributes": { "id": "i-aaa111" }
        },
        {
          "index_key": 1,
          "attributes": { "id": "i-bbb222" }
        }
      ]
    }
  ]
}`)

	resources, err := ParseState(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(resources))
	}
	if resources[0].Address != "aws_instance.web[0]" {
		t.Errorf("expected aws_instance.web[0], got %s", resources[0].Address)
	}
	if resources[1].Address != "aws_instance.web[1]" {
		t.Errorf("expected aws_instance.web[1], got %s", resources[1].Address)
	}
}

func TestParseState_ForEachInstances(t *testing.T) {
	data := []byte(`{
  "version": 4,
  "resources": [
    {
      "mode": "managed",
      "type": "aws_s3_bucket",
      "name": "data",
      "instances": [
        {
          "index_key": "logs",
          "attributes": { "id": "my-logs-bucket" }
        },
        {
          "index_key": "assets",
          "attributes": { "id": "my-assets-bucket" }
        }
      ]
    }
  ]
}`)

	resources, err := ParseState(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(resources))
	}
	if resources[0].Address != "aws_s3_bucket.data[logs]" {
		t.Errorf("expected aws_s3_bucket.data[logs], got %s", resources[0].Address)
	}
	if resources[1].Address != "aws_s3_bucket.data[assets]" {
		t.Errorf("expected aws_s3_bucket.data[assets], got %s", resources[1].Address)
	}
}

func TestStateIndex(t *testing.T) {
	resources := []StateResource{
		{Type: "aws_vpc", Name: "main", ID: "vpc-abc123"},
		{Type: "aws_s3_bucket", Name: "data", ID: "my-bucket"},
	}
	idx := StateIndex(resources)
	if _, ok := idx["aws_vpc:vpc-abc123"]; !ok {
		t.Error("expected vpc in index")
	}
	if _, ok := idx["aws_s3_bucket:my-bucket"]; !ok {
		t.Error("expected bucket in index")
	}
	if _, ok := idx["aws_ec2:nonexistent"]; ok {
		t.Error("should not find nonexistent")
	}
}
