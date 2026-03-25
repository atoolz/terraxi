package drift

import (
	"fmt"

	"github.com/atoolz/terraxi/pkg/types"
)

// DriftReport contains the results of a drift analysis.
type DriftReport struct {
	// Unmanaged are resources that exist in the cloud but not in Terraform state.
	Unmanaged []types.Resource `json:"unmanaged"`

	// Deleted are resources that exist in state but were not found in the cloud.
	Deleted []StateResource `json:"deleted"`

	// Managed are resources that exist in both cloud and state.
	Managed []types.Resource `json:"managed"`
}

// Analyze compares discovered cloud resources against Terraform state
// and produces a drift report.
func Analyze(discovered []types.Resource, stateResources []StateResource) *DriftReport {
	stateIdx := StateIndex(stateResources)

	// Track which state resources were matched
	matched := make(map[string]bool, len(stateResources))

	report := &DriftReport{}

	for _, r := range discovered {
		key := r.Type + ":" + r.ID
		if _, ok := stateIdx[key]; ok {
			report.Managed = append(report.Managed, r)
			matched[key] = true
		} else {
			report.Unmanaged = append(report.Unmanaged, r)
		}
	}

	// Find deleted resources (in state but not in cloud)
	for _, sr := range stateResources {
		key := sr.Type + ":" + sr.ID
		if !matched[key] {
			report.Deleted = append(report.Deleted, sr)
		}
	}

	return report
}

// HasDrift returns true if there are unmanaged or deleted resources.
func (r *DriftReport) HasDrift() bool {
	return len(r.Unmanaged) > 0 || len(r.Deleted) > 0
}

// Summary returns a human-readable summary line.
func (r *DriftReport) Summary() string {
	return formatSummary(len(r.Managed), len(r.Unmanaged), len(r.Deleted))
}

func formatSummary(managed, unmanaged, deleted int) string {
	return formatCount(managed, "managed") + ", " +
		formatCount(unmanaged, "unmanaged") + ", " +
		formatCount(deleted, "deleted")
}

func formatCount(n int, label string) string {
	return fmt.Sprintf("%d %s", n, label)
}
