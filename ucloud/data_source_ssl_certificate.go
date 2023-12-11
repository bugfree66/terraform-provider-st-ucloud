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
	_ datasource.DataSource              = &certDataSource{}
	_ datasource.DataSourceWithConfigure = &certDataSource{}
)

type certificate struct {
	CertName types.String `tfsdk:"cert_name"`
	Domains  types.List   `tfsdk:"domains"`
}

type certDataSourceModel struct {
	CertNameList types.List     `tfsdk:"cert_name_list"`
	CertList     []*certificate `tfsdk:"cert_list"`
}

type certDataSource struct {
	client *ucdn.UCDNClient
}

func NewCertDataSource() datasource.DataSource {
	return &certDataSource{}
}

func (d *certDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssl_certificate"
}

func (d *certDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source provides certificates configured in ucloud, including certificate name,domains associated with the certificate,etc.",
		Attributes: map[string]schema.Attribute{
			"cert_name_list": schema.ListAttribute{
				Description: "List of cert_name.If `cert_name_list` is null,retrieve all certificates.If `cert_name_list` is not null,retrieve certificates with specific name",
				ElementType: types.StringType,
				Optional:    true,
			},
			"cert_list": schema.ListNestedAttribute{
				Description: "List of certificate.",
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

func (d *certDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(ucloudClients).cdnClient
}

func (d *certDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model, state certDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.CertNameList = model.CertNameList

	var queryList []string
	resp.Diagnostics.Append(state.CertNameList.ElementsAs(ctx, &queryList, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	certlist, err := api.GetCertificates(d.client, queryList...)
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR]Fail to get ssl status", err.Error())
		return
	}

	state.CertList = make([]*certificate, len(queryList))
	for _, cert := range certlist {
		if cert == nil {
			continue
		}
		domains, _ := types.ListValueFrom(ctx, types.StringType, cert.Domains)
		c := &certificate{
			CertName: types.StringValue(cert.CertName),
			Domains:  domains,
		}
		state.CertList = append(state.CertList, c)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
