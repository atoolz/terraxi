package aws

import (
	"context"
	"fmt"
	"log/slog"

	awsutil "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/ahlert/terraxi/internal/discovery"
	"github.com/ahlert/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_vpc", discoverVPCs)
	RegisterDiscoverer("aws_subnet", discoverSubnets)
	RegisterDiscoverer("aws_route_table", discoverRouteTables)
	RegisterDiscoverer("aws_nat_gateway", discoverNatGateways)
	RegisterDiscoverer("aws_internet_gateway", discoverInternetGateways)
}

// DescribeVpcs does not paginate in the AWS API (returns all VPCs at once).
func discoverVPCs(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	out, err := p.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		if isAccessDenied(err) {
			return nil, fmt.Errorf("insufficient permissions for ec2:DescribeVpcs: %w", err)
		}
		return nil, fmt.Errorf("describing VPCs: %w", err)
	}

	var resources []types.Resource
	for _, vpc := range out.Vpcs {
		tags := ec2TagsToMap(vpc.Tags)
		if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
			continue
		}
		resources = append(resources, types.Resource{
			Type:   "aws_vpc",
			ID:     awsutil.ToString(vpc.VpcId),
			Name:   nameFromEC2Tags(vpc.Tags),
			Region: p.region,
			Tags:   tags,
		})
	}

	slog.Debug("VPCs discovery complete", "count", len(resources))
	return resources, nil
}

func discoverSubnets(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		out, err := p.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{NextToken: nextToken})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for ec2:DescribeSubnets: %w", err)
			}
			return nil, fmt.Errorf("describing subnets: %w", err)
		}

		for _, subnet := range out.Subnets {
			tags := ec2TagsToMap(subnet.Tags)
			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			r := types.Resource{
				Type:   "aws_subnet",
				ID:     awsutil.ToString(subnet.SubnetId),
				Name:   nameFromEC2Tags(subnet.Tags),
				Region: p.region,
				Tags:   tags,
			}
			if subnet.VpcId != nil {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_vpc", ID: *subnet.VpcId,
				})
			}
			resources = append(resources, r)
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	slog.Debug("Subnets discovery complete", "count", len(resources))
	return resources, nil
}

func discoverRouteTables(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		out, err := p.ec2.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{NextToken: nextToken})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for ec2:DescribeRouteTables: %w", err)
			}
			return nil, fmt.Errorf("describing route tables: %w", err)
		}

		for _, rt := range out.RouteTables {
			tags := ec2TagsToMap(rt.Tags)
			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			r := types.Resource{
				Type:   "aws_route_table",
				ID:     awsutil.ToString(rt.RouteTableId),
				Name:   nameFromEC2Tags(rt.Tags),
				Region: p.region,
				Tags:   tags,
			}
			if rt.VpcId != nil {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_vpc", ID: *rt.VpcId,
				})
			}
			resources = append(resources, r)
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	slog.Debug("Route tables discovery complete", "count", len(resources))
	return resources, nil
}

func discoverNatGateways(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		out, err := p.ec2.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{NextToken: nextToken})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for ec2:DescribeNatGateways: %w", err)
			}
			return nil, fmt.Errorf("describing NAT gateways: %w", err)
		}

		for _, ng := range out.NatGateways {
			tags := ec2TagsToMap(ng.Tags)
			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			r := types.Resource{
				Type:   "aws_nat_gateway",
				ID:     awsutil.ToString(ng.NatGatewayId),
				Name:   nameFromEC2Tags(ng.Tags),
				Region: p.region,
				Tags:   tags,
			}
			if ng.SubnetId != nil {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_subnet", ID: *ng.SubnetId,
				})
			}
			if ng.VpcId != nil {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_vpc", ID: *ng.VpcId,
				})
			}
			resources = append(resources, r)
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	slog.Debug("NAT gateways discovery complete", "count", len(resources))
	return resources, nil
}

// DescribeInternetGateways does not paginate in the AWS API.
func discoverInternetGateways(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	out, err := p.ec2.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{})
	if err != nil {
		if isAccessDenied(err) {
			return nil, fmt.Errorf("insufficient permissions for ec2:DescribeInternetGateways: %w", err)
		}
		return nil, fmt.Errorf("describing internet gateways: %w", err)
	}

	var resources []types.Resource
	for _, igw := range out.InternetGateways {
		tags := ec2TagsToMap(igw.Tags)
		if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
			continue
		}

		r := types.Resource{
			Type:   "aws_internet_gateway",
			ID:     awsutil.ToString(igw.InternetGatewayId),
			Name:   nameFromEC2Tags(igw.Tags),
			Region: p.region,
			Tags:   tags,
		}
		for _, att := range igw.Attachments {
			if att.VpcId != nil {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_vpc", ID: *att.VpcId,
				})
			}
		}
		resources = append(resources, r)
	}

	slog.Debug("Internet gateways discovery complete", "count", len(resources))
	return resources, nil
}
