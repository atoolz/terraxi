package discovery

import (
	"testing"

	"github.com/ahlert/terraxi/pkg/types"
)

func TestParseFilter_Empty(t *testing.T) {
	f, err := ParseFilter("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.Services) != 0 || len(f.Types) != 0 || len(f.Tags) != 0 {
		t.Errorf("expected empty filter, got: %+v", f)
	}
}

func TestParseFilter_SingleTag(t *testing.T) {
	f, err := ParseFilter("tags.env=production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Tags["env"] != "production" {
		t.Errorf("expected tags.env=production, got: %+v", f.Tags)
	}
}

func TestParseFilter_MultipleExpressions(t *testing.T) {
	f, err := ParseFilter("service=ec2 AND tags.team=platform AND exclude=aws_key_pair")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.Services) != 1 || f.Services[0] != "ec2" {
		t.Errorf("expected service=ec2, got: %v", f.Services)
	}
	if f.Tags["team"] != "platform" {
		t.Errorf("expected tags.team=platform, got: %v", f.Tags)
	}
	if len(f.Exclude) != 1 || f.Exclude[0] != "aws_key_pair" {
		t.Errorf("expected exclude=aws_key_pair, got: %v", f.Exclude)
	}
}

func TestParseFilter_InvalidExpression(t *testing.T) {
	_, err := ParseFilter("invalid")
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
}

func TestParseFilter_UnknownKey(t *testing.T) {
	_, err := ParseFilter("foobar=value")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestMatchesTags_AllMatch(t *testing.T) {
	r := types.Resource{Tags: map[string]string{"env": "prod", "team": "infra"}}
	if !MatchesTags(r, map[string]string{"env": "prod"}) {
		t.Error("expected match")
	}
}

func TestMatchesTags_NoMatch(t *testing.T) {
	r := types.Resource{Tags: map[string]string{"env": "staging"}}
	if MatchesTags(r, map[string]string{"env": "prod"}) {
		t.Error("expected no match")
	}
}

func TestMatchesTags_EmptyFilter(t *testing.T) {
	r := types.Resource{Tags: map[string]string{"env": "prod"}}
	if !MatchesTags(r, map[string]string{}) {
		t.Error("empty filter should match everything")
	}
}
