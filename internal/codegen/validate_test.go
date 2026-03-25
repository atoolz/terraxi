//go:build integration

package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/atoolz/terraxi/internal/codegen/hclutil"
	"github.com/atoolz/terraxi/internal/graph"
	"github.com/atoolz/terraxi/pkg/types"
)

// TestValidateGeneratedHCL is an integration test that requires terraform or tofu.
// Run with: go test -tags integration ./internal/codegen/ -v
func TestValidateGeneratedHCL(t *testing.T) {
	tfBinary := "terraform"
	if _, err := exec.LookPath(tfBinary); err != nil {
		tfBinary = "tofu"
		if _, err := exec.LookPath(tfBinary); err != nil {
			t.Skip("neither terraform nor tofu found in PATH, skipping integration test")
		}
	}

	tmpDir := t.TempDir()

	// Write providers.tf with mock credentials (no real AWS needed)
	providersContent := []byte(`terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  access_key                  = "mock"
  secret_key                  = "mock"
}
`)
	if err := os.WriteFile(filepath.Join(tmpDir, "providers.tf"), providersContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Simulate post-processed output by running Process on raw HCL
	resources := []types.Resource{
		{Type: "aws_vpc", ID: "vpc-abc123", Name: "main"},
		{Type: "aws_subnet", ID: "subnet-def456", Name: "public"},
	}

	rawHCL := []byte(`resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "public" {
  vpc_id     = "vpc-abc123"
  cidr_block = "10.0.1.0/24"
}

resource "aws_s3_bucket" "data" {
  bucket = "my-data-bucket"
}
`)

	names := NewNameResolver()
	idx := NewIDIndex(resources, names)
	pp := NewPostProcessor(graph.New(), idx)

	processed, err := pp.Process(rawHCL, resources)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify the processed HCL contains references
	result := string(processed)
	t.Logf("Processed HCL:\n%s", result)

	if err := os.WriteFile(filepath.Join(tmpDir, "generated.tf"), processed, 0644); err != nil {
		t.Fatal(err)
	}

	// Write variables.tf from post-processor
	varsContent := pp.ExtractVariables(nil, "us-east-1")
	if err := os.WriteFile(filepath.Join(tmpDir, "variables.tf"), varsContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Test for_each collapse output separately
	foreachHCL := []byte(`resource "aws_s3_bucket" "a" {
  bucket = "data-a"
  acl    = "private"
}

resource "aws_s3_bucket" "b" {
  bucket = "data-b"
  acl    = "private"
}

resource "aws_s3_bucket" "c" {
  bucket = "data-c"
  acl    = "private"
}
`)
	foreachFile, err := hclutil.ParseFile(foreachHCL)
	if err != nil {
		t.Fatalf("parse foreach HCL: %v", err)
	}
	CollapseForEach(foreachFile)
	collapsedBytes := hclutil.FormatFile(foreachFile)

	if err := os.WriteFile(filepath.Join(tmpDir, "collapsed.tf"), collapsedBytes, 0644); err != nil {
		t.Fatal(err)
	}

	t.Logf("Collapsed HCL:\n%s", string(collapsedBytes))

	// Run terraform init
	initCmd := exec.Command(tfBinary, "init")
	initCmd.Dir = tmpDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("%s init failed: %v\n%s", tfBinary, err, string(out))
	}

	// Run terraform validate
	validateCmd := exec.Command(tfBinary, "validate")
	validateCmd.Dir = tmpDir
	if out, err := validateCmd.CombinedOutput(); err != nil {
		t.Fatalf("%s validate failed: %v\n%s", tfBinary, err, string(out))
	}

	t.Logf("%s validate passed on post-processed + collapsed output", tfBinary)
}
