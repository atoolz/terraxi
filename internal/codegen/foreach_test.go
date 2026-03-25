package codegen

import (
	"strings"
	"testing"

	"github.com/atoolz/terraxi/internal/codegen/hclutil"
)

func TestCollapseForEach_ThreeIdenticalBuckets(t *testing.T) {
	hcl := []byte(`resource "aws_s3_bucket" "bucket_a" {
  bucket = "data-a"
  acl    = "private"
}

resource "aws_s3_bucket" "bucket_b" {
  bucket = "data-b"
  acl    = "private"
}

resource "aws_s3_bucket" "bucket_c" {
  bucket = "data-c"
  acl    = "private"
}
`)

	f, err := hclutil.ParseFile(hcl)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	CollapseForEach(f)

	result := string(hclutil.FormatFile(f))

	// Should have collapsed into one block with for_each
	if !strings.Contains(result, "for_each") {
		t.Error("expected for_each in collapsed output")
	}

	// Should reference each.value.bucket
	if !strings.Contains(result, "each.value.bucket") {
		t.Error("expected each.value.bucket reference")
	}

	// Should only have one resource block (the collapsed one), not three separate ones
	blockCount := strings.Count(result, `resource "aws_s3_bucket"`)
	if blockCount != 1 {
		t.Errorf("expected 1 collapsed resource block, got %d\n%s", blockCount, result)
	}

	// acl should be constant (not each.value.acl)
	if strings.Contains(result, "each.value.acl") {
		t.Error("acl is constant across all buckets, should not use each.value")
	}
}

func TestCollapseForEach_LessThanThree(t *testing.T) {
	hcl := []byte(`resource "aws_s3_bucket" "a" {
  bucket = "a"
}

resource "aws_s3_bucket" "b" {
  bucket = "b"
}
`)

	f, err := hclutil.ParseFile(hcl)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	CollapseForEach(f)

	result := string(hclutil.FormatFile(f))

	// Should NOT collapse (only 2 resources)
	if strings.Contains(result, "for_each") {
		t.Error("should not collapse groups smaller than 3")
	}
}

func TestCollapseForEach_SkipsNestedBlocks(t *testing.T) {
	hcl := []byte(`resource "aws_security_group" "sg_a" {
  name = "a"

  ingress {
    from_port = 80
  }
}

resource "aws_security_group" "sg_b" {
  name = "b"

  ingress {
    from_port = 443
  }
}

resource "aws_security_group" "sg_c" {
  name = "c"

  ingress {
    from_port = 22
  }
}
`)

	f, err := hclutil.ParseFile(hcl)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	CollapseForEach(f)

	result := string(hclutil.FormatFile(f))

	// Should NOT collapse (has nested blocks)
	if strings.Contains(result, "for_each") {
		t.Error("should not collapse resources with nested blocks")
	}
}

func TestCollapseForEach_DifferentSchemas(t *testing.T) {
	hcl := []byte(`resource "aws_s3_bucket" "a" {
  bucket = "a"
  acl    = "private"
}

resource "aws_s3_bucket" "b" {
  bucket = "b"
}

resource "aws_s3_bucket" "c" {
  bucket = "c"
}
`)

	f, err := hclutil.ParseFile(hcl)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	CollapseForEach(f)

	result := string(hclutil.FormatFile(f))

	// "a" has a different schema (extra acl attr), so b+c group is only 2
	// Nothing should collapse
	if strings.Contains(result, "for_each") {
		t.Error("should not collapse groups with different schemas")
	}
}
