package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	"github.com/ahlert/terraxi/pkg/types"
)

type mockIAM struct {
	listRolesFn            func(ctx context.Context, input *iam.ListRolesInput, optFns ...func(*iam.Options)) (*iam.ListRolesOutput, error)
	listPoliciesFn         func(ctx context.Context, input *iam.ListPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListPoliciesOutput, error)
	listUsersFn            func(ctx context.Context, input *iam.ListUsersInput, optFns ...func(*iam.Options)) (*iam.ListUsersOutput, error)
	listGroupsFn           func(ctx context.Context, input *iam.ListGroupsInput, optFns ...func(*iam.Options)) (*iam.ListGroupsOutput, error)
	listInstanceProfilesFn func(ctx context.Context, input *iam.ListInstanceProfilesInput, optFns ...func(*iam.Options)) (*iam.ListInstanceProfilesOutput, error)
}

func (m *mockIAM) ListRoles(ctx context.Context, input *iam.ListRolesInput, optFns ...func(*iam.Options)) (*iam.ListRolesOutput, error) {
	return m.listRolesFn(ctx, input, optFns...)
}
func (m *mockIAM) ListPolicies(ctx context.Context, input *iam.ListPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListPoliciesOutput, error) {
	return m.listPoliciesFn(ctx, input, optFns...)
}
func (m *mockIAM) ListUsers(ctx context.Context, input *iam.ListUsersInput, optFns ...func(*iam.Options)) (*iam.ListUsersOutput, error) {
	return m.listUsersFn(ctx, input, optFns...)
}
func (m *mockIAM) ListGroups(ctx context.Context, input *iam.ListGroupsInput, optFns ...func(*iam.Options)) (*iam.ListGroupsOutput, error) {
	return m.listGroupsFn(ctx, input, optFns...)
}
func (m *mockIAM) ListInstanceProfiles(ctx context.Context, input *iam.ListInstanceProfilesInput, optFns ...func(*iam.Options)) (*iam.ListInstanceProfilesOutput, error) {
	return m.listInstanceProfilesFn(ctx, input, optFns...)
}

func TestDiscoverIAMRoles_SkipsServiceLinked(t *testing.T) {
	p := NewWithClients("us-east-1", WithIAM(&mockIAM{
		listRolesFn: func(_ context.Context, _ *iam.ListRolesInput, _ ...func(*iam.Options)) (*iam.ListRolesOutput, error) {
			return &iam.ListRolesOutput{
				Roles: []iamtypes.Role{
					{RoleName: ptr("my-app-role"), Path: ptr("/")},
					{RoleName: ptr("AWSServiceRoleForECS"), Path: ptr("/aws-service-role/ecs.amazonaws.com/")},
					{RoleName: ptr("short-path-role"), Path: ptr("/x/")}, // short path, should not panic
				},
			}, nil
		},
	}))

	resources, err := discoverIAMRoles(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 roles (service-linked skipped, short-path kept), got %d", len(resources))
	}
	if resources[0].Name != "my-app-role" {
		t.Errorf("expected my-app-role, got %s", resources[0].Name)
	}
	if resources[0].Region != "" {
		t.Errorf("IAM resources should have empty Region, got %q", resources[0].Region)
	}
}

func TestDiscoverIAMPolicies_CustomerManagedOnly(t *testing.T) {
	p := NewWithClients("us-east-1", WithIAM(&mockIAM{
		listPoliciesFn: func(_ context.Context, input *iam.ListPoliciesInput, _ ...func(*iam.Options)) (*iam.ListPoliciesOutput, error) {
			if input.Scope == "Local" {
				return &iam.ListPoliciesOutput{
					Policies: []iamtypes.Policy{
						{PolicyName: ptr("my-policy"), Arn: ptr("arn:aws:iam::123456:policy/my-policy")},
					},
				}, nil
			}
			return &iam.ListPoliciesOutput{}, nil
		},
	}))

	resources, err := discoverIAMPolicies(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 customer-managed policy, got %d", len(resources))
	}
	if resources[0].ID != "arn:aws:iam::123456:policy/my-policy" {
		t.Errorf("expected ARN as ID, got %s", resources[0].ID)
	}
}

func TestDiscoverIAMInstanceProfiles_WithRoleDependency(t *testing.T) {
	p := NewWithClients("us-east-1", WithIAM(&mockIAM{
		listInstanceProfilesFn: func(_ context.Context, _ *iam.ListInstanceProfilesInput, _ ...func(*iam.Options)) (*iam.ListInstanceProfilesOutput, error) {
			return &iam.ListInstanceProfilesOutput{
				InstanceProfiles: []iamtypes.InstanceProfile{{
					InstanceProfileName: ptr("my-profile"),
					Roles: []iamtypes.Role{
						{RoleName: ptr("my-role")},
					},
				}},
			}, nil
		},
	}))

	resources, err := discoverIAMInstanceProfiles(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 instance profile, got %d", len(resources))
	}
	if len(resources[0].Dependencies) != 1 || resources[0].Dependencies[0].ID != "my-role" {
		t.Errorf("expected dependency on my-role, got %v", resources[0].Dependencies)
	}
}
