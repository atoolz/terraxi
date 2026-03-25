package aws

import (
	"context"
	"fmt"
	"log/slog"

	awsutil "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"github.com/ahlert/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_cloudwatch_log_group", discoverCloudWatchLogGroups)
	RegisterDiscoverer("aws_cloudwatch_metric_alarm", discoverCloudWatchAlarms)
}

func discoverCloudWatchLogGroups(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		out, err := p.cloudwatchlogs.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
			NextToken: nextToken,
		})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for logs:DescribeLogGroups: %w", err)
			}
			return nil, fmt.Errorf("describing CloudWatch log groups: %w", err)
		}

		for _, lg := range out.LogGroups {
			name := awsutil.ToString(lg.LogGroupName)

			// CloudWatch log groups don't have tags inline.
			// Skip tag filtering for this resource type.
			if len(filter.Tags) > 0 {
				slog.Debug("Skipping log group tag filter (not available inline)", "logGroup", name)
				continue
			}

			resources = append(resources, types.Resource{
				Type:   "aws_cloudwatch_log_group",
				ID:     name,
				Name:   name,
				Region: p.region,
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	slog.Debug("CloudWatch log groups discovery complete", "count", len(resources))
	return resources, nil
}

func discoverCloudWatchAlarms(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		out, err := p.cloudwatch.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{
			NextToken: nextToken,
		})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for cloudwatch:DescribeAlarms: %w", err)
			}
			return nil, fmt.Errorf("describing CloudWatch alarms: %w", err)
		}

		for _, alarm := range out.MetricAlarms {
			name := awsutil.ToString(alarm.AlarmName)

			// Alarms don't have tags inline via DescribeAlarms.
			if len(filter.Tags) > 0 {
				continue
			}

			r := types.Resource{
				Type:   "aws_cloudwatch_metric_alarm",
				ID:     name,
				Name:   name,
				Region: p.region,
			}

			// Wire dimension-based dependencies (best effort)
			for _, dim := range alarm.Dimensions {
				dimName := awsutil.ToString(dim.Name)
				dimValue := awsutil.ToString(dim.Value)
				switch dimName {
				case "InstanceId":
					r.Dependencies = append(r.Dependencies, types.ResourceRef{
						Type: "aws_instance", ID: dimValue,
					})
				case "LoadBalancer":
					r.Dependencies = append(r.Dependencies, types.ResourceRef{
						Type: "aws_lb", ID: dimValue,
					})
				case "DBInstanceIdentifier":
					r.Dependencies = append(r.Dependencies, types.ResourceRef{
						Type: "aws_db_instance", ID: dimValue,
					})
				}
			}

			resources = append(resources, r)
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	slog.Debug("CloudWatch alarms discovery complete", "count", len(resources))
	return resources, nil
}
