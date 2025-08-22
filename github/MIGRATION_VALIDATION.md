# GitHub Provider Plugin Framework Migration Validation Plan

## Overview

This document provides a comprehensive validation plan for testing the migration from terraform-plugin-sdk/v2 to terraform-plugin-framework. The plan covers all breaking changes that were fixed and ensures backward compatibility is maintained.

## Prerequisites

### Required Credentials
- GitHub Personal Access Token (classic) with appropriate permissions
- GitHub App credentials (App ID, Installation ID, Private Key) for app authentication testing
- Access to a GitHub organization for testing organization-specific features
- Test repositories that can be safely modified

### Required Tools
- Terraform CLI (latest version)
- Two provider binaries:
  - SDKv2 version (pre-migration)
  - Plugin Framework version (post-migration)

### Test Environment Setup
```bash
# Create test workspace
mkdir github-provider-migration-test
cd github-provider-migration-test

# Create separate directories for each test phase
mkdir sdkv2-state
mkdir framework-migration
mkdir backward-compatibility
```

## Validation Test Plan

### Phase 1: SDKv2 State Creation

**Objective**: Create Terraform state using the SDKv2 version of the provider to establish baseline for migration testing.

#### 1.1 Provider Configuration Test
```hcl
# sdkv2-state/main.tf
terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6.6" # Last SDKv2 version
    }
  }
}

provider "github" {
  token = var.github_token
  owner = var.github_owner
}

# Test variables
variable "github_token" {
  description = "GitHub personal access token"
  type        = string
  sensitive   = true
}

variable "github_owner" {
  description = "GitHub owner (username or organization)"
  type        = string
}
```

#### 1.2 Create Baseline Resources
```hcl
# sdkv2-state/resources.tf

# Repository for testing
resource "github_repository" "test" {
  name        = "terraform-migration-test"
  description = "Test repository for provider migration"
  private     = true

  auto_init = true

  has_issues   = true
  has_projects = false
  has_wiki     = false
}

# Branch protection for schema version testing
resource "github_branch_protection" "main" {
  repository_id = github_repository.test.node_id
  pattern       = "main"

  enforce_admins = true

  required_status_checks {
    strict = true
  }

  required_pull_request_reviews {
    required_approving_review_count = 1
    dismiss_stale_reviews          = true
  }
}

# Repository webhook for testing
resource "github_repository_webhook" "test" {
  repository = github_repository.test.name

  configuration {
    url          = "https://example.com/webhook"
    content_type = "json"
    insecure_ssl = false
  }

  active = true
  events = ["push", "pull_request"]
}

# Repository autolink reference
resource "github_repository_autolink_reference" "test" {
  repository = github_repository.test.name

  key_prefix   = "TICKET-"
  target_url   = "https://example.com/TICKET/<num>"
  is_alphanumeric = false
}

# Organization webhook (if testing with org)
resource "github_organization_webhook" "test" {
  count = var.is_organization ? 1 : 0

  configuration {
    url          = "https://example.com/org-webhook"
    content_type = "json"
    insecure_ssl = false
  }

  active = true
  events = ["push", "repository"]
}

variable "is_organization" {
  description = "Whether testing with an organization"
  type        = bool
  default     = false
}
```

#### 1.3 Execute SDKv2 Baseline
```bash
cd sdkv2-state
terraform init
terraform plan
terraform apply

# Verify state was created
terraform show
terraform state list

# Export state for comparison
terraform show -json > sdkv2-state.json
cp terraform.tfstate ../sdkv2-baseline.tfstate
```

### Phase 2: Plugin Framework Migration Testing

**Objective**: Test migration from SDKv2 state to Plugin Framework provider.

#### 2.1 Provider Version Migration
```hcl
# framework-migration/main.tf
terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6.0" # Plugin Framework version
    }
  }
}

# Copy the baseline state
# cp ../sdkv2-baseline.tfstate terraform.tfstate
```

