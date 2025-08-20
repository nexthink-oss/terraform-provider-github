package framework

import (
	"testing"
)

// TestGithubIssueLabelResource tests the basic resource creation
func TestGithubIssueLabelResource(t *testing.T) {
	resource := NewGithubIssueLabelResource()
	if resource == nil {
		t.Error("Resource should not be nil")
	}
}
