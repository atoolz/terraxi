package aws

import (
	"context"
	"fmt"
	"log/slog"

	awsutil "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/atoolz/terraxi/internal/discovery"
	"github.com/atoolz/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_lambda_function", discoverLambdaFunctions)
	RegisterDiscoverer("aws_lambda_layer_version", discoverLambdaLayers)
}

func discoverLambdaFunctions(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.lambda.ListFunctions(ctx, &lambda.ListFunctionsInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for lambda:ListFunctions: %w", err)
			}
			return nil, fmt.Errorf("listing Lambda functions: %w", err)
		}

		for _, fn := range out.Functions {
			fnName := awsutil.ToString(fn.FunctionName)

			// Fetch tags via ARN
			tags := map[string]string{}
			if fn.FunctionArn != nil {
				tagOut, tagErr := p.lambda.ListTags(ctx, &lambda.ListTagsInput{
					Resource: fn.FunctionArn,
				})
				if tagErr == nil {
					tags = tagOut.Tags
				}
			}

			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			r := types.Resource{
				Type:   "aws_lambda_function",
				ID:     fnName,
				Name:   fnName,
				Region: p.region,
				Tags:   tags,
			}

			// Wire VPC config dependencies if the function is VPC-attached
			if fn.VpcConfig != nil {
				for _, subnetID := range fn.VpcConfig.SubnetIds {
					r.Dependencies = append(r.Dependencies, types.ResourceRef{
						Type: "aws_subnet", ID: subnetID,
					})
				}
				for _, sgID := range fn.VpcConfig.SecurityGroupIds {
					r.Dependencies = append(r.Dependencies, types.ResourceRef{
						Type: "aws_security_group", ID: sgID,
					})
				}
			}

			// Wire IAM role dependency
			if fn.Role != nil {
				roleName := roleNameFromARN(*fn.Role)
				if roleName != "" {
					r.Dependencies = append(r.Dependencies, types.ResourceRef{
						Type: "aws_iam_role", ID: roleName,
					})
				}
			}

			resources = append(resources, r)
		}

		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}

	slog.Debug("Lambda functions discovery complete", "count", len(resources))
	return resources, nil
}

func discoverLambdaLayers(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.lambda.ListLayers(ctx, &lambda.ListLayersInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for lambda:ListLayers: %w", err)
			}
			return nil, fmt.Errorf("listing Lambda layers: %w", err)
		}

		for _, layer := range out.Layers {
			layerName := awsutil.ToString(layer.LayerName)
			latestVersion := layer.LatestMatchingVersion

			if latestVersion == nil {
				continue
			}

			// Import ID for layer version: arn
			resources = append(resources, types.Resource{
				Type:   "aws_lambda_layer_version",
				ID:     awsutil.ToString(latestVersion.LayerVersionArn),
				Name:   layerName,
				Region: p.region,
			})
		}

		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}

	slog.Debug("Lambda layers discovery complete", "count", len(resources))
	return resources, nil
}

// roleNameFromARN extracts the role name from an IAM role ARN.
// e.g., "arn:aws:iam::123456:role/my-role" -> "my-role"
// e.g., "arn:aws:iam::123456:role/path/nested-role" -> "nested-role"
func roleNameFromARN(arn string) string {
	// Find the last '/' which separates the role name from the path
	for i := len(arn) - 1; i >= 0; i-- {
		if arn[i] == '/' {
			return arn[i+1:]
		}
	}
	// No slash found: might be just a role name, or ARN with colon separator
	for i := len(arn) - 1; i >= 0; i-- {
		if arn[i] == ':' {
			return arn[i+1:]
		}
	}
	return arn
}
