package codegen

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ahlert/terraxi/internal/discovery"
	"github.com/ahlert/terraxi/internal/graph"
	"github.com/ahlert/terraxi/pkg/types"
)

// Generator handles delegating HCL generation to terraform/tofu import
// and post-processing the output for production quality.
type Generator struct {
	engine    types.Engine
	outputDir string
	cfg       discovery.ProviderConfig
	depGraph  *graph.DependencyGraph
	nameCount map[string]int
}

// NewGenerator creates a new HCL code generator.
func NewGenerator(engine types.Engine, outputDir string, cfg discovery.ProviderConfig, depGraph *graph.DependencyGraph) *Generator {
	return &Generator{
		engine:    engine,
		outputDir: outputDir,
		cfg:       cfg,
		depGraph:  depGraph,
		nameCount: make(map[string]int),
	}
}

// uniqueName returns a deduplicated Terraform resource name.
// If "my_bucket" was already used, returns "my_bucket_2", "my_bucket_3", etc.
func (g *Generator) uniqueName(resource types.Resource) string {
	base := sanitizeName(resource.Name)
	if base == "" || base == "resource" {
		base = sanitizeName(resource.ID)
	}

	g.nameCount[base]++
	count := g.nameCount[base]
	if count == 1 {
		return base
	}
	return fmt.Sprintf("%s_%d", base, count)
}

// GenerateImportBlock creates a Terraform import block for a resource.
func (g *Generator) GenerateImportBlock(resource types.Resource) string {
	tfName := g.uniqueName(resource)

	return fmt.Sprintf(`import {
  to = %s.%s
  id = %q
}
`, resource.Type, tfName, resource.ID)
}

// GenerateAll creates import blocks, writes them to disk, runs terraform/tofu
// import to generate HCL, then post-processes the output.
func (g *Generator) GenerateAll(ctx context.Context, resources []types.Resource) error {
	// Reset name deduplication state for a clean generation pass
	g.nameCount = make(map[string]int)

	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write providers.tf so terraform/tofu can resolve the provider schema
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
func (g *Generator) terraformInitNeeded() bool {
	providersDir := filepath.Join(g.outputDir, ".terraform", "providers")
	info, err := os.Stat(providersDir)
	if err != nil {
		return true
	}
	return !info.IsDir()
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

// sanitizeName converts a resource ID/name into a valid Terraform resource name.
func sanitizeName(name string) string {
	var result []byte
	for _, c := range []byte(name) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_':
			result = append(result, c)
		case c == '-', c == '.', c == '/', c == ':':
			result = append(result, '_')
		}
	}
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = append([]byte{'r', '_'}, result...)
	}
	if len(result) == 0 {
		return "resource"
	}
	return string(result)
}
