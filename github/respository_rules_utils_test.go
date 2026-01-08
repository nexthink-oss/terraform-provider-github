package github

import (
	"testing"

	"github.com/google/go-github/v81/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestFlattenRulesHandlesUnknownTypes(t *testing.T) {
	// Create test rules using the correct RepositoryRulesetRules structure
	rules := &github.RepositoryRulesetRules{
		Creation: &github.EmptyRuleParameters{},
		Deletion: &github.EmptyRuleParameters{},
		// Note: Unknown rule types cannot be represented in RepositoryRulesetRules
		// as it uses a struct with specific fields, not a flexible array
	}

	// This should not panic or fail
	result := flattenRules(rules, false)

	if len(result) != 1 {
		t.Fatalf("Expected 1 element in result, got %d", len(result))
	}

	rulesMap := result[0].(map[string]any)

	// Should contain the known rules
	if !rulesMap["creation"].(bool) {
		t.Error("Expected creation rule to be true")
	}

	if !rulesMap["deletion"].(bool) {
		t.Error("Expected deletion rule to be true")
	}
}

func TestFlattenRulesHandlesMaxFileSize(t *testing.T) {
	// Test that max_file_size rule is properly handled
	maxFileSize := int64(50) // 50 MB

	rules := &github.RepositoryRulesetRules{
		MaxFileSize: &github.MaxFileSizeRuleParameters{
			MaxFileSize: maxFileSize,
		},
	}

	result := flattenRules(rules, false)

	if len(result) != 1 {
		t.Fatalf("Expected 1 element in result, got %d", len(result))
	}

	rulesMap := result[0].(map[string]any)
	maxFileSizeRules := rulesMap["max_file_size"].([]map[string]any)

	if len(maxFileSizeRules) != 1 {
		t.Fatalf("Expected 1 max_file_size rule, got %d", len(maxFileSizeRules))
	}

	if maxFileSizeRules[0]["max_file_size"] != maxFileSize {
		t.Errorf("Expected max_file_size to be %d, got %v", maxFileSize, maxFileSizeRules[0]["max_file_size"])
	}
}

func TestFlattenRulesHandlesFileExtensionRestriction(t *testing.T) {
	// Test that file_extension_restriction rule is properly handled
	restrictedExtensions := []string{".exe", ".bat", ".com"}

	rules := &github.RepositoryRulesetRules{
		FileExtensionRestriction: &github.FileExtensionRestrictionRuleParameters{
			RestrictedFileExtensions: restrictedExtensions,
		},
	}

	result := flattenRules(rules, false)

	if len(result) != 1 {
		t.Fatalf("Expected 1 element in result, got %d", len(result))
	}

	rulesMap := result[0].(map[string]any)
	fileExtensionRules := rulesMap["file_extension_restriction"].([]map[string]any)

	if len(fileExtensionRules) != 1 {
		t.Fatalf("Expected 1 file_extension_restriction rule, got %d", len(fileExtensionRules))
	}

	actualExtensions := fileExtensionRules[0]["restricted_file_extensions"].([]string)
	if len(actualExtensions) != len(restrictedExtensions) {
		t.Errorf("Expected %d restricted extensions, got %d", len(restrictedExtensions), len(actualExtensions))
	}

	for i, ext := range restrictedExtensions {
		if actualExtensions[i] != ext {
			t.Errorf("Expected extension %s at index %d, got %s", ext, i, actualExtensions[i])
		}
	}
}

func TestFlattenRulesHandlesMaxFilePathLength(t *testing.T) {
	// Test that max_file_path_length rule is properly handled
	maxPathLength := 256

	rules := &github.RepositoryRulesetRules{
		MaxFilePathLength: &github.MaxFilePathLengthRuleParameters{
			MaxFilePathLength: maxPathLength,
		},
	}

	result := flattenRules(rules, false)

	if len(result) != 1 {
		t.Fatalf("Expected 1 element in result, got %d", len(result))
	}

	rulesMap := result[0].(map[string]any)
	maxFilePathLengthRules := rulesMap["max_file_path_length"].([]map[string]any)

	if len(maxFilePathLengthRules) != 1 {
		t.Fatalf("Expected 1 max_file_path_length rule, got %d", len(maxFilePathLengthRules))
	}

	if maxFilePathLengthRules[0]["max_file_path_length"] != maxPathLength {
		t.Errorf("Expected max_file_path_length to be %d, got %v", maxPathLength, maxFilePathLengthRules[0]["max_file_path_length"])
	}
}

func TestExpandRulesHandlesMaxFilePathLength(t *testing.T) {
	// Test that max_file_path_length rule is properly expanded
	maxPathLength := 512

	rulesMap := map[string]any{
		"max_file_path_length": []any{
			map[string]any{
				"max_file_path_length": maxPathLength,
			},
		},
	}

	input := []any{rulesMap}
	result := expandRules(input, false)

	if result == nil {
		t.Fatal("Expected non-nil result from expandRules")
	}

	if result.MaxFilePathLength == nil {
		t.Fatal("Expected MaxFilePathLength to be set")
	}

	if result.MaxFilePathLength.MaxFilePathLength != maxPathLength {
		t.Errorf("Expected MaxFilePathLength to be %d, got %d", maxPathLength, result.MaxFilePathLength.MaxFilePathLength)
	}
}

