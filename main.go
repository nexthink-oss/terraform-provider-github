package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"

	"github.com/isometry/terraform-provider-github/v7/framework"
	"github.com/isometry/terraform-provider-github/v7/github"
)

func main() {
	ctx := context.Background()
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers")
	flag.Parse()

	// Upgrade existing SDKv2 provider to protocol v6
	upgradedSdkProvider, err := tf5to6server.UpgradeServer(
		ctx,
		github.Provider().GRPCProvider,
	)
	if err != nil {
		log.Fatal(err)
	}

	providers := []func() tfprotov6.ProviderServer{
		// Existing SDKv2 provider (upgraded to v6 protocol)
		func() tfprotov6.ProviderServer {
			return upgradedSdkProvider
		},
		// New Plugin Framework provider (will be empty initially)
		providerserver.NewProtocol6(framework.New()),
	}

	// Mux the providers together
	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf6server.ServeOpt
	if debug {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}

	err = tf6server.Serve("registry.terraform.io/isometry/github", muxServer.ProviderServer, serveOpts...)
	if err != nil {
		log.Fatal(err)
	}
}
