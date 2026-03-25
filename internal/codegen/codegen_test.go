package codegen

import (
	"testing"

	"github.com/atoolz/terraxi/internal/discovery"
	"github.com/atoolz/terraxi/internal/graph"
	"github.com/atoolz/terraxi/pkg/types"
)

// Note: sanitizeName and NameResolver tests are in names_test.go

func TestGenerateImportBlock(t *testing.T) {
	g := NewGenerator(types.EngineTerraform, "/tmp/test", discovery.ProviderConfig{Region: "us-east-1"}, graph.New(), "")

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

	pp := NewPostProcessor(graph.New(), nil)
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
	pp := NewPostProcessor(graph.New(), nil)
	resources := []types.Resource{
		{Type: "aws_unknown_thing", ID: "u1"},
	}

	byService := pp.OrganizeByService(resources)
	if len(byService["other"]) != 1 {
		t.Errorf("expected unregistered type in 'other', got %+v", byService)
	}
}
