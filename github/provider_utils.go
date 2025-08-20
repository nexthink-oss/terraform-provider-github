package github

import (
	"fmt"
	"log"
	"os"
	"unicode"
)

var testCollaborator = os.Getenv("GITHUB_TEST_COLLABORATOR")
var isEnterprise = os.Getenv("ENTERPRISE_ACCOUNT")
var testEnterprise = os.Getenv("ENTERPRISE_SLUG")
var testOrganization = testOrganizationFunc()
var testOwner = os.Getenv("GITHUB_OWNER")
var testToken = os.Getenv("GITHUB_TOKEN")

func testAccCheckOrganization() error {

	baseURL := os.Getenv("GITHUB_BASE_URL")
	token := os.Getenv("GITHUB_TOKEN")

	owner := os.Getenv("GITHUB_OWNER")
	if owner == "" {
		organization := os.Getenv("GITHUB_ORGANIZATION")
		if organization == "" {
			return fmt.Errorf("neither `GITHUB_OWNER` or `GITHUB_ORGANIZATION` set in environment")
		}
		owner = organization
	}

	config := Config{
		BaseURL: baseURL,
		Token:   token,
		Owner:   owner,
	}

	meta, err := config.Meta()
	if err != nil {
		return err
	}
	if !meta.(*Owner).IsOrganization {
		return fmt.Errorf("configured owner %q is a user, not an organization", meta.(*Owner).name)
	}
	return nil
}

func OwnerOrOrgEnvDefaultFunc() (any, error) {
	if organization := os.Getenv("GITHUB_ORGANIZATION"); organization != "" {
		log.Printf("[INFO] Selecting owner %s from GITHUB_ORGANIZATION environment variable", organization)
		return organization, nil
	}
	owner := os.Getenv("GITHUB_OWNER")
	log.Printf("[INFO] Selecting owner %s from GITHUB_OWNER environment variable", owner)
	return owner, nil
}

func testOrganizationFunc() string {
	organization := os.Getenv("GITHUB_ORGANIZATION")
	if organization == "" {
		organization = os.Getenv("GITHUB_TEST_ORGANIZATION")
	}
	return organization
}

func testOwnerFunc() string {
	owner := os.Getenv("GITHUB_OWNER")
	if owner == "" {
		owner = os.Getenv("GITHUB_TEST_OWNER")
	}
	return owner
}

func flipUsernameCase(username string) string {
	oc := []rune(username)

	for i, ch := range oc {
		if unicode.IsLetter(ch) {
			if unicode.IsUpper(ch) {
				oc[i] = unicode.ToLower(ch)
			} else {
				oc[i] = unicode.ToUpper(ch)
			}
			break
		}
	}
	return string(oc)
}
