package main

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/myklst/terraform-provider-st-ucloud/ucloud"
)

// Provider documentation generation.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name st-ucloud

func main() {
	providerAddress := os.Getenv("PROVIDER_LOCAL_PATH")
	if providerAddress == "" {
		providerAddress = "registry.terraform.io/myklst/st-ucloud"
	}
	providerAddress = "example.local/myklst/st-ucloud"
	providerserver.Serve(context.Background(), ucloud.New, providerserver.ServeOpts{
		Address: providerAddress,
	})
}
