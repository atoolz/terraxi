# Terraxi

**Discover your cloud. Generate production-ready Terraform/OpenTofu code.**

Terraxi scans your AWS account (GCP and Azure coming soon), discovers every resource that exists, and generates clean, modular Terraform or OpenTofu configuration files.

```bash
terraxi discover aws --region us-east-1
```

You get a directory of `.tf` files organized by service, with extracted variables, resolved cross-resource dependencies, and code that actually passes `terraform validate`.

## The Problem

Most companies built their infrastructure by clicking through the AWS console before adopting Infrastructure as Code. Now they want everything under Terraform, but importing is painful:

- **`terraform import` works one resource at a time.** 500 S3 buckets means 500 manual commands.
- **`terraform import` doesn't discover anything.** You need to know every resource ID upfront.
- **The generated code is garbage.** Hardcoded IDs, no variables, no dependency references, frequently fails validation.

[Terraformer](https://github.com/GoogleCloudPlatform/terraformer) (Google, 14.5k stars) solved this, but was archived after its creator (Sergey Lanzman, 737 commits) left Google/Waze in 2021 and nobody took over. Bus factor = 1. The demand remained.

`terraform query` (TF 1.14) only supports 3 AWS resource types. It will take years to catch up. OpenTofu has nothing equivalent. Zero.

## How Terraxi Is Different

- **Auto-discovers everything.** Scans your entire account via cloud provider APIs. No native tool does discovery.
- **Delegates HCL generation to `terraform import` itself.** This avoids the maintenance treadmill that killed Terraformer (maintaining custom generators for hundreds of resource types).
- **Post-processes for production quality.** Replaces hardcoded IDs with data source references, extracts variables, organizes into modules, collapses similar resources into `for_each`. Nobody else does this.
- **First-class OpenTofu support.** No competitor does this. Doubles the addressable market.
- **AWS-only for now.** GCP and Azure coming in future phases.

## Why the Successor Won't Die the Same Way

Terraformer died from bus factor = 1, not from lack of market demand. Terraxi's delegation architecture (own discovery + HCL via `terraform import`) reduces maintenance burden by ~70%. A plugin system for providers allows community contributions without touching core code.

**Closest competitor:** [TerraCognita](https://github.com/cycloidio/terracognita) (2.3k stars, last commit Sep 2025, stagnant). `aztfexport` is Azure-only. Commercial tools (Firefly, Spacelift) are expensive.

## Quick Start

### Install

```bash
# macOS / Linux
brew install atoolz/tap/terraxi

# From source
go install github.com/atoolz/terraxi/cmd/terraxi@latest

# Or download binary from Releases
```

### Usage

```bash
# Discover all resources in a region
terraxi discover aws --region us-east-1

# Discover specific services only
terraxi discover aws --services ec2,s3,iam --region us-east-1

# Filter by tags
terraxi discover aws --filter "tags.env=production" --region us-east-1

# Preview what would be discovered (no code generation)
terraxi discover aws --dry-run --region us-east-1

# Use OpenTofu instead of Terraform
terraxi discover aws --engine tofu --region us-east-1

# Output as JSON
terraxi discover aws --region us-east-1 --format json

# Custom output directory
terraxi discover aws --region us-east-1 -o ./infrastructure

# Multi-region discovery
terraxi discover aws --regions us-east-1,eu-west-1,ap-southeast-1

# Generate per-service module directories
terraxi discover aws --region us-east-1 --structure modules

# Save discovery inventory as JSON
terraxi discover aws --region us-east-1 --inventory ./inventory.json
```

### AWS Credentials

Terraxi uses the standard AWS credential chain:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. AWS SSO / IAM Identity Center
4. EC2 instance metadata (IMDSv2)
5. ECS task role

```bash
# Using a named profile
terraxi discover aws --region us-east-1 --profile my-profile

# Using environment variables
AWS_PROFILE=my-profile terraxi discover aws --region us-east-1
```

## Output Structure

```
imported/
├── ec2/
│   └── main.tf           # EC2 instances, security groups, EBS volumes
├── vpc/
│   └── main.tf           # VPCs, subnets, route tables, gateways
├── s3/
│   └── main.tf           # S3 buckets with policies and lifecycle rules
├── iam/
│   └── main.tf           # IAM roles, policies, users, groups
├── providers.tf           # Provider configuration with version pin
└── variables.tf           # Extracted variables (region, common tags)
```

## Supported Resource Types

| Service | Resources |
|---------|-----------|
| **EC2** | Instances, Security Groups, Security Group Rules, Key Pairs, EBS Volumes, Elastic IPs |
| **VPC** | VPCs, Subnets, Route Tables, NAT Gateways, Internet Gateways |
| **S3** | Buckets (with policies, ACLs, versioning, lifecycle) |
| **IAM** | Roles, Policies, Users, Groups, Instance Profiles |
| **RDS** | Instances, Aurora Clusters, Subnet Groups, Parameter Groups |
| **ELB** | Load Balancers (ALB/NLB), Target Groups, Listeners |
| **Route53** | Hosted Zones, DNS Records |
| **Lambda** | Functions, Layers |
| **ECS** | Clusters, Services, Task Definitions |
| **CloudWatch** | Log Groups, Metric Alarms |

## Pricing

| | Free (Apache 2.0) | Pro |
|---|---|---|
| Discovery | All providers, all resource types | All |
| Import + post-processing | Full | Full |
| OpenTofu support | Yes | Yes |
| CLI usage | Unlimited | Unlimited |
| Incremental sync | - | Re-run and import only net-new resources |
| Drift reports (HTML/JSON) | - | Yes |
| Compliance mapping | - | SOC2, HIPAA resource tagging |
| CI/CD integration | - | GitHub Action, GitLab CI |
| Multi-account RBAC | - | Cross-account role assumption |
| Price | $0 | $29/seat/month |

## Why Apache 2.0

Terraxi is a CLI that runs locally or in CI. You run `terraxi discover aws`, it generates your `.tf` files, and exits. It is not a service running on a network.

AGPLv3 protects against someone offering your software as SaaS. But nobody would offer "Terraxi as a Service" because the value is in running it in your environment, with your AWS credentials, generating files on your filesystem. The network clause would never be triggered.

Meanwhile, AGPLv3 would create real friction: many enterprise companies have a blanket ban on AGPL due to fear of copyleft contamination. The Terraform/OpenTofu ecosystem is entirely Apache 2.0 and MPL 2.0. The original Terraformer (which we succeed) was Apache 2.0. Using AGPL would clash with the ecosystem and push away exactly the audience we want to reach: DevOps engineers at companies importing legacy infrastructure.

Apache 2.0 maximizes adoption with no risk. Commercial features (incremental sync, drift reports, compliance mapping, CI/CD integration) live in a separate repository (`terraxi-pro`) with a proprietary license from day one. The core never changes license. A CLA ensures community contributions can be used in commercial context.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for details on adding new resource types, running tests, and submitting PRs.

```bash
# Build
make build

# Run tests
make test

# Lint
make lint
```

## License

Apache License 2.0. See [LICENSE](LICENSE) for the full text.
