package aws

import (
	"context"

	"github.com/ahlert/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_instance", discoverEC2Instances)
	RegisterDiscoverer("aws_security_group", discoverSecurityGroups)
	RegisterDiscoverer("aws_ebs_volume", discoverEBSVolumes)
	RegisterDiscoverer("aws_eip", discoverElasticIPs)
	RegisterDiscoverer("aws_key_pair", discoverKeyPairs)
}

func discoverEC2Instances(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	// TODO: Implement using aws-sdk-go-v2
	// 1. Call ec2.DescribeInstances
	// 2. Extract instance ID, tags, VPC/subnet refs
	// 3. Build dependencies (security group refs, subnet ref, key pair ref)
	// 4. Filter by tags if specified
	return nil, nil
}

func discoverSecurityGroups(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	// TODO: ec2.DescribeSecurityGroups
	return nil, nil
}

func discoverEBSVolumes(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	// TODO: ec2.DescribeVolumes
	return nil, nil
}

func discoverElasticIPs(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	// TODO: ec2.DescribeAddresses
	return nil, nil
}

func discoverKeyPairs(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	// TODO: ec2.DescribeKeyPairs
	return nil, nil
}
