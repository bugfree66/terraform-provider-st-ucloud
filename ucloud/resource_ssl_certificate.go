package ucloud

import (
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/myklst/terraform-provider-st-ucloud/ucloud/api"
	"github.com/ucloud/ucloud-sdk-go/services/ucdn"
	"golang.org/x/net/context"
)

type sslCertificateResourceModel struct {
	CertName types.String `tfsdk:"cert_name"`
	CaCert   types.String `tfsdk:"ca_cert"`
	Cert     types.String `tfsdk:"cert"`
	Key      types.String `tfsdk:"key"`
}

type sslCertificateResource struct {
	client *ucdn.UCDNClient
}

var (
	_ resource.Resource                = &sslCertificateResource{}
	_ resource.ResourceWithConfigure   = &sslCertificateResource{}
	_ resource.ResourceWithImportState = &sslCertificateResource{}
)

func NewSslCertificateResource() resource.Resource {
	return &sslCertificateResource{}
}

func (r *sslCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssl_certificate"
}

func (r *sslCertificateResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The resource provides a SSL certificate for CDN domain",
		Attributes: map[string]schema.Attribute{
			"cert_name": &schema.StringAttribute{
				Description: "The name of certificate",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ca_cert": &schema.StringAttribute{
				Description: "CA certificate content",
				Optional:    true,
			},
			"cert": &schema.StringAttribute{
				Description: "Certificate content",
				Required:    true,
			},
			"key": &schema.StringAttribute{
				Description: "Private key content",
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *sslCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(ucloudClients).cdnClient
}

func (r *sslCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model *sslCertificateResourceModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := api.AddCertificate(r.client,
		model.CertName.ValueString(),
		model.Cert.ValueString(),
		model.Key.ValueString(),
		model.CaCert.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Failed to Add Certificate", err.Error())
		return
	}
	resp.State.Set(ctx, model)
}

func (r *sslCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state *sslCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certlist, err := api.GetCertificates(r.client, state.CertName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR]Fail to get ssl_certificate", err.Error())
		return
	}

	if len(certlist) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *sslCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan sslCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.CertName.Equal(plan.CertName) {
		resp.Diagnostics.AddError("[API ERROR] Fail to Update Certificate", "cert_name exists")
		return
	}

	err := api.AddCertificate(r.client,
		plan.CertName.ValueString(),
		plan.Cert.ValueString(),
		plan.Key.ValueString(),
		plan.CaCert.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to Create New Certificate", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *sslCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model sslCertificateResourceModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := api.DeleteCertificate(r.client, model.CertName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Failed to Del Certificate", err.Error())
		return
	}
}

func (r *sslCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("cert_name"), req, resp)
}
