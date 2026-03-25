package codegen

import (
	"testing"

	"github.com/atoolz/terraxi/pkg/types"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-bucket", "my_bucket"},
		{"i-abc123", "i_abc123"},
		{"123start", "r_123start"},
		{"valid_name", "valid_name"},
		{"path/to/thing", "path_to_thing"},
		{"", "resource"},
		{"arn:aws:s3:::bucket", "arn_aws_s3___bucket"},
	}

	for _, tt := range tests {
		got := sanitizeName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNameResolver_Deduplication(t *testing.T) {
	nr := NewNameResolver()

	r1 := types.Resource{Type: "aws_s3_bucket", ID: "my-bucket", Name: "my-bucket"}
	r2 := types.Resource{Type: "aws_s3_bucket", ID: "my.bucket", Name: "my.bucket"}
	r3 := types.Resource{Type: "aws_s3_bucket", ID: "my:bucket", Name: "my:bucket"}

	name1 := nr.Resolve(r1)
	name2 := nr.Resolve(r2)
	name3 := nr.Resolve(r3)

	if name1 != "my_bucket" {
		t.Errorf("first name should be 'my_bucket', got %q", name1)
	}
	if name2 != "my_bucket_2" {
		t.Errorf("second name should be 'my_bucket_2', got %q", name2)
	}
	if name3 != "my_bucket_3" {
		t.Errorf("third name should be 'my_bucket_3', got %q", name3)
	}
}

func TestNameResolver_FallbackToID(t *testing.T) {
	nr := NewNameResolver()
	r := types.Resource{Type: "aws_s3_bucket", ID: "my-bucket", Name: ""}
	name := nr.Resolve(r)
	if name != "my_bucket" {
		t.Errorf("expected fallback to ID, got %q", name)
	}
}

func TestNameResolver_Reset(t *testing.T) {
	nr := NewNameResolver()
	r := types.Resource{Type: "aws_s3_bucket", ID: "b1", Name: "b1"}
	nr.Resolve(r)
	nr.Reset()
	name := nr.Resolve(r)
	if name != "b1" {
		t.Errorf("after reset, expected 'b1' (no suffix), got %q", name)
	}
}

func TestNameResolver_Deterministic(t *testing.T) {
	resources := []types.Resource{
		{Type: "aws_vpc", ID: "vpc-1", Name: "main"},
		{Type: "aws_subnet", ID: "subnet-1", Name: "public-a"},
		{Type: "aws_subnet", ID: "subnet-2", Name: "public-b"},
	}

	// Run twice, should produce identical names
	nr1 := NewNameResolver()
	nr2 := NewNameResolver()

	for i, r := range resources {
		n1 := nr1.Resolve(r)
		n2 := nr2.Resolve(r)
		if n1 != n2 {
			t.Errorf("resource %d: names diverge: %q vs %q", i, n1, n2)
		}
	}
}
