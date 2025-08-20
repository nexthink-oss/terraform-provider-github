package github

import (
	"testing"
)

// TestGithubUserSshKeyResource tests the basic resource creation
func TestGithubUserSshKeyResource(t *testing.T) {
	resource := NewGithubUserSshKeyResource()
	if resource == nil {
		t.Error("Resource should not be nil")
	}
}
