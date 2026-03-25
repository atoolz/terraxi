package codegen

import (
	"testing"

	"github.com/atoolz/terraxi/pkg/types"
)

func TestIDIndex_Lookup(t *testing.T) {
	resources := []types.Resource{
		{Type: "aws_vpc", ID: "vpc-abc123", Name: "main"},
		{Type: "aws_subnet", ID: "subnet-111", Name: "public-a"},
		{Type: "aws_security_group", ID: "sg-222", Name: "web"},
	}

	names := NewNameResolver()
	idx := NewIDIndex(resources, names)

	ref, ok := idx.Lookup("vpc-abc123")
	if !ok || ref != "aws_vpc.main.id" {
		t.Errorf("expected aws_vpc.main.id, got %q (ok=%v)", ref, ok)
	}

	ref, ok = idx.Lookup("subnet-111")
	if !ok || ref != "aws_subnet.public_a.id" {
		t.Errorf("expected aws_subnet.public_a.id, got %q (ok=%v)", ref, ok)
	}

	ref, ok = idx.Lookup("sg-222")
	if !ok || ref != "aws_security_group.web.id" {
		t.Errorf("expected aws_security_group.web.id, got %q (ok=%v)", ref, ok)
	}

	_, ok = idx.Lookup("nonexistent")
	if ok {
		t.Error("expected false for nonexistent ID")
	}
}

func TestIDIndex_ConsistentWithNameResolver(t *testing.T) {
	resources := []types.Resource{
		{Type: "aws_s3_bucket", ID: "bucket-1", Name: "data"},
		{Type: "aws_s3_bucket", ID: "bucket-2", Name: "data"},
	}

	// Simulate what Generator does: resolve names for import blocks
	genNames := NewNameResolver()
	name1 := genNames.Resolve(resources[0])
	name2 := genNames.Resolve(resources[1])

	// Simulate what PostProcessor does: build ID index
	ppNames := NewNameResolver()
	idx := NewIDIndex(resources, ppNames)

	// The names must match
	addr1, ok := idx.LookupAddress("bucket-1")
	if !ok || addr1 != "aws_s3_bucket."+name1 {
		t.Errorf("ID index address mismatch: got %q, expected aws_s3_bucket.%s", addr1, name1)
	}

	addr2, ok := idx.LookupAddress("bucket-2")
	if !ok || addr2 != "aws_s3_bucket."+name2 {
		t.Errorf("ID index address mismatch: got %q, expected aws_s3_bucket.%s", addr2, name2)
	}
}

func TestIDIndex_Len(t *testing.T) {
	resources := []types.Resource{
		{Type: "aws_vpc", ID: "vpc-1", Name: "main"},
	}
	idx := NewIDIndex(resources, NewNameResolver())
	if idx.Len() != 1 {
		t.Errorf("expected len 1, got %d", idx.Len())
	}
}
