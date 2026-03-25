package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/atoolz/terraxi/pkg/types"
)

func TestDiscoverSubnets_WithVPCDependency(t *testing.T) {
	p := NewWithClients("us-east-1", WithEC2(&mockEC2{
		describeSubnetsFn: func(_ context.Context, _ *ec2.DescribeSubnetsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
			return &ec2.DescribeSubnetsOutput{
				Subnets: []ec2types.Subnet{{
					SubnetId: ptr("subnet-111"),
					VpcId:    ptr("vpc-222"),
					Tags:     []ec2types.Tag{{Key: ptr("Name"), Value: ptr("public-a")}},
				}},
			}, nil
		},
	}))

	resources, err := discoverSubnets(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 subnet, got %d", len(resources))
	}
	r := resources[0]
	if r.ID != "subnet-111" {
		t.Errorf("expected subnet-111, got %s", r.ID)
	}
	if r.Name != "public-a" {
		t.Errorf("expected name public-a, got %s", r.Name)
	}
	if len(r.Dependencies) != 1 || r.Dependencies[0].Type != "aws_vpc" || r.Dependencies[0].ID != "vpc-222" {
		t.Errorf("expected VPC dependency, got %v", r.Dependencies)
	}
}

func TestDiscoverInternetGateways_WithVPCAttachment(t *testing.T) {
	p := NewWithClients("us-east-1", WithEC2(&mockEC2{
		describeInternetGatewaysFn: func(_ context.Context, _ *ec2.DescribeInternetGatewaysInput, _ ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error) {
			return &ec2.DescribeInternetGatewaysOutput{
				InternetGateways: []ec2types.InternetGateway{{
					InternetGatewayId: ptr("igw-111"),
					Attachments: []ec2types.InternetGatewayAttachment{
						{VpcId: ptr("vpc-222")},
					},
				}},
			}, nil
		},
	}))

	resources, err := discoverInternetGateways(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 IGW, got %d", len(resources))
	}
	if len(resources[0].Dependencies) != 1 || resources[0].Dependencies[0].ID != "vpc-222" {
		t.Errorf("expected VPC dep from attachment, got %v", resources[0].Dependencies)
	}
}

func TestDiscoverNatGateways_WithSubnetAndVPCDeps(t *testing.T) {
	p := NewWithClients("us-east-1", WithEC2(&mockEC2{
		describeNatGatewaysFn: func(_ context.Context, _ *ec2.DescribeNatGatewaysInput, _ ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error) {
			return &ec2.DescribeNatGatewaysOutput{
				NatGateways: []ec2types.NatGateway{{
					NatGatewayId: ptr("nat-111"),
					SubnetId:     ptr("subnet-222"),
					VpcId:        ptr("vpc-333"),
				}},
			}, nil
		},
	}))

	resources, err := discoverNatGateways(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 NAT GW, got %d", len(resources))
	}
	if len(resources[0].Dependencies) != 2 {
		t.Errorf("expected 2 deps (subnet+vpc), got %d", len(resources[0].Dependencies))
	}
}
