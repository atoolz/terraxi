package aws

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"

	"github.com/atoolz/terraxi/pkg/types"
)

type mockRDS struct {
	describeDBInstancesFn       func(ctx context.Context, input *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
	describeDBClustersFn        func(ctx context.Context, input *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error)
	describeDBSubnetGroupsFn    func(ctx context.Context, input *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error)
	describeDBParameterGroupsFn func(ctx context.Context, input *rds.DescribeDBParameterGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBParameterGroupsOutput, error)
	listTagsForResourceFn       func(ctx context.Context, input *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error)
}

func (m *mockRDS) DescribeDBInstances(ctx context.Context, input *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	return m.describeDBInstancesFn(ctx, input, optFns...)
}
func (m *mockRDS) DescribeDBClusters(ctx context.Context, input *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error) {
	if m.describeDBClustersFn != nil {
		return m.describeDBClustersFn(ctx, input, optFns...)
	}
	return &rds.DescribeDBClustersOutput{}, nil
}
func (m *mockRDS) DescribeDBSubnetGroups(ctx context.Context, input *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error) {
	return m.describeDBSubnetGroupsFn(ctx, input, optFns...)
}
func (m *mockRDS) DescribeDBParameterGroups(ctx context.Context, input *rds.DescribeDBParameterGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBParameterGroupsOutput, error) {
	return m.describeDBParameterGroupsFn(ctx, input, optFns...)
}
func (m *mockRDS) ListTagsForResource(ctx context.Context, input *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error) {
	return m.listTagsForResourceFn(ctx, input, optFns...)
}

func TestDiscoverRDSInstances_WithDependencies(t *testing.T) {
	p := NewWithClients("us-east-1", WithRDS(&mockRDS{
		describeDBInstancesFn: func(_ context.Context, _ *rds.DescribeDBInstancesInput, _ ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
			return &rds.DescribeDBInstancesOutput{
				DBInstances: []rdstypes.DBInstance{{
					DBInstanceIdentifier: ptr("my-db"),
					DBInstanceArn:        ptr("arn:aws:rds:us-east-1:123456:db:my-db"),
					DBSubnetGroup:        &rdstypes.DBSubnetGroup{DBSubnetGroupName: ptr("my-subnet-group")},
					VpcSecurityGroups: []rdstypes.VpcSecurityGroupMembership{
						{VpcSecurityGroupId: ptr("sg-111")},
					},
				}},
			}, nil
		},
		listTagsForResourceFn: func(_ context.Context, _ *rds.ListTagsForResourceInput, _ ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error) {
			return &rds.ListTagsForResourceOutput{
				TagList: []rdstypes.Tag{{Key: ptr("env"), Value: ptr("prod")}},
			}, nil
		},
	}))

	resources, err := discoverRDSInstances(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 RDS instance, got %d", len(resources))
	}
	r := resources[0]
	if r.ID != "my-db" {
		t.Errorf("expected ID my-db, got %s", r.ID)
	}
	if r.Tags["env"] != "prod" {
		t.Errorf("expected env=prod tag, got %v", r.Tags)
	}
	// Should have subnet group + security group deps
	if len(r.Dependencies) != 2 {
		t.Errorf("expected 2 deps, got %d: %v", len(r.Dependencies), r.Dependencies)
	}
}

func TestDiscoverDBParameterGroups_SkipsDefaults(t *testing.T) {
	p := NewWithClients("us-east-1", WithRDS(&mockRDS{
		describeDBParameterGroupsFn: func(_ context.Context, _ *rds.DescribeDBParameterGroupsInput, _ ...func(*rds.Options)) (*rds.DescribeDBParameterGroupsOutput, error) {
			return &rds.DescribeDBParameterGroupsOutput{
				DBParameterGroups: []rdstypes.DBParameterGroup{
					{DBParameterGroupName: ptr("default.mysql8.0")},
					{DBParameterGroupName: ptr("my-custom-params")},
					{DBParameterGroupName: ptr("default")},
				},
			}, nil
		},
	}))

	resources, err := discoverDBParameterGroups(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 custom param group (defaults skipped), got %d", len(resources))
	}
	if resources[0].ID != "my-custom-params" {
		t.Errorf("expected my-custom-params, got %s", resources[0].ID)
	}
}

func TestDiscoverDBParameterGroups_SkipsDefaultPrefix(t *testing.T) {
	p := NewWithClients("us-east-1", WithRDS(&mockRDS{
		describeDBParameterGroupsFn: func(_ context.Context, _ *rds.DescribeDBParameterGroupsInput, _ ...func(*rds.Options)) (*rds.DescribeDBParameterGroupsOutput, error) {
			return &rds.DescribeDBParameterGroupsOutput{
				DBParameterGroups: []rdstypes.DBParameterGroup{
					{DBParameterGroupName: ptr("d")}, // short name, should not panic
				},
			}, nil
		},
	}))

	resources, err := discoverDBParameterGroups(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	// "d" is not "default" and doesn't start with "default.", should be included
	if len(resources) != 1 {
		t.Fatalf("expected 1, got %d", len(resources))
	}
	_ = strings.HasPrefix // suppress unused import if needed
}
