package codegen

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/atoolz/terraxi/internal/discovery"
	"github.com/atoolz/terraxi/internal/graph"
	"github.com/atoolz/terraxi/pkg/types"
)

// Generator handles delegating HCL generation to terraform/tofu import
// and post-processing the output for production quality.
type Generator struct {
	engine    types.Engine
	outputDir string
	cfg       discovery.ProviderConfig
	depGraph  *graph.DependencyGraph
	names     *NameResolver
}

// NewGenerator creates a new HCL code generator.
func NewGenerator(engine types.Engine, outputDir string, cfg discovery.ProviderConfig, depGraph *graph.DependencyGraph) *Generator {
	return &Generator{
		engine:    engine,
		outputDir: outputDir,
		cfg:       cfg,
		depGraph:  depGraph,
		names:     NewNameResolver(),
	}
}

// GenerateImportBlock creates a Terraform import block for a resource.
func (g *Generator) GenerateImportBlock(resource types.Resource) string {
	tfName := g.names.Resolve(resource)

	return fmt.Sprintf(`import {
  to = %s.%s
  id = %q
}
`, resource.Type, tfName, resource.ID)
}

// GenerateAll creates import blocks, writes them to disk, runs terraform/tofu
// import to generate HCL, then post-processes the output.
func (g *Generator) GenerateAll(ctx context.Context, resources []types.Resource) error {
	// Reset name resolver for a clean generation pass
	g.names.Reset()

	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := g.writeProvidersFile(); err != nil {
		return fmt.Errorf("failed to write providers.tf: %w", err)
	}

	// Collect and write import blocks to imports.tf
	var buf strings.Builder
	for _, r := range resources {
		buf.WriteString(g.GenerateImportBlock(r))
		buf.WriteString("\n")
	}

	importsFile := filepath.Join(g.outputDir, "imports.tf")
	if err := os.WriteFile(importsFile, []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("failed to write import blocks: %w", err)
	}

	// Run terraform/tofu plan -generate-config-out to produce raw HCL
	generatedFile := filepath.Join(g.outputDir, "generated.tf")
	if err := g.runImport(ctx, generatedFile); err != nil {
		return err
	}

	// Post-process the generated HCL
	rawHCL, err := os.ReadFile(generatedFile)
	if err != nil {
		return fmt.Errorf("failed to read generated HCL: %w", err)
	}

	pp := NewPostProcessor(g.depGraph)
	processed, err := pp.Process(rawHCL, resources)
	if err != nil {
		return fmt.Errorf("post-processing failed: %w", err)
	}

	if err := os.WriteFile(generatedFile, processed, 0644); err != nil {
		return fmt.Errorf("failed to write post-processed HCL: %w", err)
	}

	// Best-effort cleanup; not fatal if it fails
	_ = os.Remove(importsFile)

	return nil
}

// writeProvidersFile creates a providers.tf with the correct provider config.
func (g *Generator) writeProvidersFile() error {
	var providerBlock string
	switch {
	case g.cfg.Profile != "":
		providerBlock = fmt.Sprintf(`terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region  = %q
  profile = %q
}
`, g.cfg.Region, g.cfg.Profile)
	default:
		providerBlock = fmt.Sprintf(`terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = %q
}
`, g.cfg.Region)
	}

	return os.WriteFile(filepath.Join(g.outputDir, "providers.tf"), []byte(providerBlock), 0644)
}

// terraformInitNeeded checks if providers are already downloaded.
// Uses .terraform.lock.hcl as the canonical signal of a completed init.
func (g *Generator) terraformInitNeeded() bool {
	lockFile := filepath.Join(g.outputDir, ".terraform.lock.hcl")
	_, err := os.Stat(lockFile)
	return os.IsNotExist(err)
}

// runImport executes terraform or tofu to generate HCL from import blocks.
func (g *Generator) runImport(ctx context.Context, configOut string) error {
	binary := string(g.engine)

	// Skip init if providers are already cached
	if g.terraformInitNeeded() {
		slog.Info("Running terraform init (downloading provider)...")
		initCmd := exec.CommandContext(ctx, binary, "init")
		initCmd.Dir = g.outputDir
		if out, err := initCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s init failed: %w\nOutput: %s", binary, err, string(out))
		}
	} else {
		slog.Debug("Skipping terraform init (providers cached)")
	}

	// terraform plan -generate-config-out
	planCmd := exec.CommandContext(ctx, binary, "plan", "-generate-config-out="+configOut)
	planCmd.Dir = g.outputDir
	if out, err := planCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s plan failed: %w\nOutput: %s", binary, err, string(out))
	}

	return nil
}
