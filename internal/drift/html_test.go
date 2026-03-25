package drift

import (
	"strings"
	"testing"

	"github.com/atoolz/terraxi/pkg/types"
)

func TestRenderHTML(t *testing.T) {
	report := &DriftReport{
		Managed: []types.Resource{
			{Type: "aws_vpc", ID: "vpc-1", Name: "main"},
		},
		Unmanaged: []types.Resource{
			{Type: "aws_s3_bucket", ID: "orphan-bucket", Name: "orphan"},
		},
		Deleted: []StateResource{
			{Type: "aws_iam_role", Address: "aws_iam_role.old", ID: "old-role"},
		},
	}

	html, err := RenderHTML(report)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	result := string(html)

	if !strings.Contains(result, "Terraxi Drift Report") {
		t.Error("expected title in HTML")
	}
	if !strings.Contains(result, "orphan-bucket") {
		t.Error("expected unmanaged resource in HTML")
	}
	if !strings.Contains(result, "old-role") {
		t.Error("expected deleted resource in HTML")
	}
	if !strings.Contains(result, "aws_iam_role.old") {
		t.Error("expected deleted address in HTML")
	}
}

func TestRenderHTML_NoDrift(t *testing.T) {
	report := &DriftReport{
		Managed: []types.Resource{{Type: "aws_vpc", ID: "vpc-1"}},
	}

	html, err := RenderHTML(report)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	result := string(html)
	if !strings.Contains(result, "No unmanaged resources found") {
		t.Error("expected 'no unmanaged' message")
	}
	if !strings.Contains(result, "No deleted resources found") {
		t.Error("expected 'no deleted' message")
	}
}
