package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atoolz/terraxi/internal/codegen/hclutil"
	"github.com/atoolz/terraxi/internal/graph"
	"github.com/atoolz/terraxi/pkg/types"
)

func TestProcess_ReplacesIDs(t *testing.T) {
	resources := []types.Resource{
		{Type: "aws_vpc", ID: "vpc-abc123", Name: "main"},
		{Type: "aws_subnet", ID: "subnet-def456", Name: "public"},
	}

	names := NewNameResolver()
	idx := NewIDIndex(resources, names)
	pp := NewPostProcessor(graph.New(), idx)

	rawHCL := []byte(`resource "aws_instance" "web" {
  vpc_id    = "vpc-abc123"
  subnet_id = "subnet-def456"
  ami       = "ami-12345678"
}
`)

	processed, err := pp.Process(rawHCL, resources)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	result := string(processed)

	// vpc-abc123 should be replaced with aws_vpc.main.id
	if strings.Contains(result, "vpc-abc123") {
		t.Error("vpc-abc123 should have been replaced with a reference")
	}

	// subnet-def456 should be replaced
	if strings.Contains(result, "subnet-def456") {
		t.Error("subnet-def456 should have been replaced with a reference")
	}

	// ami-12345678 should NOT be replaced (not in the index)
	if !strings.Contains(result, "ami-12345678") {
		t.Error("ami-12345678 should remain unchanged (not a discovered resource)")
	}
}

func TestProcess_EmptyHCL(t *testing.T) {
	pp := NewPostProcessor(graph.New(), nil)
	result, err := pp.Process([]byte{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Error("expected empty output for empty input")
	}
}

func TestProcess_NilIndex(t *testing.T) {
	pp := NewPostProcessor(graph.New(), nil)
	input := []byte(`resource "aws_s3_bucket" "test" { bucket = "my-bucket" }`)
	result, err := pp.Process(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(input) {
		t.Error("with nil index, output should equal input")
	}
}

func TestExtractVariables(t *testing.T) {
	pp := NewPostProcessor(graph.New(), nil)
	vars := pp.ExtractVariables(nil, "us-east-1")
	result := string(vars)

	if !strings.Contains(result, `variable "region"`) {
		t.Error("expected region variable")
	}
	if !strings.Contains(result, `default     = "us-east-1"`) {
		t.Error("expected us-east-1 as default")
	}
}

func TestExtractVariables_EmptyRegion(t *testing.T) {
	pp := NewPostProcessor(graph.New(), nil)
	vars := pp.ExtractVariables(nil, "")
	if len(vars) != 0 {
		t.Error("expected no variables for empty region")
	}
}

func TestSplitByService(t *testing.T) {
	RegisterServiceMapping("aws_s3_bucket", "s3")
	RegisterServiceMapping("aws_iam_role", "iam")

	resources := []types.Resource{
		{Type: "aws_s3_bucket", ID: "b1", Name: "data"},
		{Type: "aws_iam_role", ID: "r1", Name: "admin"},
	}

	idx := NewIDIndex(resources, NewNameResolver())
	pp := NewPostProcessor(graph.New(), idx)

	hcl := []byte(`resource "aws_s3_bucket" "data" {
  bucket = "data-bucket"
}

resource "aws_iam_role" "admin" {
  name = "admin-role"
}
`)

	processed, err := pp.Process(hcl, resources)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Parse the processed HCL and split by service
	f, err := hclutil.ParseFile(processed)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	tmpDir := t.TempDir()
	if err := pp.SplitByService(f, tmpDir); err != nil {
		t.Fatalf("SplitByService failed: %v", err)
	}

	// Verify files were created
	s3File := filepath.Join(tmpDir, "s3", "main.tf")
	iamFile := filepath.Join(tmpDir, "iam", "main.tf")

	if _, err := os.Stat(s3File); os.IsNotExist(err) {
		t.Errorf("expected %s to exist", s3File)
	}
	if _, err := os.Stat(iamFile); os.IsNotExist(err) {
		t.Errorf("expected %s to exist", iamFile)
	}

	// Verify content
	s3Content, _ := os.ReadFile(s3File)
	if !strings.Contains(string(s3Content), "aws_s3_bucket") {
		t.Error("s3/main.tf should contain aws_s3_bucket")
	}

	iamContent, _ := os.ReadFile(iamFile)
	if !strings.Contains(string(iamContent), "aws_iam_role") {
		t.Error("iam/main.tf should contain aws_iam_role")
	}
}

func TestLooksLikeAWSID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"vpc-abc123", true},
		{"subnet-def456", true},
		{"sg-111222", true},
		{"i-abc123def", true},
		{"arn:aws:iam::123456:role/test", true},
		{"my-bucket", false},
		{"us-east-1", false},
		{"", false},
		{"hello", false},
	}

	for _, tt := range tests {
		got := looksLikeAWSID(tt.input)
		if got != tt.want {
			t.Errorf("looksLikeAWSID(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsValidServiceName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"ec2", true},
		{"s3", true},
		{"route53", true},
		{"cloud-watch", true},
		{"other", true},
		{"../../../etc", false},
		{"", false},
		{"a/b", false},
		{"a b", false},
	}
	for _, tt := range tests {
		got := isValidServiceName(tt.input)
		if got != tt.want {
			t.Errorf("isValidServiceName(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
