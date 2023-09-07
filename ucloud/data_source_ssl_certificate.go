package ucloud

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ucloud/ucloud-sdk-go/services/ucdn"
	uerr "github.com/ucloud/ucloud-sdk-go/ucloud/error"
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

	var getCertificateV2Request ucdn.GetCertificateV2Request

	offset, limit := 0, 10
	getCertificateV2Request.Offset = &offset
	getCertificateV2Request.Limit = &limit
	getCertificateV2Request.ProjectId = &d.client.GetConfig().ProjectId

	reconnectBackoff := backoff.NewExponentialBackOff()
	reconnectBackoff.MaxElapsedTime = 30 * time.Second
	var (
		getCertificateV2Response *ucdn.GetCertificateV2Response
		err                      error
	)
	for {
		getCertificate := func() error {
			getCertificateV2Response, err = d.client.GetCertificateV2(&getCertificateV2Request)
			if err != nil {
				if cErr, ok := err.(uerr.ClientError); ok && cErr.Retryable() {
					return err
				}
				return backoff.Permanent(err)
			}
			if getCertificateV2Response.RetCode != 0 {
				return backoff.Permanent(fmt.Errorf("%s", getCertificateV2Response.Message))
			}
			return nil
		}
		err = backoff.Retry(getCertificate, reconnectBackoff)
		if err != nil {
			resp.Diagnostics.AddError("[API ERROR] Fail to Get Certificate", err.Error())
			return
		}
		for _, cert := range getCertificateV2Response.CertList {
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
		if len(getCertificateV2Response.CertList) < limit {
			break
		}
		offset += limit
	}
	resp.State.Set(ctx, model)
}
