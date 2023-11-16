package ucloud

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/myklst/terraform-provider-st-ucloud/ucloud/api"
	"github.com/ucloud/ucloud-sdk-go/services/ucdn"
)

var (
	_ datasource.DataSource              = &ucloudCertDataSource{}
	_ datasource.DataSourceWithConfigure = &ucloudCertDataSource{}
)

type ucloudCert struct {
	CertName types.String `tfsdk:"cert_name"`
	Domains  types.List   `tfsdk:"domains"`
}

type ucloudCertDataSourceModel struct {
	CertList []ucloudCert `tfsdk:"cert_list"`
}

func newCertDataSourceModel() *ucloudCertDataSourceModel {
	m := &ucloudCertDataSourceModel{}
	m.CertList = make([]ucloudCert, 0)
	return m
}

type ucloudCertDataSource struct {
	client *ucdn.UCDNClient
}

func NewUcloudCertDataSource() datasource.DataSource {
	return &ucloudCertDataSource{}
}

func (d *ucloudCertDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssl_certificate"
}

func (d *ucloudCertDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source provides certificates configured in ucloud, including certificate name,domains associated with the certificate,etc.",
		Attributes: map[string]schema.Attribute{
			"cert_list": schema.ListNestedAttribute{
				Description: "List of certificate",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cert_name": schema.StringAttribute{
							Description: "The name of certificate",
							Computed:    true,
						},
						"domains": schema.ListAttribute{
							Description: "Domain associcated with this certificate.",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
				Computed: true,
			},
		},
	}
}

func (d *ucloudCertDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(ucloudClients).cdnClient
}

func (d *ucloudCertDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model := newCertDataSourceModel()

	resp.Diagnostics.Append(req.Config.Get(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certlist := api.GetCertificates(d.client, "")
	for _, cert := range certlist {
		ucloudCert := ucloudCert{}
		ucloudCert.CertName = types.StringValue(cert.CertName)
		domains, diags := types.ListValueFrom(ctx, types.StringType, cert.Domains)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		ucloudCert.Domains = domains
		model.CertList = append(model.CertList, ucloudCert)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
