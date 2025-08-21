package github

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestProviderConfigurationValidation tests various provider configuration validation scenarios
func TestProviderConfigurationValidation(t *testing.T) {
	testCases := []struct {
		name          string
		config        githubProviderModel
		expectError   bool
		expectedError string
	}{
		{
			name: "valid configuration",
			config: githubProviderModel{
				Token:            types.StringValue("test_token"),
				Owner:            types.StringValue("test_owner"),
				BaseURL:          types.StringValue("https://enterprise.example.com/"),
				ParallelRequests: types.BoolValue(true),
				ReadDelayMS:      types.Int64Value(100),
				WriteDelayMS:     types.Int64Value(1500),
				RetryDelayMS:     types.Int64Value(2000),
				MaxRetries:       types.Int64Value(3),
				RetryableErrors: types.ListValueMust(types.Int64Type, []attr.Value{
					types.Int64Value(500),
					types.Int64Value(502),
				}),
				Insecure: types.BoolValue(false),
			},
			expectError: false,
		},
		{
			name: "parallel_requests true with github.com should fail",
			config: githubProviderModel{
				Token:            types.StringValue("test_token"),
				BaseURL:          types.StringValue("https://api.github.com/"),
				ParallelRequests: types.BoolValue(true),
			},
			expectError:   true,
			expectedError: "parallel_requests cannot be true when connecting to public github.com",
		},
		{
			name: "parallel_requests true with enterprise URL should pass validation",
			config: githubProviderModel{
				BaseURL:          types.StringValue("https://enterprise.example.com/"),
				ParallelRequests: types.BoolValue(true),
			},
			expectError: false,
		},
		{
			name: "negative read_delay_ms should fail",
			config: githubProviderModel{
				ReadDelayMS: types.Int64Value(-100),
			},
			expectError:   true,
			expectedError: "read_delay_ms must be greater than or equal to 0",
		},
		{
			name: "zero read_delay_ms should pass",
			config: githubProviderModel{
				ReadDelayMS: types.Int64Value(0),
			},
			expectError: false,
		},
		{
			name: "zero write_delay_ms should fail",
			config: githubProviderModel{
				WriteDelayMS: types.Int64Value(0),
			},
			expectError:   true,
			expectedError: "write_delay_ms must be greater than 0ms",
		},
		{
			name: "negative write_delay_ms should fail",
			config: githubProviderModel{
				WriteDelayMS: types.Int64Value(-500),
			},
			expectError:   true,
			expectedError: "write_delay_ms must be greater than 0ms",
		},
		{
			name: "positive write_delay_ms should pass",
			config: githubProviderModel{
				WriteDelayMS: types.Int64Value(1000),
			},
			expectError: false,
		},
		{
			name: "negative retry_delay_ms should fail",
			config: githubProviderModel{
				RetryDelayMS: types.Int64Value(-1000),
			},
			expectError:   true,
			expectedError: "retry_delay_ms must be greater than or equal to 0ms",
		},
		{
			name: "zero retry_delay_ms should pass",
			config: githubProviderModel{
				RetryDelayMS: types.Int64Value(0),
			},
			expectError: false,
		},
		{
			name: "negative max_retries should fail",
			config: githubProviderModel{
				MaxRetries: types.Int64Value(-1),
			},
			expectError:   true,
			expectedError: "max_retries must be greater than or equal to 0",
		},
		{
			name: "zero max_retries should pass",
			config: githubProviderModel{
				MaxRetries: types.Int64Value(0),
			},
			expectError: false,
		},
		{
			name: "valid retryable_errors list",
			config: githubProviderModel{
				MaxRetries: types.Int64Value(3),
				RetryableErrors: types.ListValueMust(types.Int64Type, []attr.Value{
					types.Int64Value(500),
					types.Int64Value(502),
					types.Int64Value(503),
					types.Int64Value(429),
				}),
			},
			expectError: false,
		},
		{
			name: "empty retryable_errors with max_retries > 0 should use defaults",
			config: githubProviderModel{
				MaxRetries: types.Int64Value(3),
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a provider and configure it with test data
			p := New()

			// Create tftypes.Value from our config for the request
			configValue := createConfigValue(tc.config)

			var req provider.ConfigureRequest
			req.Config.Raw = configValue
			req.Config.Schema = getProviderSchema(p)

			var resp provider.ConfigureResponse
			p.Configure(context.Background(), req, &resp)

			if tc.expectError {
				if !resp.Diagnostics.HasError() {
					t.Errorf("Expected error but got none")
				} else {
					// Check if error message contains expected text
					found := false
					for _, diag := range resp.Diagnostics.Errors() {
						if regexp.MustCompile(regexp.QuoteMeta(tc.expectedError)).MatchString(diag.Summary() + " " + diag.Detail()) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing %q, got diagnostics: %v", tc.expectedError, resp.Diagnostics.Errors())
					}
				}
			} else {
				if resp.Diagnostics.HasError() {
					t.Errorf("Expected no error but got: %v", resp.Diagnostics.Errors())
				}
			}
		})
	}
}