func TestRoundTripMaxFilePathLength(t *testing.T) {
	// Test that max_file_path_length rule survives expand -> flatten round trip
	maxPathLength := 1024

	// Start with terraform configuration
	rulesMap := map[string]any{
		"max_file_path_length": []any{
			map[string]any{
				"max_file_path_length": maxPathLength,
			},
		},
	}

	input := []any{rulesMap}

	// Expand to GitHub API format
	expandedRules := expandRules(input, false)

	if expandedRules == nil {
		t.Fatal("Expected non-nil expanded rules")
	}

	if expandedRules.MaxFilePathLength == nil {
		t.Fatal("Expected MaxFilePathLength to be set after expand")
	}

	// Flatten back to terraform format
	flattenedResult := flattenRules(expandedRules, false)

	if len(flattenedResult) != 1 {
		t.Fatalf("Expected 1 flattened result, got %d", len(flattenedResult))
	}

	flattenedRulesMap := flattenedResult[0].(map[string]any)
	maxFilePathLengthRules := flattenedRulesMap["max_file_path_length"].([]map[string]any)

	if len(maxFilePathLengthRules) != 1 {
		t.Fatalf("Expected 1 max_file_path_length rule after round trip, got %d", len(maxFilePathLengthRules))
	}

	if maxFilePathLengthRules[0]["max_file_path_length"] != maxPathLength {
		t.Errorf("Expected max_file_path_length to be %d after round trip, got %v", maxPathLength, maxFilePathLengthRules[0]["max_file_path_length"])
	}
}

func TestMaxFilePathLengthWithOtherRules(t *testing.T) {
	// Test that max_file_path_length works correctly alongside other rules
	maxPathLength := 200

	rulesMap := map[string]any{
		"creation": true,
		"deletion": true,
		"max_file_path_length": []any{
			map[string]any{
				"max_file_path_length": maxPathLength,
			},
		},
		"max_file_size": []any{
			map[string]any{
				"max_file_size": 1, // 1 MB
			},
		},
	}

	input := []any{rulesMap}

	// Expand to GitHub API format
	expandedRules := expandRules(input, false)

	if expandedRules == nil {
		t.Fatal("Expected non-nil expanded rules")
	}

	// Verify we have all expected rule types
	ruleCount := 0
	if expandedRules.Creation != nil {
		ruleCount++
	}
	if expandedRules.Deletion != nil {
		ruleCount++
	}
	if expandedRules.MaxFilePathLength != nil {
		ruleCount++
	}
	if expandedRules.MaxFileSize != nil {
		ruleCount++
	}

	if ruleCount != 4 {
		t.Fatalf("Expected 4 expanded rules, got %d", ruleCount)
	}

	// Flatten back and verify
	flattenedResult := flattenRules(expandedRules, false)
	flattenedRulesMap := flattenedResult[0].(map[string]any)

	// Check that all rules are preserved
	if !flattenedRulesMap["creation"].(bool) {
		t.Error("Expected creation rule to be true")
	}

	if !flattenedRulesMap["deletion"].(bool) {
		t.Error("Expected deletion rule to be true")
	}

	maxFilePathLengthRules := flattenedRulesMap["max_file_path_length"].([]map[string]any)
	if len(maxFilePathLengthRules) != 1 || maxFilePathLengthRules[0]["max_file_path_length"] != maxPathLength {
		t.Errorf("Expected max_file_path_length rule with value %d", maxPathLength)
	}

	maxFileSizeRules := flattenedRulesMap["max_file_size"].([]map[string]any)
	if len(maxFileSizeRules) != 1 || maxFileSizeRules[0]["max_file_size"] != int64(1) {
		t.Error("Expected max_file_size rule with value 1")
	}
}

func TestMaxFilePathLengthErrorHandling(t *testing.T) {
	// Test that zero-value max_file_path_length parameters are handled gracefully
	rules := &github.RepositoryRulesetRules{
		MaxFilePathLength: &github.MaxFilePathLengthRuleParameters{
			MaxFilePathLength: 0,
		},
	}

	// This should not panic, even with zero value
	result := flattenRules(rules, false)

	if len(result) != 1 {
		t.Fatalf("Expected 1 element in result, got %d", len(result))
	}

	rulesMap := result[0].(map[string]any)
	maxFilePathLengthRules, exists := rulesMap["max_file_path_length"]

	if !exists {
		t.Error("Expected max_file_path_length rule to be present even with zero value")
	}

	// The rule should be present with the zero value
	rules_slice := maxFilePathLengthRules.([]map[string]any)
	if len(rules_slice) != 1 {
		t.Errorf("Expected 1 max_file_path_length rule, got %d", len(rules_slice))
	}

	if rules_slice[0]["max_file_path_length"] != 0 {
		t.Errorf("Expected max_file_path_length to be 0, got %v", rules_slice[0]["max_file_path_length"])
	}
}

