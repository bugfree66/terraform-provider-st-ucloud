package ucloud

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/myklst/terraform-provider-st-ucloud/ucloud/api"
	"github.com/ucloud/ucloud-sdk-go/services/ucdn"
)

type ucloudCdnDomainSslAssociationModel struct {
	DomainId           types.String `tfsdk:"domain_id"`
	SslCertificateName types.String `tfsdk:"ssl_certificate_name"`
}

type ucloudCdnDomainSslAssociationResource struct {
	client *ucdn.UCDNClient
}

var (
	_ resource.Resource = &ucloudCdnDomainSslAssociationResource{}
)

func NewUcloudCdnDomainSslResource() resource.Resource {
	return &ucloudCdnDomainSslAssociationResource{}
}

func (r *ucloudCdnDomainSslAssociationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_domain_ssl_association"
}

func (r *ucloudCdnDomainSslAssociationResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "",
		Attributes: map[string]schema.Attribute{
			"domain_id": &schema.StringAttribute{
				Description: "Id of acceleration domain, generated by ucloud.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ssl_certificate_name": &schema.StringAttribute{
				Description: "Ssl certificate name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ucloudCdnDomainSslAssociationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(ucloudClients).cdnClient
}

func (r *ucloudCdnDomainSslAssociationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model *ucloudCdnDomainSslAssociationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := api.UpdateDomainHttpsConfig(r.client, model.DomainId.ValueString(), true, model.SslCertificateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to Create CdnDomainSslAssociation", err.Error())
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *ucloudCdnDomainSslAssociationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model *ucloudCdnDomainSslAssociationModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainConfig, err := api.GetUcdnDomainConfig(r.client, model.DomainId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to Read CdnDomainSslAssociation", err.Error())
		return
	}

	if domainConfig == nil || domainConfig.HttpsStatusCn == "disable" {
		resp.State.RemoveResource(ctx)
		return
	}

	model.SslCertificateName = types.StringPointerValue(&domainConfig.CertNameCn)
	resp.State.Set(ctx, &model)
}

func (r *ucloudCdnDomainSslAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model *ucloudCdnDomainSslAssociationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := api.UpdateDomainHttpsConfig(r.client, model.DomainId.ValueString(), true, model.SslCertificateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to Update CdnDomainSslAssociation", err.Error())
	}
	api.DeleteUnusedCertificate(r.client, model.SslCertificateName.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *ucloudCdnDomainSslAssociationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model *ucloudCdnDomainSslAssociationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := api.UpdateDomainHttpsConfig(r.client, model.DomainId.ValueString(), false, model.SslCertificateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to Delete CdnDomainSslAssociation", err.Error())
	}
	api.DeleteUnusedCertificate(r.client, model.SslCertificateName.ValueString())
}
