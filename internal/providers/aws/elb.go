package aws

import (
	"context"
	"fmt"
	"log/slog"

	awsutil "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/atoolz/terraxi/internal/discovery"
	"github.com/atoolz/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_lb", discoverLoadBalancers)
	RegisterDiscoverer("aws_lb_target_group", discoverTargetGroups)
	RegisterDiscoverer("aws_lb_listener", discoverLBListeners)
}

func discoverLoadBalancers(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.elb.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for elasticloadbalancing:DescribeLoadBalancers: %w", err)
			}
			return nil, fmt.Errorf("describing load balancers: %w", err)
		}

		// Fetch tags for all LBs in this page
		var arns []string
		for _, lb := range out.LoadBalancers {
			if lb.LoadBalancerArn != nil {
				arns = append(arns, *lb.LoadBalancerArn)
			}
		}

		tagMap := map[string]map[string]string{}
		if len(arns) > 0 {
			tagOut, tagErr := p.elb.DescribeTags(ctx, &elasticloadbalancingv2.DescribeTagsInput{
				ResourceArns: arns,
			})
			if tagErr == nil {
				for _, desc := range tagOut.TagDescriptions {
					tags := make(map[string]string, len(desc.Tags))
					for _, t := range desc.Tags {
						tags[awsutil.ToString(t.Key)] = awsutil.ToString(t.Value)
					}
					tagMap[awsutil.ToString(desc.ResourceArn)] = tags
				}
			}
		}

		for _, lb := range out.LoadBalancers {
			arn := awsutil.ToString(lb.LoadBalancerArn)
			tags := tagMap[arn]
			if tags == nil {
				tags = map[string]string{}
			}

			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			r := types.Resource{
				Type:   "aws_lb",
				ID:     arn,
				Name:   awsutil.ToString(lb.LoadBalancerName),
				Region: p.region,
				Tags:   tags,
			}
			if lb.VpcId != nil {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_vpc", ID: *lb.VpcId,
				})
			}
			for _, sg := range lb.SecurityGroups {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_security_group", ID: sg,
				})
			}
			resources = append(resources, r)
		}

		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}

	slog.Debug("Load balancers discovery complete", "count", len(resources))
	return resources, nil
}

func discoverTargetGroups(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.elb.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for elasticloadbalancing:DescribeTargetGroups: %w", err)
			}
			return nil, fmt.Errorf("describing target groups: %w", err)
		}

		// Fetch tags for target groups in this page
		var tgArns []string
		for _, tg := range out.TargetGroups {
			if tg.TargetGroupArn != nil {
				tgArns = append(tgArns, *tg.TargetGroupArn)
			}
		}

		tgTagMap := map[string]map[string]string{}
		if len(tgArns) > 0 {
			tagOut, tagErr := p.elb.DescribeTags(ctx, &elasticloadbalancingv2.DescribeTagsInput{
				ResourceArns: tgArns,
			})
			if tagErr == nil {
				for _, desc := range tagOut.TagDescriptions {
					tags := make(map[string]string, len(desc.Tags))
					for _, t := range desc.Tags {
						tags[awsutil.ToString(t.Key)] = awsutil.ToString(t.Value)
					}
					tgTagMap[awsutil.ToString(desc.ResourceArn)] = tags
				}
			}
		}

		for _, tg := range out.TargetGroups {
			arn := awsutil.ToString(tg.TargetGroupArn)
			tags := tgTagMap[arn]
			if tags == nil {
				tags = map[string]string{}
			}

			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			r := types.Resource{
				Type:   "aws_lb_target_group",
				ID:     arn,
				Name:   awsutil.ToString(tg.TargetGroupName),
				Region: p.region,
				Tags:   tags,
			}
			if tg.VpcId != nil {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_vpc", ID: *tg.VpcId,
				})
			}
			for _, lbArn := range tg.LoadBalancerArns {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_lb", ID: lbArn,
				})
			}
			resources = append(resources, r)
		}

		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}

	slog.Debug("Target groups discovery complete", "count", len(resources))
	return resources, nil
}

func discoverLBListeners(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	// Listeners require a load balancer ARN. Discover LBs first (with user's filter).
	lbs, err := discoverLoadBalancers(ctx, p, filter)
	if err != nil {
		return nil, err
	}

	var resources []types.Resource
	for _, lb := range lbs {
		lbArn := lb.ID
		var marker *string

		for {
			out, err := p.elb.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{
				LoadBalancerArn: &lbArn,
				Marker:          marker,
			})
			if err != nil {
				if isAccessDenied(err) || isNotFound(err) {
					break
				}
				return nil, fmt.Errorf("describing listeners for %s: %w", lbArn, err)
			}

			for _, l := range out.Listeners {
				resources = append(resources, types.Resource{
					Type:   "aws_lb_listener",
					ID:     awsutil.ToString(l.ListenerArn),
					Name:   fmt.Sprintf("%s-%d", lb.Name, awsutil.ToInt32(l.Port)),
					Region: p.region,
					Dependencies: []types.ResourceRef{
						{Type: "aws_lb", ID: lbArn},
					},
				})
			}

			if out.NextMarker == nil {
				break
			}
			marker = out.NextMarker
		}
	}

	slog.Debug("LB listeners discovery complete", "count", len(resources))
	return resources, nil
}
