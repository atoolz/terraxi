package codegen

import (
	"testing"

	"github.com/atoolz/terraxi/internal/discovery"
	"github.com/atoolz/terraxi/internal/graph"
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

func TestUniqueName_Deduplication(t *testing.T) {
	g := NewGenerator(types.EngineTerraform, "/tmp/test", discovery.ProviderConfig{Region: "us-east-1"}, graph.New())

	r1 := types.Resource{Type: "aws_s3_bucket", ID: "my-bucket", Name: "my-bucket"}
	r2 := types.Resource{Type: "aws_s3_bucket", ID: "my.bucket", Name: "my.bucket"}
	r3 := types.Resource{Type: "aws_s3_bucket", ID: "my:bucket", Name: "my:bucket"}

	name1 := g.uniqueName(r1)
	name2 := g.uniqueName(r2)
	name3 := g.uniqueName(r3)

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

func TestGenerateImportBlock(t *testing.T) {
	g := NewGenerator(types.EngineTerraform, "/tmp/test", discovery.ProviderConfig{Region: "us-east-1"}, graph.New())

	block := g.GenerateImportBlock(types.Resource{
		Type: "aws_s3_bucket",
		ID:   "my-bucket",
		Name: "my-bucket",
	})

	expected := `import {
  to = aws_s3_bucket.my_bucket
  id = "my-bucket"
}
`
	if block != expected {
		t.Errorf("unexpected import block:\ngot:\n%s\nwant:\n%s", block, expected)
	}
}

func TestOrganizeByService_RegisteredTypes(t *testing.T) {
	RegisterServiceMapping("aws_s3_bucket", "s3")
	RegisterServiceMapping("aws_iam_role", "iam")

	pp := NewPostProcessor(graph.New())
	resources := []types.Resource{
		{Type: "aws_s3_bucket", ID: "b1"},
		{Type: "aws_s3_bucket", ID: "b2"},
		{Type: "aws_iam_role", ID: "r1"},
	}

	byService := pp.OrganizeByService(resources)
	if len(byService["s3"]) != 2 {
		t.Errorf("expected 2 s3 resources, got %d", len(byService["s3"]))
	}
	if len(byService["iam"]) != 1 {
		t.Errorf("expected 1 iam resource, got %d", len(byService["iam"]))
	}
}

func TestOrganizeByService_UnregisteredFallsToOther(t *testing.T) {
	pp := NewPostProcessor(graph.New())
	resources := []types.Resource{
		{Type: "aws_unknown_thing", ID: "u1"},
	}

	byService := pp.OrganizeByService(resources)
	if len(byService["other"]) != 1 {
		t.Errorf("expected unregistered type in 'other', got %+v", byService)
	}
}
