//go:build integration

package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestValidateGeneratedHCL is an integration test that requires terraform or tofu
// to be installed. Run with: go test -tags integration ./internal/codegen/
func TestValidateGeneratedHCL(t *testing.T) {
	// Check if terraform is available
	tfBinary := "terraform"
	if _, err := exec.LookPath(tfBinary); err != nil {
		tfBinary = "tofu"
		if _, err := exec.LookPath(tfBinary); err != nil {
			t.Skip("neither terraform nor tofu found in PATH, skipping integration test")
		}
	}

	tmpDir := t.TempDir()

	// Write a minimal providers.tf
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

	// Write a generated.tf that mimics post-processed output
	generatedContent := []byte(`resource "aws_s3_bucket" "data" {
  bucket = "my-data-bucket"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "public" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
}
`)
	if err := os.WriteFile(filepath.Join(tmpDir, "generated.tf"), generatedContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Write variables.tf
	varsContent := []byte(`variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}
`)
	if err := os.WriteFile(filepath.Join(tmpDir, "variables.tf"), varsContent, 0644); err != nil {
		t.Fatal(err)
	}

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

	t.Logf("%s validate passed", tfBinary)
}
