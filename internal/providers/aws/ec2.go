package aws

import (
	"context"
	"fmt"
	"log/slog"

	awsutil "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/atoolz/terraxi/internal/discovery"
	"github.com/atoolz/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_instance", discoverEC2Instances)
	RegisterDiscoverer("aws_security_group", discoverSecurityGroups)
	RegisterDiscoverer("aws_security_group_rule", discoverSecurityGroupRules)
	RegisterDiscoverer("aws_ebs_volume", discoverEBSVolumes)
	RegisterDiscoverer("aws_eip", discoverElasticIPs)
	RegisterDiscoverer("aws_key_pair", discoverKeyPairs)
}

func discoverEC2Instances(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		out, err := p.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{NextToken: nextToken})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for ec2:DescribeInstances: %w", err)
			}
			return nil, fmt.Errorf("describing EC2 instances: %w", err)
		}

		for _, reservation := range out.Reservations {
			for _, inst := range reservation.Instances {
				tags := ec2TagsToMap(inst.Tags)
				if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
					continue
				}

				id := awsutil.ToString(inst.InstanceId)
				r := types.Resource{
					Type:   "aws_instance",
					ID:     id,
					Name:   nameFromEC2Tags(inst.Tags),
					Region: p.region,
					Tags:   tags,
				}

				if inst.SubnetId != nil {
					r.Dependencies = append(r.Dependencies, types.ResourceRef{
						Type: "aws_subnet", ID: *inst.SubnetId,
					})
				}
				if inst.VpcId != nil {
					r.Dependencies = append(r.Dependencies, types.ResourceRef{
						Type: "aws_vpc", ID: *inst.VpcId,
					})
				}
				for _, sg := range inst.SecurityGroups {
					r.Dependencies = append(r.Dependencies, types.ResourceRef{
						Type: "aws_security_group", ID: awsutil.ToString(sg.GroupId),
					})
				}

				resources = append(resources, r)
			}
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	slog.Debug("EC2 instances discovery complete", "count", len(resources))
	return resources, nil
}

func discoverSecurityGroups(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		out, err := p.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{NextToken: nextToken})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for ec2:DescribeSecurityGroups: %w", err)
			}
			return nil, fmt.Errorf("describing security groups: %w", err)
		}

		for _, sg := range out.SecurityGroups {
			tags := ec2TagsToMap(sg.Tags)
			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			r := types.Resource{
				Type:   "aws_security_group",
				ID:     awsutil.ToString(sg.GroupId),
				Name:   nameFromEC2Tags(sg.Tags),
				Region: p.region,
				Tags:   tags,
			}
			if sg.VpcId != nil {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_vpc", ID: *sg.VpcId,
				})
			}
			resources = append(resources, r)
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	slog.Debug("Security groups discovery complete", "count", len(resources))
	return resources, nil
}

func discoverEBSVolumes(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		out, err := p.ec2.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{NextToken: nextToken})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for ec2:DescribeVolumes: %w", err)
			}
			return nil, fmt.Errorf("describing EBS volumes: %w", err)
		}

		for _, vol := range out.Volumes {
			tags := ec2TagsToMap(vol.Tags)
			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			resources = append(resources, types.Resource{
				Type:   "aws_ebs_volume",
				ID:     awsutil.ToString(vol.VolumeId),
				Name:   nameFromEC2Tags(vol.Tags),
				Region: p.region,
				Tags:   tags,
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	slog.Debug("EBS volumes discovery complete", "count", len(resources))
	return resources, nil
}

func discoverElasticIPs(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	out, err := p.ec2.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		if isAccessDenied(err) {
			return nil, fmt.Errorf("insufficient permissions for ec2:DescribeAddresses: %w", err)
		}
		return nil, fmt.Errorf("describing Elastic IPs: %w", err)
	}

	var resources []types.Resource
	for _, addr := range out.Addresses {
		// Classic (non-VPC) EIPs have no AllocationId and cannot be imported
		id := awsutil.ToString(addr.AllocationId)
		if id == "" {
			continue
		}

		tags := ec2TagsToMap(addr.Tags)
		if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
			continue
		}

		r := types.Resource{
			Type:   "aws_eip",
			ID:     id,
			Name:   nameFromEC2Tags(addr.Tags),
			Region: p.region,
			Tags:   tags,
		}
		if addr.InstanceId != nil {
			r.Dependencies = append(r.Dependencies, types.ResourceRef{
				Type: "aws_instance", ID: *addr.InstanceId,
			})
		}
		resources = append(resources, r)
	}

	slog.Debug("Elastic IPs discovery complete", "count", len(resources))
	return resources, nil
}

func discoverKeyPairs(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	out, err := p.ec2.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{})
	if err != nil {
		if isAccessDenied(err) {
			return nil, fmt.Errorf("insufficient permissions for ec2:DescribeKeyPairs: %w", err)
		}
		return nil, fmt.Errorf("describing key pairs: %w", err)
	}

	var resources []types.Resource
	for _, kp := range out.KeyPairs {
		tags := ec2TagsToMap(kp.Tags)
		if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
			continue
		}

		resources = append(resources, types.Resource{
			Type:   "aws_key_pair",
			ID:     awsutil.ToString(kp.KeyPairId),
			Name:   awsutil.ToString(kp.KeyName),
			Region: p.region,
			Tags:   tags,
		})
	}

	slog.Debug("Key pairs discovery complete", "count", len(resources))
	return resources, nil
}

func discoverSecurityGroupRules(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		out, err := p.ec2.DescribeSecurityGroupRules(ctx, &ec2.DescribeSecurityGroupRulesInput{NextToken: nextToken})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for ec2:DescribeSecurityGroupRules: %w", err)
			}
			return nil, fmt.Errorf("describing security group rules: %w", err)
		}

		for _, rule := range out.SecurityGroupRules {
			id := awsutil.ToString(rule.SecurityGroupRuleId)
			sgID := awsutil.ToString(rule.GroupId)

			tags := ec2TagsToMap(rule.Tags)
			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			direction := "ingress"
			if rule.IsEgress != nil && *rule.IsEgress {
				direction = "egress"
			}

			r := types.Resource{
				Type:   "aws_security_group_rule",
				ID:     id,
				Name:   fmt.Sprintf("%s-%s-%s", sgID, direction, id),
				Region: p.region,
				Tags:   tags,
				Dependencies: []types.ResourceRef{
					{Type: "aws_security_group", ID: sgID},
				},
			}

			resources = append(resources, r)
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	slog.Debug("Security group rules discovery complete", "count", len(resources))
	return resources, nil
}
