package aws

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	awsutil "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	"github.com/ahlert/terraxi/internal/discovery"
	"github.com/ahlert/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_iam_role", discoverIAMRoles)
	RegisterDiscoverer("aws_iam_policy", discoverIAMPolicies)
	RegisterDiscoverer("aws_iam_user", discoverIAMUsers)
	RegisterDiscoverer("aws_iam_group", discoverIAMGroups)
	RegisterDiscoverer("aws_iam_instance_profile", discoverIAMInstanceProfiles)
}

// IAM resources are global. Region is always empty.

func discoverIAMRoles(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.iam.ListRoles(ctx, &iam.ListRolesInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for iam:ListRoles: %w", err)
			}
			return nil, fmt.Errorf("listing IAM roles: %w", err)
		}

		for _, role := range out.Roles {
			tags := iamTagsToMap(role.Tags)
			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			// Skip AWS service-linked roles (not user-manageable)
			path := awsutil.ToString(role.Path)
			if strings.HasPrefix(path, "/aws-service-role/") {
				continue
			}

			resources = append(resources, types.Resource{
				Type:   "aws_iam_role",
				ID:     awsutil.ToString(role.RoleName),
				Name:   awsutil.ToString(role.RoleName),
				Region: "",
				Tags:   tags,
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	slog.Debug("IAM roles discovery complete", "count", len(resources))
	return resources, nil
}

func discoverIAMPolicies(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.iam.ListPolicies(ctx, &iam.ListPoliciesInput{
			Scope:  iamtypes.PolicyScopeTypeLocal,
			Marker: marker,
		})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for iam:ListPolicies: %w", err)
			}
			return nil, fmt.Errorf("listing IAM policies: %w", err)
		}

		for _, pol := range out.Policies {
			tags := iamTagsToMap(pol.Tags)
			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			resources = append(resources, types.Resource{
				Type:   "aws_iam_policy",
				ID:     awsutil.ToString(pol.Arn),
				Name:   awsutil.ToString(pol.PolicyName),
				Region: "",
				Tags:   tags,
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	slog.Debug("IAM policies discovery complete", "count", len(resources))
	return resources, nil
}

func discoverIAMUsers(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.iam.ListUsers(ctx, &iam.ListUsersInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for iam:ListUsers: %w", err)
			}
			return nil, fmt.Errorf("listing IAM users: %w", err)
		}

		for _, user := range out.Users {
			tags := iamTagsToMap(user.Tags)
			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			resources = append(resources, types.Resource{
				Type:   "aws_iam_user",
				ID:     awsutil.ToString(user.UserName),
				Name:   awsutil.ToString(user.UserName),
				Region: "",
				Tags:   tags,
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	slog.Debug("IAM users discovery complete", "count", len(resources))
	return resources, nil
}

func discoverIAMGroups(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	// IAM groups do not support inline tags via ListGroups API.
	// If a tag filter is active, skip groups entirely (consistent behavior).
	if len(filter.Tags) > 0 {
		slog.Debug("Skipping IAM groups: tag filtering not supported by ListGroups API")
		return nil, nil
	}

	var resources []types.Resource
	var marker *string

	for {
		out, err := p.iam.ListGroups(ctx, &iam.ListGroupsInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for iam:ListGroups: %w", err)
			}
			return nil, fmt.Errorf("listing IAM groups: %w", err)
		}

		for _, group := range out.Groups {
			resources = append(resources, types.Resource{
				Type:   "aws_iam_group",
				ID:     awsutil.ToString(group.GroupName),
				Name:   awsutil.ToString(group.GroupName),
				Region: "",
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	slog.Debug("IAM groups discovery complete", "count", len(resources))
	return resources, nil
}

func discoverIAMInstanceProfiles(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.iam.ListInstanceProfiles(ctx, &iam.ListInstanceProfilesInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for iam:ListInstanceProfiles: %w", err)
			}
			return nil, fmt.Errorf("listing IAM instance profiles: %w", err)
		}

		for _, ip := range out.InstanceProfiles {
			tags := iamTagsToMap(ip.Tags)
			if !discovery.MatchesTags(types.Resource{Tags: tags}, filter.Tags) {
				continue
			}

			r := types.Resource{
				Type:   "aws_iam_instance_profile",
				ID:     awsutil.ToString(ip.InstanceProfileName),
				Name:   awsutil.ToString(ip.InstanceProfileName),
				Region: "",
				Tags:   tags,
			}
			for _, role := range ip.Roles {
				r.Dependencies = append(r.Dependencies, types.ResourceRef{
					Type: "aws_iam_role", ID: awsutil.ToString(role.RoleName),
				})
			}
			resources = append(resources, r)
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	slog.Debug("IAM instance profiles discovery complete", "count", len(resources))
	return resources, nil
}
