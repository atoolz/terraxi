package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/atoolz/terraxi/pkg/types"
)

type mockLambda struct {
	listFunctionsFn func(ctx context.Context, input *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error)
	listLayersFn    func(ctx context.Context, input *lambda.ListLayersInput, optFns ...func(*lambda.Options)) (*lambda.ListLayersOutput, error)
	listTagsFn      func(ctx context.Context, input *lambda.ListTagsInput, optFns ...func(*lambda.Options)) (*lambda.ListTagsOutput, error)
}

func (m *mockLambda) ListFunctions(ctx context.Context, input *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error) {
	return m.listFunctionsFn(ctx, input, optFns...)
}
func (m *mockLambda) ListLayers(ctx context.Context, input *lambda.ListLayersInput, optFns ...func(*lambda.Options)) (*lambda.ListLayersOutput, error) {
	return m.listLayersFn(ctx, input, optFns...)
}
func (m *mockLambda) ListTags(ctx context.Context, input *lambda.ListTagsInput, optFns ...func(*lambda.Options)) (*lambda.ListTagsOutput, error) {
	return m.listTagsFn(ctx, input, optFns...)
}

func TestDiscoverLambdaFunctions_WithVPCAndRoleDeps(t *testing.T) {
	p := NewWithClients("us-east-1", WithLambda(&mockLambda{
		listFunctionsFn: func(_ context.Context, _ *lambda.ListFunctionsInput, _ ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error) {
			return &lambda.ListFunctionsOutput{
				Functions: []lambdatypes.FunctionConfiguration{{
					FunctionName: ptr("my-function"),
					FunctionArn:  ptr("arn:aws:lambda:us-east-1:123456:function:my-function"),
					Role:         ptr("arn:aws:iam::123456:role/lambda-exec-role"),
					VpcConfig: &lambdatypes.VpcConfigResponse{
						SubnetIds:        []string{"subnet-111", "subnet-222"},
						SecurityGroupIds: []string{"sg-333"},
					},
				}},
			}, nil
		},
		listTagsFn: func(_ context.Context, _ *lambda.ListTagsInput, _ ...func(*lambda.Options)) (*lambda.ListTagsOutput, error) {
			return &lambda.ListTagsOutput{
				Tags: map[string]string{"env": "prod"},
			}, nil
		},
	}))

	resources, err := discoverLambdaFunctions(context.Background(), p, types.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 function, got %d", len(resources))
	}
	r := resources[0]
	if r.ID != "my-function" {
		t.Errorf("expected ID my-function, got %s", r.ID)
	}
	if r.Tags["env"] != "prod" {
		t.Errorf("expected env=prod, got %v", r.Tags)
	}
	// 2 subnets + 1 SG + 1 IAM role = 4 deps
	if len(r.Dependencies) != 4 {
		t.Errorf("expected 4 deps (2 subnets + 1 sg + 1 role), got %d: %v", len(r.Dependencies), r.Dependencies)
	}
}

func TestRoleNameFromARN(t *testing.T) {
	tests := []struct {
		arn  string
		want string
	}{
		{"arn:aws:iam::123456:role/my-role", "my-role"},
		{"arn:aws:iam::123456:role/path/nested-role", "nested-role"},
		{"simple-role-name", "simple-role-name"},
	}
	for _, tt := range tests {
		got := roleNameFromARN(tt.arn)
		if got != tt.want {
			t.Errorf("roleNameFromARN(%q) = %q, want %q", tt.arn, got, tt.want)
		}
	}
}
