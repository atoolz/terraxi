package codegen

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// blockInfo holds metadata about a resource block for collapse analysis.
type blockInfo struct {
	block      *hclwrite.Block
	label      string
	attrKeys   string
	attrValues map[string]string
}

// CollapseForEach detects groups of 3+ resource blocks of the same type
// with structurally identical schemas (same attributes, same nested block
// structure) but different scalar values. Converts each group into a single
// resource block using for_each with a map.
//
// Only collapses when ALL non-varying attributes are identical across the group.
// Resources with differing nested blocks are skipped.
func CollapseForEach(f *hclwrite.File) {
	byType := make(map[string][]blockInfo)

	for _, block := range f.Body().Blocks() {
		if block.Type() != "resource" || len(block.Labels()) < 2 {
			continue
		}
		resourceType := block.Labels()[0]
		resourceName := block.Labels()[1]

		// Skip resources with nested blocks (too complex for for_each collapse)
		if len(block.Body().Blocks()) > 0 {
			continue
		}

		attrs := block.Body().Attributes()
		keys := make([]string, 0, len(attrs))
		values := make(map[string]string, len(attrs))
		for name, attr := range attrs {
			keys = append(keys, name)
			values[name] = string(attr.Expr().BuildTokens(nil).Bytes())
		}
		sort.Strings(keys)

		byType[resourceType] = append(byType[resourceType], blockInfo{
			block:      block,
			label:      resourceName,
			attrKeys:   strings.Join(keys, ","),
			attrValues: values,
		})
	}

	for resourceType, blocks := range byType {
		if len(blocks) < 3 {
			continue
		}

		// Sub-group by structural signature (same attribute names)
		bySignature := make(map[string][]blockInfo)
		for _, bi := range blocks {
			bySignature[bi.attrKeys] = append(bySignature[bi.attrKeys], bi)
		}

		for sig, group := range bySignature {
			if len(group) < 3 {
				continue
			}

			// Find which attributes vary vs which are constant
			keys := strings.Split(sig, ",")
			varying := findVaryingAttrs(group, keys)

			if len(varying) == 0 {
				// All attributes identical, nothing to parameterize
				continue
			}

			slog.Debug("Collapsing resources into for_each",
				"type", resourceType,
				"count", len(group),
				"varying", varying,
			)

			// Build the for_each map
			foreachBlock := buildForEachBlock(resourceType, group, keys, varying)
			if foreachBlock == nil {
				continue
			}

			// Remove original blocks and add the collapsed one
			for _, bi := range group {
				f.Body().RemoveBlock(bi.block)
			}
			f.Body().AppendNewline()
			f.Body().AppendBlock(foreachBlock)
		}
	}
}

// findVaryingAttrs returns attribute names whose values differ across the group.
func findVaryingAttrs(group []blockInfo, keys []string) []string {
	var varying []string
	for _, key := range keys {
		firstVal := group[0].attrValues[key]
		for _, bi := range group[1:] {
			if bi.attrValues[key] != firstVal {
				varying = append(varying, key)
				break
			}
		}
	}
	return varying
}

// buildForEachBlock creates a new resource block with for_each and each.value references.
func buildForEachBlock(resourceType string, group []blockInfo, allKeys, varying []string) *hclwrite.Block {
	// Build the for_each map: { "name1" = { attr1 = val1, ... }, "name2" = { ... } }
	var mapEntries []string
	for _, bi := range group {
		var entryParts []string
		for _, v := range varying {
			entryParts = append(entryParts, fmt.Sprintf("    %s = %s", v, strings.TrimSpace(bi.attrValues[v])))
		}
		mapEntries = append(mapEntries, fmt.Sprintf("  %q = {\n%s\n  }", bi.label, strings.Join(entryParts, "\n")))
	}

	foreachMap := fmt.Sprintf("{\n%s\n}", strings.Join(mapEntries, "\n"))

	// Create the new block
	newBlock := hclwrite.NewBlock("resource", []string{resourceType, "this"})
	body := newBlock.Body()

	// Set for_each
	body.SetAttributeRaw("for_each", hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(foreachMap), SpacesBefore: 1},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
	})

	// Set constant attributes (same across all resources)
	varyingSet := make(map[string]bool, len(varying))
	for _, v := range varying {
		varyingSet[v] = true
	}

	for _, key := range allKeys {
		if varyingSet[key] {
			// Varying attribute: use each.value.key
			body.SetAttributeRaw(key, hclwrite.Tokens{
				{Type: hclsyntax.TokenIdent, Bytes: []byte(fmt.Sprintf("each.value.%s", key)), SpacesBefore: 1},
				{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
			})
		} else {
			// Constant attribute: keep the original value from the first block
			tokens := group[0].block.Body().Attributes()[key].Expr().BuildTokens(nil)
			body.SetAttributeRaw(key, tokens)
		}
	}

	return newBlock
}
