package ucloud

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ucloud/ucloud-sdk-go/services/ucdn"
	uerr "github.com/ucloud/ucloud-sdk-go/ucloud/error"
	"github.com/ucloud/ucloud-sdk-go/ucloud/request"
	"golang.org/x/net/context"
)

type ucloudSslCertificateResourceModel struct {
	CertName types.String `tfsdk:"cert_name"`
	CaCert   types.String `tfsdk:"ca_cert"`
	Cert     types.String `tfsdk:"cert"`
	Key      types.String `tfsdk:"key"`
}

type ucloudSslCertificateResource struct {
	client *ucdn.UCDNClient
}

var (
	_ resource.Resource              = &ucloudSslCertificateResource{}
	_ resource.ResourceWithConfigure = &ucloudSslCertificateResource{}
)

func NewUcloudSslCertificateResource() resource.Resource {
	return &ucloudSslCertificateResource{}
}

func (r *ucloudSslCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssl_certificate"
}

func (r *ucloudSslCertificateResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The resource provides a SSL certificate for CDN domain",
		Attributes: map[string]schema.Attribute{
			"cert_name": &schema.StringAttribute{
				Description: "The name of certificate",
				Required:    true,
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

func (r *ucloudSslCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(ucloudClients).cdnClient
}

func (r *ucloudSslCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model ucloudSslCertificateResourceModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	addCertificateRequest := &ucdn.AddCertificateRequest{
		CommonBase: request.CommonBase{
			ProjectId: &r.client.GetConfig().ProjectId,
		},
		CertName:   model.CertName.ValueStringPointer(),
		UserCert:   model.Cert.ValueStringPointer(),
		PrivateKey: model.Key.ValueStringPointer(),
		CaCert:     model.CaCert.ValueStringPointer(),
	}

	addCertificate := func() error {
		addCertificateResponse, err := r.client.AddCertificate(addCertificateRequest)
		if err != nil {
			if cErr, ok := err.(uerr.ClientError); ok && cErr.Retryable() {
				return err
			}
			return backoff.Permanent(err)
		}
		if addCertificateResponse.RetCode != 0 {
			return backoff.Permanent(fmt.Errorf("%s", addCertificateResponse.Message))
		}
		return nil
	}
	reconnectBackoff := backoff.NewExponentialBackOff()
	reconnectBackoff.MaxElapsedTime = 30 * time.Second
	err := backoff.Retry(addCertificate, reconnectBackoff)
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Failed to Add Certificate", err.Error())
		return
	}
	resp.State.Set(ctx, &model)
}

func (r *ucloudSslCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

func (r *ucloudSslCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan ucloudSslCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.CertName != plan.CertName {
		resp.Diagnostics.AddError(
			"[API ERROR] Failed to Update Cert",
			"not allowed to modify cert_name")
		return
	}
	resp.Diagnostics.AddWarning("[API WARNING]", "not implemented to update cert")
	resp.State.Set(ctx, req.State.Raw)
}

func (r *ucloudSslCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model ucloudSslCertificateResourceModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteCertificateRequest := ucdn.DeleteCertificateRequest{
		CommonBase: request.CommonBase{
			ProjectId: &r.client.GetConfig().ProjectId,
		},
		CertName: model.CertName.ValueStringPointer(),
	}

	deleteCertificate := func() error {
		deleteCertificateResponse, err := r.client.DeleteCertificate(&deleteCertificateRequest)
		if err != nil {
			if cErr, ok := err.(uerr.ClientError); ok && cErr.Retryable() {
				return err
			}
			return backoff.Permanent(err)
		}
		if deleteCertificateResponse.RetCode != 0 {
			return backoff.Permanent(fmt.Errorf("%s", deleteCertificateResponse.Message))
		}
		return nil
	}
	reconnectBackoff := backoff.NewExponentialBackOff()
	reconnectBackoff.MaxElapsedTime = 30 * time.Second
	err := backoff.Retry(deleteCertificate, reconnectBackoff)
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Failed to Del Certificate", err.Error())
		return
	}
}
