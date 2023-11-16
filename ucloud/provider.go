package ucloud

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ucloud/ucloud-sdk-go/services/ucdn"
	"github.com/ucloud/ucloud-sdk-go/ucloud"
	"github.com/ucloud/ucloud-sdk-go/ucloud/auth"
)

type ucloudProviderModel struct {
	APIUrl     types.String `tfsdk:"api_url"`
	PrivateKey types.String `tfsdk:"private_key"`
	PublicKey  types.String `tfsdk:"public_key"`
	ProjectId  types.String `tfsdk:"project_id"`
	Region     types.String `tfsdk:"region"`
	Zone       types.String `tfsdk:"zone"`
}

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &ucloudProvider{}
)

// Wrapper of Ucloud client
type ucloudClients struct {
	cdnClient *ucdn.UCDNClient
}

type ucloudProvider struct{}

func New() provider.Provider {
	return &ucloudProvider{}
}

// Metadata returns the provider type name.
func (p *ucloudProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "st-ucloud"
}

// Schema defines the provider-level schema for configuration data.
func (p *ucloudProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Ucloud provider is used to interact with the many resources supported by Ucloud. " +
			"The provider needs to be configured with the proper credentials before it can be used.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Description: "",
				Optional:    true,
			},
			"public_key": schema.StringAttribute{
				Description: "Public key for Ucloud API. May also be provided via UCLOUD_PUBLIC_KEY environment variable",
				Optional:    true,
			},
			"private_key": schema.StringAttribute{
				Description: "Secret key for Ucloud API. May also be provided via UCLOUD_SECRET_KEY environment variable",
				Optional:    true,
				Sensitive:   true,
			},
			"project_id": schema.StringAttribute{
				Description: "Project id should not be empty if public_key/private_key belongs to sub-account",
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: "Ucloud region",
				Optional:    true,
			},
			"zone": schema.StringAttribute{
				Description: "Ucloud zone",
				Optional:    true,
			},
		},
	}
}

