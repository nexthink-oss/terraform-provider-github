package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/nexthink-oss/terraform-provider-github/v7/github"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: github.Provider})
}
