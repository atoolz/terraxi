package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/atoolz/terraxi/pkg/types"
)

type mockEC2 struct {
	describeInstancesFn          func(ctx context.Context, input *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	describeSecurityGroupsFn     func(ctx context.Context, input *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	describeVolumesFn            func(ctx context.Context, input *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error)
	describeAddressesFn          func(ctx context.Context, input *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
	describeKeyPairsFn           func(ctx context.Context, input *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error)
	describeVpcsFn               func(ctx context.Context, input *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	describeSubnetsFn            func(ctx context.Context, input *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	describeRouteTablesFn        func(ctx context.Context, input *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	describeNatGatewaysFn        func(ctx context.Context, input *ec2.DescribeNatGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error)
	describeInternetGatewaysFn   func(ctx context.Context, input *ec2.DescribeInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error)
	describeSecurityGroupRulesFn func(ctx context.Context, input *ec2.DescribeSecurityGroupRulesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupRulesOutput, error)
}

func (m *mockEC2) DescribeInstances(ctx context.Context, input *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m.describeInstancesFn(ctx, input, optFns...)
}
func (m *mockEC2) DescribeSecurityGroups(ctx context.Context, input *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	return m.describeSecurityGroupsFn(ctx, input, optFns...)
}
func (m *mockEC2) DescribeVolumes(ctx context.Context, input *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	return m.describeVolumesFn(ctx, input, optFns...)
}
func (m *mockEC2) DescribeAddresses(ctx context.Context, input *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	return m.describeAddressesFn(ctx, input, optFns...)
}
func (m *mockEC2) DescribeKeyPairs(ctx context.Context, input *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error) {
	return m.describeKeyPairsFn(ctx, input, optFns...)
}
func (m *mockEC2) DescribeVpcs(ctx context.Context, input *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return m.describeVpcsFn(ctx, input, optFns...)
}
func (m *mockEC2) DescribeSubnets(ctx context.Context, input *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return m.describeSubnetsFn(ctx, input, optFns...)
}
func (m *mockEC2) DescribeRouteTables(ctx context.Context, input *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	return m.describeRouteTablesFn(ctx, input, optFns...)
}
func (m *mockEC2) DescribeNatGateways(ctx context.Context, input *ec2.DescribeNatGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error) {
	return m.describeNatGatewaysFn(ctx, input, optFns...)
}
func (m *mockEC2) DescribeInternetGateways(ctx context.Context, input *ec2.DescribeInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error) {
	return m.describeInternetGatewaysFn(ctx, input, optFns...)
}
func (m *mockEC2) DescribeSecurityGroupRules(ctx context.Context, input *ec2.DescribeSecurityGroupRulesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupRulesOutput, error) {
	return m.describeSecurityGroupRulesFn(ctx, input, optFns...)
}

func TestDiscoverEC2Instances(t *testing.T) {
	p := NewWithClients("us-east-1", WithEC2(&mockEC2{
		describeInstancesFn: func(_ context.Context, _ *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
			return &ec2.DescribeInstancesOutput{
				Reservations: []ec2types.Reservation{{
					Instances: []ec2types.Instance{{
						InstanceId: ptr("i-abc123"),
						SubnetId:   ptr("subnet-111"),
						VpcId:      ptr("vpc-222"),
						SecurityGroups: []ec2types.GroupIdentifier{
							{GroupId: ptr("sg-333")},
						},
						Tags: []ec2types.Tag{
							{Key: ptr("Name"), Value: ptr("web-server")},
						},
					}},
				}},
			}, nil
		},
	}))

	resources, err := discoverEC2Instances(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(resources))
	}

	r := resources[0]
	if r.ID != "i-abc123" {
		t.Errorf("expected ID i-abc123, got %s", r.ID)
	}
	if r.Name != "web-server" {
		t.Errorf("expected Name web-server, got %s", r.Name)
	}
	if r.Region != "us-east-1" {
		t.Errorf("expected Region us-east-1, got %s", r.Region)
	}
	if len(r.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies (subnet, vpc, sg), got %d", len(r.Dependencies))
	}
}

func TestDiscoverSecurityGroups_WithVPCDependency(t *testing.T) {
	p := NewWithClients("us-east-1", WithEC2(&mockEC2{
		describeSecurityGroupsFn: func(_ context.Context, _ *ec2.DescribeSecurityGroupsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
			return &ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []ec2types.SecurityGroup{{
					GroupId: ptr("sg-111"),
					VpcId:   ptr("vpc-222"),
					Tags:    []ec2types.Tag{{Key: ptr("Name"), Value: ptr("web-sg")}},
				}},
			}, nil
		},
	}))

	resources, err := discoverSecurityGroups(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 sg, got %d", len(resources))
	}
	if len(resources[0].Dependencies) != 1 || resources[0].Dependencies[0].Type != "aws_vpc" {
		t.Errorf("expected VPC dependency, got %v", resources[0].Dependencies)
	}
}
