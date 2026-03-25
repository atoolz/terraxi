package hclutil

import (
	"testing"
)

func TestParseFile_ValidHCL(t *testing.T) {
	src := []byte(`resource "aws_s3_bucket" "test" {
  bucket = "my-bucket"
}
`)
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := f.Body().Blocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Type() != "resource" {
		t.Errorf("expected resource block, got %s", blocks[0].Type())
	}
	labels := blocks[0].Labels()
	if len(labels) != 2 || labels[0] != "aws_s3_bucket" || labels[1] != "test" {
		t.Errorf("unexpected labels: %v", labels)
	}
}

func TestParseFile_InvalidHCL(t *testing.T) {
	src := []byte(`this is not valid { HCL {{{{`)
	_, err := ParseFile(src)
	if err == nil {
		t.Fatal("expected error for invalid HCL")
	}
}

func TestFormatFile(t *testing.T) {
	src := []byte(`resource "aws_s3_bucket" "test" {
bucket="my-bucket"
}
`)
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	formatted := FormatFile(f)
	if len(formatted) == 0 {
		t.Error("expected non-empty formatted output")
	}
}
