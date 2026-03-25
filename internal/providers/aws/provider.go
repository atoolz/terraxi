package aws

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/atoolz/terraxi/internal/codegen"
	"github.com/atoolz/terraxi/internal/discovery"
	"github.com/atoolz/terraxi/pkg/types"
)

// Provider discovers AWS resources.
type Provider struct {
	region  string
	profile string

	ec2            EC2API
	s3             S3API
	iam            IAMAPI
	rds            RDSAPI
	elb            ELBAPI
	route53        Route53API
	lambda         LambdaAPI
	ecs            ECSAPI
	cloudwatch     CloudWatchAPI
	cloudwatchlogs CloudWatchLogsAPI
}

// New creates a new AWS provider.
func New() *Provider {
	return &Provider{}
}

// NewWithClients creates a Provider with pre-built clients (for testing).
func NewWithClients(region string, opts ...ClientOption) *Provider {
	p := &Provider{region: region}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ClientOption configures a Provider with a specific client (for testing).
type ClientOption func(*Provider)

func WithEC2(c EC2API) ClientOption               { return func(p *Provider) { p.ec2 = c } }
func WithS3(c S3API) ClientOption                 { return func(p *Provider) { p.s3 = c } }
func WithIAM(c IAMAPI) ClientOption               { return func(p *Provider) { p.iam = c } }
func WithRDS(c RDSAPI) ClientOption               { return func(p *Provider) { p.rds = c } }
func WithELB(c ELBAPI) ClientOption               { return func(p *Provider) { p.elb = c } }
func WithRoute53(c Route53API) ClientOption       { return func(p *Provider) { p.route53 = c } }
func WithLambda(c LambdaAPI) ClientOption         { return func(p *Provider) { p.lambda = c } }
func WithECS(c ECSAPI) ClientOption               { return func(p *Provider) { p.ecs = c } }
func WithCloudWatch(c CloudWatchAPI) ClientOption { return func(p *Provider) { p.cloudwatch = c } }
func WithCloudWatchLogs(c CloudWatchLogsAPI) ClientOption {
	return func(p *Provider) { p.cloudwatchlogs = c }
}

func (p *Provider) Name() string { return "aws" }

func (p *Provider) Configure(ctx context.Context, cfg discovery.ProviderConfig) error {
	p.region = cfg.Region
	p.profile = cfg.Profile
	if p.region == "" {
		return fmt.Errorf("AWS region is required: set --region or AWS_DEFAULT_REGION")
	}

	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(p.region))
	if p.profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(p.profile))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials: %w\nEnsure AWS_PROFILE, AWS_ACCESS_KEY_ID, or instance role is configured", err)
	}

	// Validate credentials early
	stsClient := sts.NewFromConfig(awsCfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("AWS credential validation failed: %w", err)
	}
	slog.Info("AWS credentials validated",
		"account", aws.ToString(identity.Account),
		"arn", aws.ToString(identity.Arn),
		"region", p.region,
	)

	// Initialize all service clients
	p.ec2 = ec2.NewFromConfig(awsCfg)
	p.s3 = s3.NewFromConfig(awsCfg)
	p.iam = iam.NewFromConfig(awsCfg)
	p.rds = rds.NewFromConfig(awsCfg)
	p.elb = elasticloadbalancingv2.NewFromConfig(awsCfg)
	p.route53 = route53.NewFromConfig(awsCfg)
	p.lambda = lambda.NewFromConfig(awsCfg)
	p.ecs = ecs.NewFromConfig(awsCfg)
	p.cloudwatch = cloudwatch.NewFromConfig(awsCfg)
	p.cloudwatchlogs = cloudwatchlogs.NewFromConfig(awsCfg)

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
var discoverers = map[string]discovererFunc{}

// RegisterDiscoverer registers a discovery function for a resource type.
func RegisterDiscoverer(resourceType string, fn discovererFunc) {
	discoverers[resourceType] = fn
}
