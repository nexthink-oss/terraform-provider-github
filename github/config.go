package github

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type Config struct {
	Token            string
	Owner            string
	BaseURL          string
	Insecure         bool
	WriteDelay       time.Duration
	ReadDelay        time.Duration
	RetryDelay       time.Duration
	RetryableErrors  map[int]bool
	MaxRetries       int
	ParallelRequests bool
}

type Owner struct {
	name           string
	id             int64
	v3client       *github.Client
	v4client       *githubv4.Client
	StopContext    context.Context
	IsOrganization bool
}

// V3Client returns the GitHub v3 REST API client
func (o *Owner) V3Client() *github.Client {
	return o.v3client
}

// V4Client returns the GitHub v4 GraphQL API client
func (o *Owner) V4Client() *githubv4.Client {
	return o.v4client
}

// Name returns the owner name
func (o *Owner) Name() string {
	return o.name
}

// ID returns the owner ID
func (o *Owner) ID() int64 {
	return o.id
}

// GHECDataResidencyMatch is a regex to match a GitHub Enterprise Cloud data residency URL:
// https://[hostname].ghe.com instances expect paths that behave similar to GitHub.com, not GitHub Enterprise Server.
var GHECDataResidencyMatch = regexp.MustCompile(`^https:\/\/[a-zA-Z0-9.\-]*\.ghe\.com$`)

func RateLimitedHTTPClient(client *http.Client, writeDelay time.Duration, readDelay time.Duration, retryDelay time.Duration, parallelRequests bool, retryableErrors map[int]bool, maxRetries int) *http.Client {

	client.Transport = NewEtagTransport(client.Transport)
	client.Transport = NewRateLimitTransport(client.Transport, WithWriteDelay(writeDelay), WithReadDelay(readDelay), WithParallelRequests(parallelRequests))
	client.Transport = newPreviewHeaderInjectorTransport(map[string]string{
		// TODO: remove when Stone Crop preview is moved to general availability in the GraphQL API
		"Accept": "application/vnd.github.stone-crop-preview+json",
	}, client.Transport)

	if maxRetries > 0 {
		client.Transport = NewRetryTransport(client.Transport, WithRetryDelay(retryDelay), WithRetryableErrors(retryableErrors), WithMaxRetries(maxRetries))
	}

	return client
}

func (c *Config) AuthenticatedHTTPClient() *http.Client {

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.Token},
	)

	// Create base client with optional insecure TLS configuration
	var baseClient *http.Client
	if c.Insecure {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		client := oauth2.NewClient(ctx, ts)
		// Replace the transport with our insecure one while preserving oauth2 functionality
		client.Transport = &oauth2.Transport{
			Source: ts,
			Base:   transport,
		}
		baseClient = client
	} else {
		baseClient = oauth2.NewClient(ctx, ts)
	}

	return RateLimitedHTTPClient(baseClient, c.WriteDelay, c.ReadDelay, c.RetryDelay, c.ParallelRequests, c.RetryableErrors, c.MaxRetries)
}

func (c *Config) Anonymous() bool {
	return c.Token == ""
}

func (c *Config) AnonymousHTTPClient() *http.Client {
	var transport *http.Transport
	if c.Insecure {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	} else {
		transport = &http.Transport{}
	}

	client := &http.Client{Transport: transport}
	return RateLimitedHTTPClient(client, c.WriteDelay, c.ReadDelay, c.RetryDelay, c.ParallelRequests, c.RetryableErrors, c.MaxRetries)
}

func (c *Config) NewGraphQLClient(client *http.Client) (*githubv4.Client, error) {

	uv4, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}

	if uv4.String() != "https://api.github.com/" && !GHECDataResidencyMatch.MatchString(uv4.String()) {
		uv4.Path = path.Join(uv4.Path, "api/graphql/")
	} else {
		uv4.Path = path.Join(uv4.Path, "graphql")
	}

	return githubv4.NewEnterpriseClient(uv4.String(), client), nil
}

func (c *Config) NewRESTClient(client *http.Client) (*github.Client, error) {

	uv3, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}

	if uv3.String() != "https://api.github.com/" && !GHECDataResidencyMatch.MatchString(uv3.String()) {
		uv3.Path = uv3.Path + "api/v3/"
	}

	v3client, err := github.NewClient(client).WithEnterpriseURLs(uv3.String(), "")
	if err != nil {
		return nil, err
	}

	v3client.BaseURL = uv3

	return v3client, nil
}

func (c *Config) ConfigureOwner(owner *Owner) (*Owner, error) {
	ctx := context.Background()
	owner.name = c.Owner
	if owner.name == "" {
		if c.Anonymous() {
			return owner, nil
		}
		// Discover authenticated user
		user, _, err := owner.v3client.Users.Get(ctx, "")
		if err != nil {
			return nil, err
		}
		owner.name = user.GetLogin()
	} else {
		remoteOrg, _, err := owner.v3client.Organizations.Get(ctx, owner.name)
		if err == nil {
			if remoteOrg != nil {
				owner.id = remoteOrg.GetID()
				owner.IsOrganization = true
			}
		}
	}

	return owner, nil
}

// Meta returns the meta parameter that is passed into subsequent resources
func (c *Config) Meta() (any, error) {

	var client *http.Client
	if c.Anonymous() {
		client = c.AnonymousHTTPClient()
	} else {
		client = c.AuthenticatedHTTPClient()
	}

	v3client, err := c.NewRESTClient(client)
	if err != nil {
		return nil, err
	}

	v4client, err := c.NewGraphQLClient(client)
	if err != nil {
		return nil, err
	}

	var owner Owner
	owner.v4client = v4client
	owner.v3client = v3client
	owner.StopContext = context.Background()

	_, err = c.ConfigureOwner(&owner)
	if err != nil {
		return &owner, err
	}
	return &owner, nil
}

type previewHeaderInjectorTransport struct {
	rt             http.RoundTripper
	previewHeaders map[string]string
}

func newPreviewHeaderInjectorTransport(headers map[string]string, rt http.RoundTripper) *previewHeaderInjectorTransport {
	return &previewHeaderInjectorTransport{
		rt:             rt,
		previewHeaders: headers,
	}
}

func (injector *previewHeaderInjectorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for name, value := range injector.previewHeaders {
		header := req.Header.Get(name)
		if header == "" {
			header = value
		} else {
			header = strings.Join([]string{header, value}, ",")
		}
		req.Header.Set(name, header)
	}
	return injector.rt.RoundTrip(req)
}
