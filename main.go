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

	"github.com/nexthink-oss/terraform-provider-github/v7/github"
)

func main() {
	ctx := context.Background()

	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	// Upgrade SDKv2 provider (Protocol 5) to Protocol 6
	upgradedSdkServer, err := tf5to6server.UpgradeServer(
		ctx,
		github.Provider().GRPCProvider,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Combine Framework (Protocol 6) and upgraded SDKv2 providers
	providers := []func() tfprotov6.ProviderServer{
		providerserver.NewProtocol6(github.NewFrameworkProvider()()),
		func() tfprotov6.ProviderServer { return upgradedSdkServer },
	}

	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf6server.ServeOpt
	if debug {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}

	err = tf6server.Serve(
		"registry.terraform.io/nexthink-oss/github",
		muxServer.ProviderServer,
		serveOpts...,
	)
	if err != nil {
		log.Fatal(err)
	}
}
