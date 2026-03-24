package graph

import (
	"testing"

	"github.com/ahlert/terraxi/pkg/types"
)

func TestTopologicalSort_Simple(t *testing.T) {
	g := New()
	g.Add(types.Resource{Type: "aws_instance", ID: "i-1", Dependencies: []types.ResourceRef{
		{Type: "aws_security_group", ID: "sg-1"},
	}})
	g.Add(types.Resource{Type: "aws_security_group", ID: "sg-1"})

	sorted := g.TopologicalSort()
	if len(sorted) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(sorted))
	}
	// sg-1 should come before i-1 (dependency first)
	if sorted[0].ID != "sg-1" {
		t.Errorf("expected sg-1 first (dependency), got %s", sorted[0].ID)
	}
}

func TestTopologicalSort_CycleDoesNotPanic(t *testing.T) {
	g := New()
	// Circular dependency: sg-1 -> sg-2 -> sg-1
	g.Add(types.Resource{Type: "aws_security_group", ID: "sg-1", Dependencies: []types.ResourceRef{
		{Type: "aws_security_group", ID: "sg-2"},
	}})
	g.Add(types.Resource{Type: "aws_security_group", ID: "sg-2", Dependencies: []types.ResourceRef{
		{Type: "aws_security_group", ID: "sg-1"},
	}})

	// Should not panic or infinite loop
	sorted := g.TopologicalSort()
	if len(sorted) != 2 {
		t.Fatalf("expected 2 resources despite cycle, got %d", len(sorted))
	}
}

func TestTopologicalSort_NoDeps(t *testing.T) {
	g := New()
	g.Add(types.Resource{Type: "aws_s3_bucket", ID: "b1"})
	g.Add(types.Resource{Type: "aws_s3_bucket", ID: "b2"})
	g.Add(types.Resource{Type: "aws_s3_bucket", ID: "b3"})

	sorted := g.TopologicalSort()
	if len(sorted) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(sorted))
	}
}

func TestDependenciesOf(t *testing.T) {
	g := New()
	g.Add(types.Resource{Type: "aws_instance", ID: "i-1", Dependencies: []types.ResourceRef{
		{Type: "aws_vpc", ID: "vpc-1"},
		{Type: "aws_security_group", ID: "sg-1"},
	}})
	g.Add(types.Resource{Type: "aws_vpc", ID: "vpc-1"})
	g.Add(types.Resource{Type: "aws_security_group", ID: "sg-1"})

	deps := g.DependenciesOf(types.Resource{Type: "aws_instance", ID: "i-1"})
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}
}

func TestDependentsOf(t *testing.T) {
	g := New()
	g.Add(types.Resource{Type: "aws_instance", ID: "i-1", Dependencies: []types.ResourceRef{
		{Type: "aws_vpc", ID: "vpc-1"},
	}})
	g.Add(types.Resource{Type: "aws_instance", ID: "i-2", Dependencies: []types.ResourceRef{
		{Type: "aws_vpc", ID: "vpc-1"},
	}})
	g.Add(types.Resource{Type: "aws_vpc", ID: "vpc-1"})

	dependents := g.DependentsOf(types.Resource{Type: "aws_vpc", ID: "vpc-1"})
	if len(dependents) != 2 {
		t.Fatalf("expected 2 dependents, got %d", len(dependents))
	}
}

func TestLen(t *testing.T) {
	g := New()
	if g.Len() != 0 {
		t.Error("expected 0")
	}
	g.Add(types.Resource{Type: "aws_s3_bucket", ID: "b1"})
	if g.Len() != 1 {
		t.Error("expected 1")
	}
}