#### 2.2 Migration Validation Steps
```bash
cd ../framework-migration

# Copy SDKv2 state and configuration
cp ../sdkv2-state/terraform.tfstate .
cp ../sdkv2-state/*.tf .

# Update provider version
# Edit main.tf to use Plugin Framework version

# Initialize with new provider
terraform init -upgrade

# Test state compatibility
terraform plan

# Verify no changes are required (should show "No changes")
# If changes are shown, this indicates a migration issue

# Apply to ensure state is properly migrated
terraform apply

# Compare final state with baseline
terraform show -json > framework-migrated-state.json
```

### Phase 3: Breaking Change Validation

**Objective**: Validate that all breaking changes have been properly addressed.

#### 3.1 GitHub App Authentication Testing
```hcl
# Test the restored app_auth configuration block
provider "github" {
  app_auth {
    id              = var.github_app_id
    installation_id = var.github_app_installation_id
    pem_file        = var.github_app_pem_file
  }
  owner = var.github_owner
}

variable "github_app_id" {
  description = "GitHub App ID"
  type        = string
}

variable "github_app_installation_id" {
  description = "GitHub App Installation ID"
  type        = string
}

variable "github_app_pem_file" {
  description = "Path to GitHub App private key PEM file"
  type        = string
}
```

**Validation Steps**:
```bash
# Test GitHub App authentication
terraform plan

# Should successfully authenticate and plan without errors
# Verify the provider can access resources with App authentication
```

#### 3.2 Organization Attribute Backward Compatibility
```hcl
# Test deprecated organization attribute still works
provider "github" {
  token        = var.github_token
  organization = var.github_organization # Should work but show deprecation warning
}

variable "github_organization" {
  description = "GitHub organization (deprecated, use owner instead)"
  type        = string
}
```

**Validation Steps**:
```bash
# Test organization attribute
terraform plan

# Expected:
# - Plan should succeed
# - Should show deprecation warning about using 'owner' instead
# - Should function identically to 'owner' attribute
```

#### 3.3 Schema Version Validation
Create test configurations for all resources with schema version fixes:

```hcl
# Test all resources that had schema version fixes
resource "github_branch_protection" "schema_test" {
  repository_id = github_repository.test.node_id
  pattern       = "develop"
  # ... configuration
}

resource "github_organization_ruleset" "schema_test" {
  count = var.is_organization ? 1 : 0
  name  = "test-ruleset"

  target = "branch"
  enforcement = "active"

  rules {
    creation = true
  }
}

resource "github_repository_ruleset" "schema_test" {
  repository = github_repository.test.name
  name       = "test-repo-ruleset"

  target      = "branch"
  enforcement = "active"

  rules {
    creation = true
  }
}
```

**Validation Steps**:
```bash
# Test schema version compatibility
terraform plan
terraform apply

# Import existing resources to test import functionality
terraform import github_repository.existing existing-repo-name
terraform import github_branch_protection.existing "existing-repo:main"
```

### Phase 4: Comprehensive Import Testing

**Objective**: Validate that import functionality works correctly for all resources.

#### 4.1 Resource Import Tests
```bash
# Test importing various resource types
terraform import github_repository.import_test "test-repo"
terraform import github_branch_protection.import_test "test-repo:main"
terraform import github_repository_webhook.import_test "test-repo:12345"
terraform import github_organization_webhook.import_test "12345"
terraform import github_repository_autolink_reference.import_test "test-repo:1"

# Verify imported resources show no changes
terraform plan
```

#### 4.2 Import State Validation
```bash
# After each import, verify the resource configuration matches
terraform show

# Compare imported resource attributes with expected values
# Ensure no unexpected changes are shown in subsequent plans
```

### Phase 5: Regression Testing

**Objective**: Ensure no functionality regressions were introduced.

#### 5.1 CRUD Operations Test
```bash
# Create
terraform apply

# Read (refresh state)
terraform refresh

# Update (modify configuration and apply)
# Modify some resource attributes
terraform apply

# Delete (destroy)
terraform destroy
```