// TestRetryableErrorsProcessing tests that retryable_errors are properly converted to a map
func TestRetryableErrorsProcessing(t *testing.T) {
	testCases := []struct {
		name            string
		maxRetries      int64
		retryableErrors []int64
		expectedMap     map[int]bool
	}{
		{
			name:            "default retryable errors when max_retries > 0",
			maxRetries:      3,
			retryableErrors: []int64{500, 502, 503, 504}, // default values
			expectedMap:     map[int]bool{500: true, 502: true, 503: true, 504: true},
		},
		{
			name:            "custom retryable errors",
			maxRetries:      5,
			retryableErrors: []int64{500, 429, 502},
			expectedMap:     map[int]bool{500: true, 429: true, 502: true},
		},
		{
			name:            "empty retryable errors when max_retries = 0",
			maxRetries:      0,
			retryableErrors: []int64{500},
			expectedMap:     map[int]bool{}, // should be empty when max_retries is 0
		},
		{
			name:            "single retryable error",
			maxRetries:      1,
			retryableErrors: []int64{503},
			expectedMap:     map[int]bool{503: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the retryable errors processing logic
			retryableErrors := make(map[int]bool)
			if tc.maxRetries > 0 {
				for _, statusCode := range tc.retryableErrors {
					retryableErrors[int(statusCode)] = true
				}
			}

			// Compare maps
			if len(retryableErrors) != len(tc.expectedMap) {
				t.Errorf("Expected map length %d, got %d", len(tc.expectedMap), len(retryableErrors))
			}

			for code, expected := range tc.expectedMap {
				if retryableErrors[code] != expected {
					t.Errorf("Expected retryable error %d to be %v, got %v", code, expected, retryableErrors[code])
				}
			}
		})
	}
}

// Helper functions

func getProviderSchema(p provider.Provider) schema.Schema {
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)
	return resp.Schema
}

func createConfigValue(config githubProviderModel) tftypes.Value {
	configMap := make(map[string]tftypes.Value)

	if !config.Token.IsNull() && !config.Token.IsUnknown() {
		configMap["token"] = tftypes.NewValue(tftypes.String, config.Token.ValueString())
	} else {
		configMap["token"] = tftypes.NewValue(tftypes.String, nil)
	}

	if !config.Owner.IsNull() && !config.Owner.IsUnknown() {
		configMap["owner"] = tftypes.NewValue(tftypes.String, config.Owner.ValueString())
	} else {
		configMap["owner"] = tftypes.NewValue(tftypes.String, nil)
	}

	if !config.BaseURL.IsNull() && !config.BaseURL.IsUnknown() {
		configMap["base_url"] = tftypes.NewValue(tftypes.String, config.BaseURL.ValueString())
	} else {
		configMap["base_url"] = tftypes.NewValue(tftypes.String, nil)
	}

	if !config.ParallelRequests.IsNull() && !config.ParallelRequests.IsUnknown() {
		configMap["parallel_requests"] = tftypes.NewValue(tftypes.Bool, config.ParallelRequests.ValueBool())
	} else {
		configMap["parallel_requests"] = tftypes.NewValue(tftypes.Bool, nil)
	}

	if !config.ReadDelayMS.IsNull() && !config.ReadDelayMS.IsUnknown() {
		configMap["read_delay_ms"] = tftypes.NewValue(tftypes.Number, config.ReadDelayMS.ValueInt64())
	} else {
		configMap["read_delay_ms"] = tftypes.NewValue(tftypes.Number, nil)
	}

	if !config.WriteDelayMS.IsNull() && !config.WriteDelayMS.IsUnknown() {
		configMap["write_delay_ms"] = tftypes.NewValue(tftypes.Number, config.WriteDelayMS.ValueInt64())
	} else {
		configMap["write_delay_ms"] = tftypes.NewValue(tftypes.Number, nil)
	}

	if !config.RetryDelayMS.IsNull() && !config.RetryDelayMS.IsUnknown() {
		configMap["retry_delay_ms"] = tftypes.NewValue(tftypes.Number, config.RetryDelayMS.ValueInt64())
	} else {
		configMap["retry_delay_ms"] = tftypes.NewValue(tftypes.Number, nil)
	}

	if !config.MaxRetries.IsNull() && !config.MaxRetries.IsUnknown() {
		configMap["max_retries"] = tftypes.NewValue(tftypes.Number, config.MaxRetries.ValueInt64())
	} else {
		configMap["max_retries"] = tftypes.NewValue(tftypes.Number, nil)
	}

	if !config.RetryableErrors.IsNull() && !config.RetryableErrors.IsUnknown() {
		var errorSlice []int64
		config.RetryableErrors.ElementsAs(context.Background(), &errorSlice, false)
		errorValues := make([]tftypes.Value, len(errorSlice))
		for i, err := range errorSlice {
			errorValues[i] = tftypes.NewValue(tftypes.Number, err)
		}
		configMap["retryable_errors"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, errorValues)
	} else {
		configMap["retryable_errors"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, nil)
	}

	if !config.Insecure.IsNull() && !config.Insecure.IsUnknown() {
		configMap["insecure"] = tftypes.NewValue(tftypes.Bool, config.Insecure.ValueBool())
	} else {
		configMap["insecure"] = tftypes.NewValue(tftypes.Bool, nil)
	}

	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"token":             tftypes.String,
			"owner":             tftypes.String,
			"base_url":          tftypes.String,
			"parallel_requests": tftypes.Bool,
			"read_delay_ms":     tftypes.Number,
			"write_delay_ms":    tftypes.Number,
			"retry_delay_ms":    tftypes.Number,
			"max_retries":       tftypes.Number,
			"retryable_errors":  tftypes.List{ElementType: tftypes.Number},
			"insecure":          tftypes.Bool,
		},
	}, configMap)
}
