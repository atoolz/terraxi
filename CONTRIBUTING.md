# Contributing to Terraxi

Thank you for your interest in contributing to Terraxi!

## Quick Start

```bash
git clone https://github.com/atoolz/terraxi.git
cd terraxi
make build
make test
make lint
```

## Adding a New AWS Resource Type

This is the most common type of contribution. Follow these steps:

### 1. Add the method to the client interface

Edit `internal/providers/aws/clients.go` and add the AWS SDK method to the appropriate interface (e.g., `EC2API`, `RDSAPI`):

```go
type EC2API interface {
    // ... existing methods ...
    DescribeNewThing(ctx context.Context, params *ec2.DescribeNewThingInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNewThingOutput, error)
}
```

### 2. Register the resource type

Edit `internal/providers/aws/provider.go` and add the type to `ListResourceTypes()`:

```go
{Type: "aws_new_thing", Service: "ec2", Description: "New things"},
```

### 3. Implement the discoverer

Create or edit the appropriate service file (e.g., `internal/providers/aws/ec2.go`):

```go
func init() {
    RegisterDiscoverer("aws_new_thing", discoverNewThings)
}

func discoverNewThings(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
    // Follow the existing patterns:
    // 1. Paginate using NextToken/Marker
    // 2. Apply tag filtering via discovery.MatchesTags
    // 3. Handle isAccessDenied gracefully (return error, not nil)
    // 4. Handle isNotFound by skipping
    // 5. Extract dependencies via ResourceRef
    // 6. Set Region: "" for global resources (IAM, Route53)
    // 7. Log at slog.Debug when done
}
```

### 4. Add the mock method to the test mock

Edit the corresponding test file and add the method to the mock struct:

```go
type mockEC2 struct {
    // ... existing fields ...
    describeNewThingsFn func(ctx context.Context, input *ec2.DescribeNewThingInput, ...) (*ec2.DescribeNewThingOutput, error)
}

func (m *mockEC2) DescribeNewThing(...) { return m.describeNewThingsFn(...) }
```

### 5. Write tests

Test at minimum:
- Happy path (resources returned correctly)
- Tag filtering works
- Dependencies are wired

### 6. Run checks

```bash
make check  # runs test + lint
```

## Patterns to Follow

- **Pagination**: Always paginate. Use `NextToken` for EC2, `Marker` for RDS/IAM.
- **Error handling**: Use `isAccessDenied()` (non-fatal) and `isNotFound()` (skip silently).
- **Tags**: Use `ec2TagsToMap()`, `rdsTagsToMap()`, etc. from `tags.go`.
- **Names**: Use `nameFromEC2Tags()` or `awsutil.ToString()` for the `Name` field.
- **Global resources**: Set `Region: ""` for IAM, Route53.
- **Logging**: `slog.Debug` for completion messages, `slog.Warn` for non-fatal issues.

## PR Checklist

- [ ] `make check` passes (tests + lint)
- [ ] New resource types have tests with mock clients
- [ ] Discoverers paginate correctly
- [ ] Tag filtering is applied (or `slog.Warn` if not supported)
- [ ] Dependencies are wired where applicable
- [ ] No `fmt.Fprintf(os.Stderr)` (use `slog`)

## Code of Conduct

Be respectful, constructive, and focused on making Terraxi better.

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
