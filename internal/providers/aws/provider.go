package aws

import (
	"context"
	"fmt"

	"github.com/ahlert/terraxi/internal/codegen"
	"github.com/ahlert/terraxi/internal/discovery"
	"github.com/ahlert/terraxi/pkg/types"
)

// Provider discovers AWS resources.
type Provider struct {
	region  string
	profile string
}

// New creates a new AWS provider.
func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string { return "aws" }

func (p *Provider) Configure(_ context.Context, cfg discovery.ProviderConfig) error {
	p.region = cfg.Region
	p.profile = cfg.Profile
	if p.region == "" {
		return fmt.Errorf("region is required for AWS provider")
	}

	// Register service mappings for post-processing (single source of truth)
	for _, rt := range p.ListResourceTypes() {
		codegen.RegisterServiceMapping(rt.Type, rt.Service)
	}

	return nil
}

func (p *Provider) ListResourceTypes() []types.ResourceType {
	return []types.ResourceType{
		// EC2
		{Type: "aws_instance", Service: "ec2", Description: "EC2 instances"},
		{Type: "aws_security_group", Service: "ec2", Description: "Security groups"},
		{Type: "aws_key_pair", Service: "ec2", Description: "Key pairs"},
		{Type: "aws_ebs_volume", Service: "ec2", Description: "EBS volumes"},
		{Type: "aws_eip", Service: "ec2", Description: "Elastic IPs"},

		// VPC
		{Type: "aws_vpc", Service: "vpc", Description: "VPCs"},
		{Type: "aws_subnet", Service: "vpc", Description: "Subnets"},
		{Type: "aws_route_table", Service: "vpc", Description: "Route tables"},
		{Type: "aws_nat_gateway", Service: "vpc", Description: "NAT gateways"},
		{Type: "aws_internet_gateway", Service: "vpc", Description: "Internet gateways"},

		// S3
		{Type: "aws_s3_bucket", Service: "s3", Description: "S3 buckets"},

		// IAM
		{Type: "aws_iam_role", Service: "iam", Description: "IAM roles"},
		{Type: "aws_iam_policy", Service: "iam", Description: "IAM policies"},
		{Type: "aws_iam_user", Service: "iam", Description: "IAM users"},
		{Type: "aws_iam_group", Service: "iam", Description: "IAM groups"},
		{Type: "aws_iam_instance_profile", Service: "iam", Description: "Instance profiles"},

		// RDS
		{Type: "aws_db_instance", Service: "rds", Description: "RDS instances"},
		{Type: "aws_db_subnet_group", Service: "rds", Description: "DB subnet groups"},
		{Type: "aws_db_parameter_group", Service: "rds", Description: "DB parameter groups"},

		// ELB
		{Type: "aws_lb", Service: "elb", Description: "Load balancers (ALB/NLB)"},
		{Type: "aws_lb_target_group", Service: "elb", Description: "Target groups"},
		{Type: "aws_lb_listener", Service: "elb", Description: "LB listeners"},

		// Route53
		{Type: "aws_route53_zone", Service: "route53", Description: "Hosted zones"},
		{Type: "aws_route53_record", Service: "route53", Description: "DNS records"},

		// Lambda
		{Type: "aws_lambda_function", Service: "lambda", Description: "Lambda functions"},
		{Type: "aws_lambda_layer_version", Service: "lambda", Description: "Lambda layers"},

		// ECS
		{Type: "aws_ecs_cluster", Service: "ecs", Description: "ECS clusters"},
		{Type: "aws_ecs_service", Service: "ecs", Description: "ECS services"},
		{Type: "aws_ecs_task_definition", Service: "ecs", Description: "ECS task definitions"},

		// CloudWatch
		{Type: "aws_cloudwatch_log_group", Service: "cloudwatch", Description: "Log groups"},
		{Type: "aws_cloudwatch_metric_alarm", Service: "cloudwatch", Description: "Metric alarms"},
	}
}

func (p *Provider) Discover(ctx context.Context, resourceType string, filter types.Filter) ([]types.Resource, error) {
	discoverer, ok := discoverers[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
	return discoverer(ctx, p, filter)
}

// discovererFunc is a function that discovers resources of a specific type.
type discovererFunc func(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error)

// discoverers maps resource types to their discovery functions.
var discoverers = map[string]discovererFunc{
	// Each resource type gets its own discoverer.
	// Implementations go in separate files (ec2.go, s3.go, iam.go, etc.)
}

// RegisterDiscoverer registers a discovery function for a resource type.
func RegisterDiscoverer(resourceType string, fn discovererFunc) {
	discoverers[resourceType] = fn
}
