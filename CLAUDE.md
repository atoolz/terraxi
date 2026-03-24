# Terraxi

## What is this

Terraxi is a Go CLI that auto-discovers cloud resources and generates production-quality Terraform/OpenTofu code. It is the spiritual successor to Google's Terraformer (14.5k stars, archived March 2026).

## Philosophy

**"Discover everything. Import anything. Own your infrastructure."**

Terraxi does NOT reimplement HCL generation for every resource type (this killed Terraformer). Instead:

1. **Discovery layer** (our code): scans cloud APIs to find all resources
2. **Import layer** (delegated): uses `terraform import -generate-config-out` or `tofu import` to generate raw HCL
3. **Post-processing layer** (our code): transforms raw HCL into production-quality modules with variables, data sources, and for_each patterns

## Architecture

```
cmd/terraxi/          CLI entrypoint (cobra)
internal/
  discovery/           Core discovery engine (parallel workers, filtering)
  providers/aws/       AWS resource discovery (EC2, S3, IAM, VPC, RDS, etc.)
  codegen/             HCL post-processing (dependency linking, variable extraction, module structuring)
  graph/               Resource dependency graph (DAG)
  output/              Output formatters (HCL files, JSON inventory, reports)
pkg/types/             Shared types (Resource, Provider, Filter)
```

## Key Design Decisions

- **Go** because Terraform ecosystem is Go. Single binary, no runtime deps.
- **Provider interface** so community can add providers via plugins
- **Delegate to terraform/tofu import** for HCL generation to avoid maintenance treadmill
- **Post-processing** is the core differentiator: nobody else produces clean HCL
- **OpenTofu first-class** support (not just Terraform)
- **AWS-only MVP**, GCP and Azure in future phases

## MVP Scope (Phase 1)

Target: 40 AWS resource types covering ~80% of real-world import needs:
- EC2: instances, security groups, key pairs, EBS volumes, AMIs, EIPs
- VPC: VPCs, subnets, route tables, NAT gateways, internet gateways
- S3: buckets with policy, ACL, versioning, lifecycle
- IAM: roles, policies, instance profiles, users, groups
- RDS: instances, clusters, subnet groups, parameter groups
- ELB/ALB: load balancers, target groups, listeners
- Route53: hosted zones, records
- Lambda: functions, layers
- ECS: clusters, services, task definitions
- CloudWatch: log groups, alarms

## CLI Interface

```bash
terraxi discover aws --region us-east-1
terraxi discover aws --services ec2,s3,iam
terraxi discover aws --filter "tags.team=platform"
terraxi discover aws --dry-run
terraxi discover aws --output ./imported --structure modules
terraxi discover aws --engine tofu
terraxi drift aws --state ./terraform.tfstate
```

## Conventions

- Use `cobra` for CLI framework
- Use `aws-sdk-go-v2` for AWS
- Use `hashicorp/hcl/v2` and `hclwrite` for HCL manipulation
- Use `sourcegraph/conc` for structured concurrency
- Use `charmbracelet/bubbletea` for interactive TUI (resource selection)
- Errors must be actionable: tell the user what to do, not just what went wrong
- All output is machine-parseable (JSON) unless --human flag is set