// Configure prepares a Ucloud API client for data sources and resources.
func (p *ucloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var model ucloudProviderModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.
	if model.APIUrl.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Unknown Ucloud API Url",
			"The provider cannot create the Ucloud API client as there is an unknown configuration value for the"+
				"Ucloud API url. Set the value statically in the configuration, or use the UCLOUD_API_URL environment variable.",
		)
	}

	if model.Region.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("region"),
			"Unknown Ucloud Region",
			"The provider cannot create the Ucloud API client as there is an unknown configuration value for the"+
				"Ucloud Region. Set the value statically in the configuration, or use the UCLOUD_REGION environment variable.",
		)
	}

	if model.Zone.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("zone"),
			"Unknown Ucloud Zone",
			"The provider cannot create the Ucloud API client as there is an unknown configuration value for the"+
				"Ucloud Zone. Set the value statically in the configuration, or use the UCLOUD_ZONE environment variable.",
		)
	}

	if model.ProjectId.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("project_id"),
			"Unknown Ucloud ProjectId",
			"The provider cannot create the Ucloud API client as there is an unknown configuration value for the"+
				"Ucloud ProjectId. Set the value statically in the configuration, or use the UCLOUD_PROJECT_ID environment variable.",
		)
	}

	if model.PublicKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("public_key"),
			"Unknown Ucloud PublicKey",
			"The provider cannot create the Ucloud API client as there is an unknown configuration value for the"+
				"Ucloud PublicKey. Set the value statically in the configuration, or use the UCLOUD_PUBLIC_KEY environment variable.",
		)
	}

	if model.PrivateKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("private_key"),
			"Unknown Ucloud PrivateKey",
			"The provider cannot create the Ucloud API client as there is an unknown configuration value for the"+
				"Ucloud PrivateKey. Set the value statically in the configuration, or use the UCLOUD_PRIVATE_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var (
		apiUrl,
		region,
		zone,
		projectId,
		publicKey,
		privateKey string
	)

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	if !model.APIUrl.IsNull() {
		apiUrl = model.APIUrl.ValueString()
	} else {
		apiUrl = os.Getenv("UCLOUD_API_URL")
	}

	if !model.Region.IsNull() {
		region = model.Region.ValueString()
	} else {
		region = os.Getenv("UCLOUD_REGION")
	}

	if !model.Zone.IsNull() {
		zone = model.Zone.ValueString()
	} else {
		zone = os.Getenv("UCLOUD_ZONE")
	}

	if !model.ProjectId.IsNull() {
		projectId = model.ProjectId.ValueString()
	} else {
		projectId = os.Getenv("UCLOUD_PROJECT_ID")
	}

	if !model.PublicKey.IsNull() {
		publicKey = model.PublicKey.ValueString()
	} else {
		publicKey = os.Getenv("UCLOUD_PUBLIC_KEY")
	}

	if !model.PrivateKey.IsNull() {
		privateKey = model.PrivateKey.ValueString()
	} else {
		privateKey = os.Getenv("UCLOUD_PRIVATE_KEY")
	}

	// If any of the expected configuration are missing, return
	// errors with provider-specific guidance.
	if apiUrl == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Missing Ucloud API Url",
			"The provider cannot create the Ucloud API client as there is a "+
				"missing or empty value for the Ucloud API Url. Set the "+
				"value in the configuration or use the UCLOUD_API_URL"+
				"environment variable. If either is already set, ensure the value "+
				"is not empty.",
		)
	}

	if region == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("region"),
			"Missing Ucloud Region",
			"The provider cannot create the Ucloud API client as there is a "+
				"missing or empty value for the Ucloud Region. Set the "+
				"value in the configuration or use the UCLOUD_REGION"+
				"environment variable. If either is already set, ensure the value "+
				"is not empty.",
		)
	}

	if zone == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("zone"),
			"Missing Ucloud Zone",
			"The provider cannot create the Ucloud API client as there is a "+
				"missing or empty value for the Ucloud Zone. Set the "+
				"value in the configuration or use the UCLOUD_ZONE"+
				"environment variable. If either is already set, ensure the value "+
				"is not empty.",
		)
	}

	if projectId == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("project_id"),
			"Missing Ucloud ProjectId",
			"The provider cannot create the Ucloud API client as there is a "+
				"missing or empty value for the Ucloud ProjectId. Set the "+
				"value in the configuration or use the UCLOUD_PROJECT_ID"+
				"environment variable. If either is already set, ensure the value "+
				"is not empty.",
		)
	}

	if publicKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("public_key"),
			"Missing Ucloud PublicKey",
			"The provider cannot create the Ucloud API client as there is a "+
				"missing or empty value for the Ucloud PublicKey. Set the "+
				"value in the configuration or use the UCLOUD_PUBLIC_KEY"+
				"environment variable. If either is already set, ensure the value "+
				"is not empty.",
		)
	}

	if privateKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("private_key"),
			"Missing Ucloud PrivateKey",
			"The provider cannot create the Ucloud API client as there is a "+
				"missing or empty value for the Ucloud PrivateKey. Set the "+
				"value in the configuration or use the UCLOUD_PRIVATE_KEY"+
				"environment variable. If either is already set, ensure the value "+
				"is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	keys := auth.Credential{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
	cfg := ucloud.Config{
		BaseUrl:   apiUrl,
		Region:    region,
		Zone:      zone,
		ProjectId: projectId,
	}
	client := ucdn.NewClient(&cfg, &keys)

	// Ucloud clients wrapper
	ucloudClients := ucloudClients{
		cdnClient: client,
	}

	resp.DataSourceData = ucloudClients
	resp.ResourceData = ucloudClients
}

func (p *ucloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUcloudCertDataSource,
	}
}

func (p *ucloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUcloudSslCertificateResource,
		NewUcloudCdnDomainResource,
		NewUcloudCdnDomainSslResource,
	}
}