func TestCompletePushRulesetSupport(t *testing.T) {
	// Test that all push-specific rules are supported together

	// Create a Set for restricted_file_extensions
	restrictedExtensions := []any{".exe", ".bat", ".sh"}
	restrictedExtensionsSet := schema.NewSet(schema.HashString, restrictedExtensions)

	rulesMap := map[string]any{
		"file_path_restriction": []any{
			map[string]any{
				"restricted_file_paths": []any{"secrets/", "*.key", "private/"},
			},
		},
		"max_file_size": []any{
			map[string]any{
				"max_file_size": 5, // 5 MB
			},
		},
		"max_file_path_length": []any{
			map[string]any{
				"max_file_path_length": 300,
			},
		},
		"file_extension_restriction": []any{
			map[string]any{
				"restricted_file_extensions": restrictedExtensionsSet,
			},
		},
	}

	input := []any{rulesMap}

	// Expand to GitHub API format
	expandedRules := expandRules(input, false)

	if expandedRules == nil {
		t.Fatal("Expected non-nil expanded rules")
	}

	// Verify we have all expected push rule types
	ruleCount := 0
	if expandedRules.FilePathRestriction != nil {
		ruleCount++
	}
	if expandedRules.MaxFileSize != nil {
		ruleCount++
	}
	if expandedRules.MaxFilePathLength != nil {
		ruleCount++
	}
	if expandedRules.FileExtensionRestriction != nil {
		ruleCount++
	}

	if ruleCount != 4 {
		t.Fatalf("Expected 4 expanded rules for complete push ruleset, got %d", ruleCount)
	}

	// Flatten back to terraform format
	flattenedResult := flattenRules(expandedRules, false)

	if len(flattenedResult) != 1 {
		t.Fatalf("Expected 1 flattened result, got %d", len(flattenedResult))
	}

	flattenedRulesMap := flattenedResult[0].(map[string]any)

	// Verify file_path_restriction
	filePathRules := flattenedRulesMap["file_path_restriction"].([]map[string]any)
	if len(filePathRules) != 1 {
		t.Fatalf("Expected 1 file_path_restriction rule, got %d", len(filePathRules))
	}
	restrictedPaths := filePathRules[0]["restricted_file_paths"].([]string)
	if len(restrictedPaths) != 3 {
		t.Errorf("Expected 3 restricted file paths, got %d", len(restrictedPaths))
	}

	// Verify max_file_size
	maxFileSizeRules := flattenedRulesMap["max_file_size"].([]map[string]any)
	if len(maxFileSizeRules) != 1 {
		t.Fatalf("Expected 1 max_file_size rule, got %d", len(maxFileSizeRules))
	}
	if maxFileSizeRules[0]["max_file_size"] != int64(5) {
		t.Errorf("Expected max_file_size to be 5, got %v", maxFileSizeRules[0]["max_file_size"])
	}

	// Verify max_file_path_length
	maxFilePathLengthRules := flattenedRulesMap["max_file_path_length"].([]map[string]any)
	if len(maxFilePathLengthRules) != 1 {
		t.Fatalf("Expected 1 max_file_path_length rule, got %d", len(maxFilePathLengthRules))
	}
	if maxFilePathLengthRules[0]["max_file_path_length"] != 300 {
		t.Errorf("Expected max_file_path_length to be 300, got %v", maxFilePathLengthRules[0]["max_file_path_length"])
	}

	// Verify file_extension_restriction
	fileExtRules := flattenedRulesMap["file_extension_restriction"].([]map[string]any)
	if len(fileExtRules) != 1 {
		t.Fatalf("Expected 1 file_extension_restriction rule, got %d", len(fileExtRules))
	}
	restrictedExts := fileExtRules[0]["restricted_file_extensions"].([]string)
	if len(restrictedExts) != 3 {
		t.Errorf("Expected 3 restricted file extensions, got %d", len(restrictedExts))
	}
}

func TestAllPushRulesWithOtherRules(t *testing.T) {
	// Test that push rules work correctly even when other standard rules are present
	maxPathLength := 100

	rules := &github.RepositoryRulesetRules{
		Creation: &github.EmptyRuleParameters{},
		MaxFilePathLength: &github.MaxFilePathLengthRuleParameters{
			MaxFilePathLength: maxPathLength,
		},
	}

	result := flattenRules(rules, false)

	if len(result) != 1 {
		t.Fatalf("Expected 1 element in result, got %d", len(result))
	}

	rulesMap := result[0].(map[string]any)

	// Should contain the known rules
	if !rulesMap["creation"].(bool) {
		t.Error("Expected creation rule to be true")
	}

	maxFilePathLengthRules := rulesMap["max_file_path_length"].([]map[string]any)
	if len(maxFilePathLengthRules) != 1 {
		t.Fatalf("Expected 1 max_file_path_length rule, got %d", len(maxFilePathLengthRules))
	}

	if maxFilePathLengthRules[0]["max_file_path_length"] != maxPathLength {
		t.Errorf("Expected max_file_path_length to be %d, got %v", maxPathLength, maxFilePathLengthRules[0]["max_file_path_length"])
	}
}
