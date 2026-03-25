package aws

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	awsutil "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"

	"github.com/ahlert/terraxi/internal/discovery"
	"github.com/ahlert/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_db_instance", discoverRDSInstances)
	RegisterDiscoverer("aws_db_subnet_group", discoverDBSubnetGroups)
	RegisterDiscoverer("aws_db_parameter_group", discoverDBParameterGroups)
}

func discoverRDSInstances(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.rds.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for rds:DescribeDBInstances: %w", err)
			}
			return nil, fmt.Errorf("describing RDS instances: %w", err)
		}

		for _, db := range out.DBInstances {
			// Fetch tags via ARN
			tags := map[string]string{}
			if db.DBInstanceArn != nil {
				tagOut, tagErr := p.rds.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
					ResourceName: db.DBInstanceArn,
				})
				if tagErr == nil {
					tags = rdsTagsToMap(tagOut.TagList)
				}
			}

			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			r := types.Resource{
				Type:   "aws_db_instance",
				ID:     awsutil.ToString(db.DBInstanceIdentifier),
				Name:   awsutil.ToString(db.DBInstanceIdentifier),
				Region: p.region,
				Tags:   tags,
			}
			if db.DBSubnetGroup != nil && db.DBSubnetGroup.DBSubnetGroupName != nil {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_db_subnet_group", ID: *db.DBSubnetGroup.DBSubnetGroupName,
				})
			}
			for _, sg := range db.VpcSecurityGroups {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_security_group", ID: awsutil.ToString(sg.VpcSecurityGroupId),
				})
			}
			resources = append(resources, r)
		}

		if out.Marker == nil {
			break
		}
		marker = out.Marker
	}

	slog.Debug("RDS instances discovery complete", "count", len(resources))
	return resources, nil
}

func discoverDBSubnetGroups(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.rds.DescribeDBSubnetGroups(ctx, &rds.DescribeDBSubnetGroupsInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for rds:DescribeDBSubnetGroups: %w", err)
			}
			return nil, fmt.Errorf("describing DB subnet groups: %w", err)
		}

		for _, sg := range out.DBSubnetGroups {
			r := types.Resource{
				Type:   "aws_db_subnet_group",
				ID:     awsutil.ToString(sg.DBSubnetGroupName),
				Name:   awsutil.ToString(sg.DBSubnetGroupName),
				Region: p.region,
			}
			if sg.VpcId != nil {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_vpc", ID: *sg.VpcId,
				})
			}
			for _, subnet := range sg.Subnets {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_subnet", ID: awsutil.ToString(subnet.SubnetIdentifier),
				})
			}
			resources = append(resources, r)
		}

		if out.Marker == nil {
			break
		}
		marker = out.Marker
	}

	slog.Debug("DB subnet groups discovery complete", "count", len(resources))
	return resources, nil
}

func discoverDBParameterGroups(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.rds.DescribeDBParameterGroups(ctx, &rds.DescribeDBParameterGroupsInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for rds:DescribeDBParameterGroups: %w", err)
			}
			return nil, fmt.Errorf("describing DB parameter groups: %w", err)
		}

		for _, pg := range out.DBParameterGroups {
			name := awsutil.ToString(pg.DBParameterGroupName)
			// Skip default parameter groups (not user-managed)
			if name == "default" || strings.HasPrefix(name, "default.") {
				continue
			}

			resources = append(resources, types.Resource{
				Type:   "aws_db_parameter_group",
				ID:     name,
				Name:   name,
				Region: p.region,
			})
		}

		if out.Marker == nil {
			break
		}
		marker = out.Marker
	}

	slog.Debug("DB parameter groups discovery complete", "count", len(resources))
	return resources, nil
}
