package aws

import (
	"context"
	"fmt"
	"log/slog"

	awsutil "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"github.com/ahlert/terraxi/internal/discovery"
	"github.com/ahlert/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_ecs_cluster", discoverECSClusters)
	RegisterDiscoverer("aws_ecs_service", discoverECSServices)
	RegisterDiscoverer("aws_ecs_task_definition", discoverECSTaskDefinitions)
}

func discoverECSClusters(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var allArns []string
	var nextToken *string

	for {
		out, err := p.ecs.ListClusters(ctx, &ecs.ListClustersInput{NextToken: nextToken})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for ecs:ListClusters: %w", err)
			}
			return nil, fmt.Errorf("listing ECS clusters: %w", err)
		}
		allArns = append(allArns, out.ClusterArns...)
		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	if len(allArns) == 0 {
		return nil, nil
	}

	var resources []types.Resource

	// DescribeClusters accepts max 100 ARNs per call
	const clusterBatchSize = 100
	for i := 0; i < len(allArns); i += clusterBatchSize {
		end := i + clusterBatchSize
		if end > len(allArns) {
			end = len(allArns)
		}
		batch := allArns[i:end]

		desc, err := p.ecs.DescribeClusters(ctx, &ecs.DescribeClustersInput{Clusters: batch})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for ecs:DescribeClusters: %w", err)
			}
			return nil, fmt.Errorf("describing ECS clusters: %w", err)
		}

		for _, c := range desc.Clusters {
			tags := make(map[string]string, len(c.Tags))
			for _, t := range c.Tags {
				tags[awsutil.ToString(t.Key)] = awsutil.ToString(t.Value)
			}

			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			resources = append(resources, types.Resource{
				Type:   "aws_ecs_cluster",
				ID:     awsutil.ToString(c.ClusterArn),
				Name:   awsutil.ToString(c.ClusterName),
				Region: p.region,
				Tags:   tags,
			})
		}
	}

	slog.Debug("ECS clusters discovery complete", "count", len(resources))
	return resources, nil
}

func discoverECSServices(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	// Services require a cluster. Discover clusters first.
	clusters, err := discoverECSClusters(ctx, p, filter)
	if err != nil {
		return nil, err
	}

	var resources []types.Resource
	for _, cluster := range clusters {
		clusterArn := cluster.ID
		var serviceArns []string
		var nextToken *string

		for {
			out, err := p.ecs.ListServices(ctx, &ecs.ListServicesInput{
				Cluster:   &clusterArn,
				NextToken: nextToken,
			})
			if err != nil {
				if isAccessDenied(err) || isNotFound(err) {
					break
				}
				return nil, fmt.Errorf("listing ECS services for %s: %w", cluster.Name, err)
			}
			serviceArns = append(serviceArns, out.ServiceArns...)
			if out.NextToken == nil {
				break
			}
			nextToken = out.NextToken
		}

		if len(serviceArns) == 0 {
			continue
		}

		// DescribeServices accepts max 10 ARNs at a time
		for i := 0; i < len(serviceArns); i += 10 {
			end := i + 10
			if end > len(serviceArns) {
				end = len(serviceArns)
			}
			batch := serviceArns[i:end]

			desc, err := p.ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
				Cluster:  &clusterArn,
				Services: batch,
			})
			if err != nil {
				if isAccessDenied(err) {
					break
				}
				return nil, fmt.Errorf("describing ECS services: %w", err)
			}

			for _, svc := range desc.Services {
				tags := make(map[string]string, len(svc.Tags))
				for _, t := range svc.Tags {
					tags[awsutil.ToString(t.Key)] = awsutil.ToString(t.Value)
				}

				// Import ID for ECS service: cluster/service-name
				r := types.Resource{
					Type:   "aws_ecs_service",
					ID:     fmt.Sprintf("%s/%s", cluster.Name, awsutil.ToString(svc.ServiceName)),
					Name:   awsutil.ToString(svc.ServiceName),
					Region: p.region,
					Tags:   tags,
					Dependencies: []types.ResourceRef{
						{Type: "aws_ecs_cluster", ID: clusterArn},
					},
				}

				if svc.TaskDefinition != nil {
					r.Dependencies = append(r.Dependencies, types.ResourceRef{
						Type: "aws_ecs_task_definition", ID: *svc.TaskDefinition,
					})
				}

				resources = append(resources, r)
			}
		}
	}

	slog.Debug("ECS services discovery complete", "count", len(resources))
	return resources, nil
}

func discoverECSTaskDefinitions(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var allArns []string
	var nextToken *string

	for {
		out, err := p.ecs.ListTaskDefinitions(ctx, &ecs.ListTaskDefinitionsInput{
			Status:    ecstypes.TaskDefinitionStatusActive,
			NextToken: nextToken,
		})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for ecs:ListTaskDefinitions: %w", err)
			}
			return nil, fmt.Errorf("listing ECS task definitions: %w", err)
		}
		allArns = append(allArns, out.TaskDefinitionArns...)
		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	var resources []types.Resource
	for _, arn := range allArns {
		desc, err := p.ecs.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: &arn,
		})
		if err != nil {
			if isAccessDenied(err) || isNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("describing task definition %s: %w", arn, err)
		}

		td := desc.TaskDefinition
		if td == nil {
			continue
		}

		tags := make(map[string]string, len(desc.Tags))
		for _, t := range desc.Tags {
			tags[awsutil.ToString(t.Key)] = awsutil.ToString(t.Value)
		}

		r := types.Resource{
			Type:   "aws_ecs_task_definition",
			ID:     awsutil.ToString(td.TaskDefinitionArn),
			Name:   awsutil.ToString(td.Family),
			Region: p.region,
			Tags:   tags,
		}

		if td.ExecutionRoleArn != nil {
			r.Dependencies = append(r.Dependencies, types.ResourceRef{
				Type: "aws_iam_role", ID: roleNameFromARN(*td.ExecutionRoleArn),
			})
		}
		if td.TaskRoleArn != nil {
			r.Dependencies = append(r.Dependencies, types.ResourceRef{
				Type: "aws_iam_role", ID: roleNameFromARN(*td.TaskRoleArn),
			})
		}

		resources = append(resources, r)
	}

	slog.Debug("ECS task definitions discovery complete", "count", len(resources))
	return resources, nil
}
