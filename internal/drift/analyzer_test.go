package drift

import (
	"testing"

	"github.com/atoolz/terraxi/pkg/types"
)

func TestAnalyze_AllCategories(t *testing.T) {
	discovered := []types.Resource{
		{Type: "aws_vpc", ID: "vpc-1", Name: "managed-vpc"},   // in state
		{Type: "aws_vpc", ID: "vpc-2", Name: "unmanaged-vpc"}, // NOT in state
		{Type: "aws_s3_bucket", ID: "bucket-1", Name: "data"}, // in state
	}

	stateResources := []StateResource{
		{Type: "aws_vpc", ID: "vpc-1", Name: "main", Address: "aws_vpc.main"},
		{Type: "aws_s3_bucket", ID: "bucket-1", Name: "data", Address: "aws_s3_bucket.data"},
		{Type: "aws_iam_role", ID: "deleted-role", Name: "old", Address: "aws_iam_role.old"}, // NOT in cloud
	}

	report := Analyze(discovered, stateResources)

	if len(report.Managed) != 2 {
		t.Errorf("expected 2 managed, got %d", len(report.Managed))
	}
	if len(report.Unmanaged) != 1 {
		t.Errorf("expected 1 unmanaged, got %d", len(report.Unmanaged))
	}
	if report.Unmanaged[0].ID != "vpc-2" {
		t.Errorf("expected unmanaged vpc-2, got %s", report.Unmanaged[0].ID)
	}
	if len(report.Deleted) != 1 {
		t.Errorf("expected 1 deleted, got %d", len(report.Deleted))
	}
	if report.Deleted[0].ID != "deleted-role" {
		t.Errorf("expected deleted deleted-role, got %s", report.Deleted[0].ID)
	}
}

func TestAnalyze_NoDrift(t *testing.T) {
	discovered := []types.Resource{
		{Type: "aws_vpc", ID: "vpc-1"},
	}
	stateResources := []StateResource{
		{Type: "aws_vpc", ID: "vpc-1"},
	}

	report := Analyze(discovered, stateResources)
	if report.HasDrift() {
		t.Error("expected no drift")
	}
}

func TestAnalyze_AllUnmanaged(t *testing.T) {
	discovered := []types.Resource{
		{Type: "aws_vpc", ID: "vpc-1"},
		{Type: "aws_vpc", ID: "vpc-2"},
	}

	report := Analyze(discovered, nil)
	if !report.HasDrift() {
		t.Error("expected drift (all unmanaged)")
	}
	if len(report.Unmanaged) != 2 {
		t.Errorf("expected 2 unmanaged, got %d", len(report.Unmanaged))
	}
}

func TestDriftReport_Summary(t *testing.T) {
	report := &DriftReport{
		Managed:   make([]types.Resource, 5),
		Unmanaged: make([]types.Resource, 3),
		Deleted:   make([]StateResource, 1),
	}
	summary := report.Summary()
	if summary != "5 managed, 3 unmanaged, 1 deleted" {
		t.Errorf("unexpected summary: %s", summary)
	}
}
