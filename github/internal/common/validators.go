package common

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// SecretNameValidator validates secret names according to GitHub requirements
type SecretNameValidator struct{}

func (v *SecretNameValidator) Description(ctx context.Context) string {
	return "Secret names can only contain alphanumeric characters or underscores and must not start with a number or GITHUB_ prefix"
}

func (v *SecretNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *SecretNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// https://docs.github.com/en/actions/reference/encrypted-secrets#naming-your-secrets
	secretNameRegexp := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

	if !secretNameRegexp.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Secret Name",
			"Secret names can only contain alphanumeric characters or underscores and must not start with a number",
		)
	}

	if strings.HasPrefix(strings.ToUpper(value), "GITHUB_") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Secret Name",
			"Secret names must not start with GITHUB_",
		)
	}
}

// NewSecretNameValidator creates a new SecretNameValidator
func NewSecretNameValidator() validator.String {
	return &SecretNameValidator{}
}

// ConflictingWithValidator implements validation for conflicting attributes
type ConflictingWithValidator struct {
	ConflictsWith []string
}

func (v *ConflictingWithValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("Conflicts with: %v", v.ConflictsWith)
}

func (v *ConflictingWithValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *ConflictingWithValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for _, conflictingAttribute := range v.ConflictsWith {
		conflictingPath := req.Path.ParentPath().AtName(conflictingAttribute)

		var conflictingValue string
		diags := req.Config.GetAttribute(ctx, conflictingPath, &conflictingValue)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			continue
		}

		if conflictingValue != "" {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Conflicting Attribute Configuration",
				fmt.Sprintf("Cannot set %q when %q is set", req.Path.String(), conflictingPath.String()),
			)
		}
	}
}

// NewConflictingWithValidator creates a new ConflictingWithValidator
func NewConflictingWithValidator(conflictsWith []string) validator.String {
	return &ConflictingWithValidator{
		ConflictsWith: conflictsWith,
	}
}