#### 5.2 Error Handling Validation
```hcl
# Test invalid configurations to ensure proper error messages
resource "github_repository" "invalid" {
  name = "invalid-name-with-special-chars-!!!"
  # Should produce meaningful error message
}
```

### Phase 6: Performance and Compatibility Testing

#### 6.1 Large State Testing
```bash
# Create configuration with many resources (10-20 of each type)
# Test performance and memory usage
terraform apply

# Measure plan/apply times
time terraform plan
time terraform apply
```

#### 6.2 Concurrent Operations
```bash
# Test multiple terraform operations in different workspaces
# Ensure no race conditions or state corruption
```

## Validation Checklist

### ✅ Migration Validation
- [ ] SDKv2 state successfully migrates to Plugin Framework
- [ ] No unexpected changes after migration
- [ ] All resource attributes preserved
- [ ] State format compatibility maintained

### ✅ Breaking Change Fixes
- [ ] GitHub App authentication (`app_auth` block) works correctly
- [ ] Organization attribute backward compatibility maintained
- [ ] Deprecation warning shown for `organization` attribute
- [ ] All schema versions correctly set and working

### ✅ Resource Functionality
- [ ] All resources can be created successfully
- [ ] All resources can be imported successfully
- [ ] All resources can be updated successfully
- [ ] All resources can be destroyed successfully
- [ ] Import IDs work as documented

### ✅ Schema Version Compatibility
- [ ] `github_branch_protection` schema version v2
- [ ] `github_organization_ruleset` schema version v1
- [ ] `github_organization_webhook` schema version v1
- [ ] `github_repository` schema version v1
- [ ] `github_repository_autolink_reference` schema version v1
- [ ] `github_repository_ruleset` schema version v1
- [ ] `github_repository_webhook` schema version v1

### ✅ Error Handling
- [ ] Invalid configurations produce meaningful errors
- [ ] Network errors handled gracefully
- [ ] Authentication errors clearly reported
- [ ] Rate limiting properly handled

### ✅ Performance
- [ ] Plan/apply times within acceptable ranges
- [ ] Memory usage reasonable for large configurations
- [ ] No resource leaks or memory issues

## Success Criteria

The migration is considered successful when:

1. **State Migration**: All existing SDKv2 states can be migrated to Plugin Framework without data loss or unexpected changes
2. **Backward Compatibility**: All deprecated features continue to work with appropriate warnings
3. **Breaking Changes Resolved**: All documented breaking changes have been addressed and validated
4. **Import Functionality**: All resources can be imported using their documented import IDs
5. **Schema Compatibility**: All schema versions are correctly set and state migrations work
6. **No Regressions**: All existing functionality continues to work as expected
7. **Error Handling**: Errors are clear, actionable, and user-friendly

## Troubleshooting Guide

### Common Issues

#### State Migration Failures
- **Symptom**: Terraform shows unexpected changes after provider upgrade
- **Solution**: Check schema version compatibility and resource attribute mappings

#### Import Failures
- **Symptom**: `terraform import` commands fail
- **Solution**: Verify import ID format and resource existence in GitHub

#### Authentication Issues
- **Symptom**: API authentication errors
- **Solution**: Verify token permissions and GitHub App configuration

#### Schema Version Mismatches
- **Symptom**: State file corruption or unexpected behavior
- **Solution**: Check resource schema versions match expected values

## Reporting Issues

When validation fails, collect the following information:

1. Terraform version
2. Provider version (both SDKv2 and Plugin Framework)
3. Complete error messages and logs
4. Minimal reproduction configuration
5. GitHub API responses (if applicable)
6. State file differences (sanitized)

## Conclusion

This validation plan provides comprehensive testing for the GitHub Provider Plugin Framework migration. Execute all phases sequentially and document any issues found. The migration should only be considered complete when all validation tests pass successfully.
