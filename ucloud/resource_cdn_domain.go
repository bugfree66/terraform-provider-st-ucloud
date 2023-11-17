package ucloud

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/myklst/terraform-provider-st-ucloud/ucloud/api"
	"github.com/ucloud/ucloud-sdk-go/services/ucdn"
	uerr "github.com/ucloud/ucloud-sdk-go/ucloud/error"
	"github.com/ucloud/ucloud-sdk-go/ucloud/request"
	"github.com/ucloud/ucloud-sdk-go/ucloud/response"
)

type ucloudCacheRuleModel struct {
	PathPattern      types.String `tfsdk:"path_pattern"`
	Description      types.String `tfsdk:"description"`
	TTL              types.Int64  `tfsdk:"ttl"`
	CacheUnit        types.String `tfsdk:"cache_unit"`
	CacheBehavior    types.Bool   `tfsdk:"cache_behavior"`
	FollowOriginRule types.Bool   `tfsdk:"follow_origin_rule"`
}

type ucloudOriginConfigModel struct {
	OriginIpList    types.List   `tfsdk:"origin_ip_list"`
	OriginHost      types.String `tfsdk:"origin_host"`
	OriginPort      types.Int64  `tfsdk:"origin_port"`
	OriginProtocol  types.String `tfsdk:"origin_protocol"`
	OriginFollow301 types.Int64  `tfsdk:"origin_follow301"`
}

type ucloudCacheConfigModel struct {
	CacheHost types.String            `tfsdk:"cache_host"`
	RuleList  []*ucloudCacheRuleModel `tfsdk:"cache_rule"`
}

type ucloudReferConfigModel struct {
	ReferType types.Int64 `tfsdk:"refer_type"`
	NullRefer types.Int64 `tfsdk:"null_refer"`
	ReferList types.List  `tfsdk:"refer_list"`
}

var ucloudReferConfigAttributeTypes map[string]attr.Type = map[string]attr.Type{
	"refer_type": types.Int64Type,
	"null_refer": types.Int64Type,
	"refer_list": types.ListType{}.WithElementType(types.StringType),
}

type ucloudAccessControlConfigModel struct {
	IpBlackList types.List              `tfsdk:"ip_blacklist"`
	ReferConf   *ucloudReferConfigModel `tfsdk:"refer_conf"`
}

var ucloudAccessControlConfigAttributeTypes map[string]attr.Type = map[string]attr.Type{
	"ip_blacklist": types.ListType{}.WithElementType(types.StringType),
	"refer_conf":   types.ObjectType{}.WithAttributeTypes(ucloudReferConfigAttributeTypes),
}

type ucloudAdvancedConfModel struct {
	HttpClientHeaderList types.List `tfsdk:"http_client_header_list"`
	HttpOriginHeaderList types.List `tfsdk:"http_origin_header_list"`
	Http2Https           types.Bool `tfsdk:"http_to_https"`
}

var ucloudAdvancedConfigAttributeTypes map[string]attr.Type = map[string]attr.Type{
	"http_client_header_list": types.ListType{}.WithElementType(types.StringType),
	"http_origin_header_list": types.ListType{}.WithElementType(types.StringType),
	"http_to_https":           types.BoolType,
}

type ucloudCdnDomainResourceModel struct {
	DomainId   types.String `tfsdk:"domain_id"`
	Domain     types.String `tfsdk:"domain"`
	Cname      types.String `tfsdk:"cname"`
	Status     types.String `tfsdk:"status"`
	CreateTime types.Int64  `tfsdk:"create_time"`
	TestUrl    types.String `tfsdk:"test_url"`
	AreaCode   types.String `tfsdk:"area_code"`
	CdnType    types.String `tfsdk:"cdn_type"`
	Tag        types.String `tfsdk:"tag"`

	OriginConfig *ucloudOriginConfigModel `tfsdk:"origin_conf"`

	CacheConf *ucloudCacheConfigModel `tfsdk:"cache_conf"`

	AccessControlConfig types.Object `tfsdk:"access_control_conf"`

	AdvancedConf types.Object `tfsdk:"advanced_conf"`
}

type ucloudCdnDomainResource struct {
	client *ucdn.UCDNClient
}

var (
	_ resource.Resource               = &ucloudCdnDomainResource{}
	_ resource.ResourceWithConfigure  = &ucloudCdnDomainResource{}
	_ resource.ResourceWithModifyPlan = &ucloudCdnDomainResource{}
)

func NewUcloudCdnDomainResource() resource.Resource {
	return &ucloudCdnDomainResource{}
}

func (r *ucloudCdnDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_domain"
}

func (r *ucloudCdnDomainResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This resource provides the configuration of acceleration domain",
		Attributes: map[string]schema.Attribute{
			"domain_id": &schema.StringAttribute{
				Description: "Id of acceleration domain, generated by ucloud.",
				Computed:    true,
			},
			"domain": &schema.StringAttribute{
				Description: "Acceleration domain",
				Required:    true,
			},
			"cname": &schema.StringAttribute{
				Description: "Cname",
				Computed:    true,
			},
			"status": &schema.StringAttribute{
				Description: "Domain status",
				Computed:    true,
			},
			"create_time": &schema.Int64Attribute{
				Description: "Create time.",
				Computed:    true,
			},
			"test_url": &schema.StringAttribute{
				Description: "Test url",
				Required:    true,
			},
			"area_code": &schema.StringAttribute{
				Description: "Acceleration area.`cn` represents China.`abroad` represents regions outside China.If the value is unset,domain is accelerated in all regions",
				Required:    true,
			},
			"cdn_type": &schema.StringAttribute{
				Description: "`web` for website service,`stream` for video service,`download` for download service",
				Required:    true,
			},
			"tag": &schema.StringAttribute{
				Description: "The group of service.If the value is unset. `Default` is used as default value",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Default"),
			},
			"advanced_conf": &schema.SingleNestedAttribute{
				Description: "The advance configuration.",
				Attributes: map[string]schema.Attribute{
					"http_client_header_list": schema.ListAttribute{
						Description: "Add http header when send response to client.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
					},
					"http_origin_header_list": schema.ListAttribute{
						Description: "Add http header when send request to origin",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
					},
					"http_to_https": schema.BoolAttribute{
						Description: "If perform a forced conversion from http to https.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
				},
				Computed: true,
				Optional: true,
			},
			"access_control_conf": &schema.SingleNestedAttribute{
				Description: "The configuration of access control.",
				Attributes: map[string]schema.Attribute{
					"ip_blacklist": schema.ListAttribute{
						Description: "Request from address in blacklist will be denied.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
					},
					"refer_conf": &schema.SingleNestedAttribute{
						Description: "",
						Attributes: map[string]schema.Attribute{
							"refer_type": schema.Int64Attribute{
								Description: "The type of anti-leech rules.If the value is 0,`refer_list` is whitelist,requests with these refers will be allowed.If the value is 1,`refer_list` is blacklist,requests with these refers will be denied.",
								Optional:    true,
								Computed:    true,
								Default:     int64default.StaticInt64(0),
							},
							"null_refer": schema.Int64Attribute{
								Description: "When `refer_type` is 0,if the value is 0,NULL refer requests are not allowed.",
								Optional:    true,
								Computed:    true,
								Default:     int64default.StaticInt64(0),
							},
							"refer_list": schema.ListAttribute{
								Description: "The anti-leech rule list",
								ElementType: types.StringType,
								Optional:    true,
								Computed:    true,
								Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
							},
						},
						Optional: true,
						Computed: true,
					},
				},
				Computed: true,
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"origin_conf": &schema.SingleNestedBlock{
				Description: "The configuration of origin",
				Attributes: map[string]schema.Attribute{
					"origin_ip_list": schema.ListAttribute{
						Description: "The ip list of origin",
						ElementType: types.StringType,
						Required:    true,
					},
					"origin_host": schema.StringAttribute{
						Description: "The host of origin",
						Optional:    true,
						Computed:    true,
					},
					"origin_port": schema.Int64Attribute{
						Description: "The service port of origin",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(80),
					},
					"origin_protocol": schema.StringAttribute{
						Description: "The protocol of origin.The optional values are `http` and `https`",
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("https", "http"),
						},
						Default: stringdefault.StaticString("http"),
					},
					"origin_follow301": schema.Int64Attribute{
						Description: "Whether redirect according to the url from origin.The optional values are 0 and 1",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(0),
					},
				},
			},
			"cache_conf": schema.SingleNestedBlock{
				Description: "The configuration of cache",
				Attributes: map[string]schema.Attribute{
					"cache_host": schema.StringAttribute{
						Description: "Cache Host",
						Optional:    true,
						Computed:    true,
					},
				},
				Blocks: map[string]schema.Block{
					"cache_rule": &schema.ListNestedBlock{
						Description: "The list of cache rule",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"path_pattern": schema.StringAttribute{
									Description: "The pattern of path",
									Required:    true,
								},
								"description": schema.StringAttribute{
									Description: "The description of rule",
									Optional:    true,
									Computed:    true,
									Default:     stringdefault.StaticString(""),
								},
								"ttl": schema.Int64Attribute{
									Description: "The cache time",
									Optional:    true,
									Computed:    true,
									Default:     int64default.StaticInt64(0),
								},
								"cache_unit": schema.StringAttribute{
									Description: "The unit of caching time.The optional values are `sec`,`min`,`hour` and `day`.",
									Optional:    true,
									Computed:    true,
									Default:     stringdefault.StaticString("sec"),
								},
								"cache_behavior": schema.BoolAttribute{
									Description: "If caching is enabled.The optional values are true and false.",
									Required:    true,
								},
								"follow_origin_rule": schema.BoolAttribute{
									Description: "If follow caching instructions in http header from the origin.The optional values are true and false.",
									Optional:    true,
									Computed:    true,
									Default:     booldefault.StaticBool(false),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *ucloudCdnDomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(ucloudClients).cdnClient
}

func (r *ucloudCdnDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var (
		model                   ucloudCdnDomainResourceModel
		createCdnDomainResponse api.CreateCdnDomainResponse
	)

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	createCdnDomainRequest, diags := r.buildCreateCdnDomainRequest(&model)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var err error
	createCdnDomain := func() error {
		err = r.client.InvokeAction("BatchCreateNewUcdnDomain", createCdnDomainRequest, &createCdnDomainResponse)
		if err != nil {
			if cErr, ok := err.(uerr.ClientError); ok && cErr.Retryable() {
				return err
			}
			return backoff.Permanent(err)
		}

		if createCdnDomainResponse.RetCode != 0 {
			return backoff.Permanent(fmt.Errorf("%s", createCdnDomainResponse.Message))
		}
		if len(createCdnDomainResponse.DomainList) == 0 {
			return backoff.Permanent(fmt.Errorf("%s", "domain list is empty"))
		}
		if createCdnDomainResponse.DomainList[0].RetCode != 0 {
			return backoff.Permanent(fmt.Errorf("%s", createCdnDomainResponse.DomainList[0].Message))
		}

		return nil
	}
	reconnectBackoff := backoff.NewExponentialBackOff()
	reconnectBackoff.MaxElapsedTime = 30 * time.Second
	err = backoff.Retry(createCdnDomain, reconnectBackoff)
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to Create CdnDomain", err.Error())
		return
	}
	model.DomainId = types.StringValue(createCdnDomainResponse.DomainList[0].DomainId)
	_, err = api.WaitForDomainStatus(r.client, model.DomainId.ValueString(), []string{api.DomainStatusEnable, api.DomainStatusChekFail})
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to Get CdnDomain Status", err.Error())
		return
	}

	domainConfig, err := api.GetUcdnDomainConfig(r.client, model.DomainId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to Get CdnDomain", err.Error())
		return
	}

	resp.Diagnostics.Append(updateUcloudCdnDomainResourceModel(ctx, &model, domainConfig)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, model)
}

func (r *ucloudCdnDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model ucloudCdnDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainConfig, err := api.GetUcdnDomainConfig(r.client, model.DomainId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to Read CdnDomain", err.Error())
		return
	}

	resp.Diagnostics.Append(updateUcloudCdnDomainResourceModel(ctx, &model, domainConfig)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *ucloudCdnDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var (
		model                   ucloudCdnDomainResourceModel
		state                   ucloudCdnDomainResourceModel
		updateCdnDomainResponse response.CommonBase
	)

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.DomainId = state.DomainId

	var err error
	reconnectBackoff := backoff.NewExponentialBackOff()
	reconnectBackoff.MaxElapsedTime = 30 * time.Second
	updateDomainConfig := func() error {
		err = r.client.InvokeAction("UpdateUcdnDomainConfig", r.buildUpdateCdnDomainRequest(&model), &updateCdnDomainResponse)
		if err != nil {
			if cErr, ok := err.(uerr.ClientError); ok && cErr.Retryable() {
				return err
			}
			return backoff.Permanent(err)
		}
		if updateCdnDomainResponse.RetCode != 0 && updateCdnDomainResponse.RetCode != 44015 {
			return backoff.Permanent(fmt.Errorf("%s", updateCdnDomainResponse.Message))
		}
		return nil
	}
	err = backoff.Retry(updateDomainConfig, reconnectBackoff)
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to Update CdnDomain", err.Error())
		return
	}

	copyUcloudCdnDomainResourceModelComputeFields(&model, &state)

	status, err := api.WaitForDomainStatus(r.client, model.DomainId.ValueString(), []string{api.DomainStatusEnable})
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to get update status", err.Error())
		return
	}
	model.Status = types.StringValue(status)

	resp.State.Set(ctx, model)
}

func (r *ucloudCdnDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var (
		model                          ucloudCdnDomainResourceModel
		updateUcdnDomainStatusResponse response.CommonBase
	)
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateUcdnDomainStatusRequest := &struct {
		request.CommonBase
		DomainId string
		Status   string
		IsDcdn   bool
	}{
		CommonBase: request.CommonBase{
			ProjectId: &r.client.GetConfig().ProjectId,
		},
		DomainId: model.DomainId.ValueString(),
		Status:   "delete",
		IsDcdn:   false,
	}

	var err error
	updateDomainStatus := func() error {
		err = r.client.InvokeAction("UpdateUcdnDomainStatus", updateUcdnDomainStatusRequest, &updateUcdnDomainStatusResponse)
		if err != nil {
			if cErr, ok := err.(uerr.ClientError); ok && cErr.Retryable() {
				return err
			}
			return backoff.Permanent(err)
		}
		if updateUcdnDomainStatusResponse.RetCode != 0 {
			return backoff.Permanent(fmt.Errorf("%s", updateUcdnDomainStatusResponse.Message))
		}
		return nil
	}
	reconnectBackoff := backoff.NewExponentialBackOff()
	reconnectBackoff.MaxElapsedTime = 30 * time.Second
	err = backoff.Retry(updateDomainStatus, reconnectBackoff)
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to Update CdnDomain", err.Error())
		return
	}
	_, err = api.WaitForDomainStatus(r.client, model.DomainId.ValueString(), []string{api.DomainStatusDelete})
	if err != nil {
		resp.Diagnostics.AddError("[API ERROR] Fail to get update status", err.Error())
		return
	}
}

func (r *ucloudCdnDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain_id"), req, resp)
}

func (r *ucloudCdnDomainResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var plan ucloudCdnDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.OriginConfig != nil {
		if plan.OriginConfig.OriginHost.IsNull() || plan.OriginConfig.OriginHost.IsUnknown() {
			plan.OriginConfig.OriginHost = plan.Domain
		}
	}

	if plan.CacheConf == nil {
		plan.CacheConf = &ucloudCacheConfigModel{}
	}
	if plan.CacheConf.CacheHost.IsNull() || plan.CacheConf.CacheHost.IsUnknown() {
		plan.CacheConf.CacheHost = plan.Domain
	}
	if plan.CacheConf.RuleList == nil || len(plan.CacheConf.RuleList) == 0 {
		rule := &ucloudCacheRuleModel{
			PathPattern:      types.StringValue("/"),
			TTL:              types.Int64Value(0),
			CacheUnit:        types.StringValue("sec"),
			Description:      types.StringValue(""),
			CacheBehavior:    types.BoolValue(true),
			FollowOriginRule: types.BoolValue(false),
		}
		plan.CacheConf.RuleList = []*ucloudCacheRuleModel{rule}
	}
	resp.Plan.SetAttribute(ctx, path.Root("cache_conf"), plan.CacheConf)

	if plan.AdvancedConf.IsNull() || plan.AdvancedConf.IsUnknown() {
		plan.AdvancedConf = types.ObjectValueMust(ucloudAdvancedConfigAttributeTypes, map[string]attr.Value{
			"http_client_header_list": types.ListValueMust(types.StringType, []attr.Value{}),
			"http_origin_header_list": types.ListValueMust(types.StringType, []attr.Value{}),
			"http_to_https":           types.BoolValue(false),
		})
	}

	if plan.AccessControlConfig.IsNull() || plan.AccessControlConfig.IsUnknown() || plan.AccessControlConfig.Attributes()["refer_conf"].IsNull() {
		referConfig := types.ObjectValueMust(ucloudReferConfigAttributeTypes, map[string]attr.Value{
			"refer_list": types.ListValueMust(types.StringType, []attr.Value{}),
			"null_refer": types.Int64Value(0),
			"refer_type": types.Int64Value(0),
		})
		if plan.AccessControlConfig.IsNull() || plan.AccessControlConfig.IsUnknown() {
			plan.AccessControlConfig = types.ObjectValueMust(ucloudAccessControlConfigAttributeTypes, map[string]attr.Value{
				"ip_blacklist": types.ListValueMust(types.StringType, []attr.Value{}),
				"refer_conf":   referConfig,
			})
		} else {
			plan.AccessControlConfig = types.ObjectValueMust(ucloudAccessControlConfigAttributeTypes, map[string]attr.Value{
				"ip_blacklist": plan.AccessControlConfig.Attributes()["ip_blacklist"],
				"refer_conf":   referConfig,
			})
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *ucloudCdnDomainResource) buildCreateCdnDomainRequest(m *ucloudCdnDomainResourceModel) (*api.CreateCdnDomainRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	domainConfig := api.CreateDomainConfig{}
	domainConfig.Domain = m.Domain.ValueString()
	diags.Append(m.OriginConfig.OriginIpList.ElementsAs(nil, &domainConfig.OriginIp, false)...)
	if m.OriginConfig != nil {
		domainConfig.OriginHost = m.OriginConfig.OriginHost.ValueString()
	}
	domainConfig.TestUrl = m.TestUrl.ValueString()
	if m.CacheConf != nil {
		domainConfig.CacheConf = make([]api.CreateDomainCacheConf, 0)
		for _, rule := range m.CacheConf.RuleList {
			cc := api.CreateDomainCacheConf{}
			cc.PathPattern = rule.PathPattern.ValueString()
			cc.CacheTTL = rule.TTL.ValueInt64()
			cc.CacheUnit = rule.CacheUnit.ValueString()
			cc.CacheBehavior = rule.CacheBehavior.ValueBool()
			domainConfig.CacheConf = append(domainConfig.CacheConf, cc)
		}
	}
	domainConfig.AreaCode = m.AreaCode.ValueStringPointer()
	domainConfig.CdnType = m.CdnType.ValueStringPointer()
	domainConfig.Tag = m.TestUrl.ValueStringPointer()

	return &api.CreateCdnDomainRequest{
		CommonBase: request.CommonBase{
			ProjectId: &r.client.GetConfig().ProjectId,
		},
		DomainList: []api.CreateDomainConfig{domainConfig},
	}, diags
}

func (r *ucloudCdnDomainResource) buildUpdateCdnDomainRequest(m *ucloudCdnDomainResourceModel) *api.UpdateCdnDomainRequest {
	domainConf := api.UpdateCdnDomainConfig{}
	domainConf.DomainId = m.DomainId.ValueString()
	// origin
	if m.OriginConfig != nil {
		m.OriginConfig.OriginIpList.ElementsAs(nil, &domainConf.OriginConf.OriginIp, false)
		domainConf.OriginConf.OriginHost = m.OriginConfig.OriginHost.ValueStringPointer()
		domainConf.OriginConf.OriginPort = m.OriginConfig.OriginPort.ValueInt64Pointer()
		domainConf.OriginConf.OriginProtocol = m.OriginConfig.OriginProtocol.ValueStringPointer()
		domainConf.OriginConf.OriginFollow301 = m.OriginConfig.OriginFollow301.ValueInt64Pointer()
	}
	// cache control
	if m.CacheConf != nil {
		domainConf.CacheConf.CacheHost = m.CacheConf.CacheHost.ValueStringPointer()
		domainConf.CacheConf.CacheList = make([]api.UpdateCdnCache, 0)
		for _, rule := range m.CacheConf.RuleList {
			uc := api.UpdateCdnCache{}
			uc.PathPattern = rule.PathPattern.ValueString()
			uc.TTL = rule.TTL.ValueInt64()
			uc.FollowOriginRule = rule.FollowOriginRule.ValueBoolPointer()
			uc.Description = rule.Description.ValueStringPointer()
			uc.CacheUnit = rule.CacheUnit.ValueString()
			uc.CacheBehavior = rule.CacheBehavior.ValueBool()
			domainConf.CacheConf.CacheList = append(domainConf.CacheConf.CacheList, uc)
		}
	}
	// access control
	if !m.AccessControlConfig.IsNull() {
		m.AccessControlConfig.Attributes()["ip_blacklist"].(types.List).ElementsAs(nil, &domainConf.AccessControlConf.IpBlackList, false)
		domainConf.AccessControlConf.ReferConf.NullRefer = m.AccessControlConfig.Attributes()["refer_conf"].(types.Object).Attributes()["null_refer"].(types.Int64).ValueInt64Pointer()
		domainConf.AccessControlConf.ReferConf.ReferType = m.AccessControlConfig.Attributes()["refer_conf"].(types.Object).Attributes()["refer_type"].(types.Int64).ValueInt64Pointer()
		m.AccessControlConfig.Attributes()["refer_conf"].(types.Object).Attributes()["refer_list"].(types.List).ElementsAs(nil, &domainConf.AccessControlConf.ReferConf.ReferList, false)
	}
	// advanced config
	if !m.AdvancedConf.IsNull() {
		domainConf.AdvancedConf.Http2Https = m.AdvancedConf.Attributes()["http_to_https"].(types.Bool).ValueBoolPointer()
		m.AdvancedConf.Attributes()["http_client_header_list"].(types.List).ElementsAs(nil, &domainConf.AdvancedConf.HttpClientHeader, false)
		m.AdvancedConf.Attributes()["http_origin_header_list"].(types.List).ElementsAs(nil, &domainConf.AdvancedConf.HttpOriginHeader, false)
	}

	return &api.UpdateCdnDomainRequest{
		CommonBase: request.CommonBase{
			ProjectId: &r.client.GetConfig().ProjectId,
		},
		DomainList: []api.UpdateCdnDomainConfig{domainConf},
	}
}

func updateUcloudCdnDomainResourceModelComputeFields(model *ucloudCdnDomainResourceModel, info *ucdn.DomainConfigInfo) {
	model.DomainId = types.StringValue(info.DomainId)
	model.Cname = types.StringValue(info.Cname)
	model.Status = types.StringValue(info.Status)
	model.CreateTime = types.Int64Value(int64(info.CreateTime))
	if model.OriginConfig != nil {
		model.OriginConfig.OriginHost = types.StringValue(info.OriginConf.OriginHost)
	}
}

func copyUcloudCdnDomainResourceModelComputeFields(dst, src *ucloudCdnDomainResourceModel) {
	dst.DomainId = src.DomainId
	dst.Cname = src.Cname
	dst.Status = src.Status
	dst.CreateTime = src.CreateTime
}

func updateUcloudCdnDomainResourceModel(ctx context.Context, model *ucloudCdnDomainResourceModel, info *ucdn.DomainConfigInfo) diag.Diagnostics {
	var diags, result diag.Diagnostics

	model.AreaCode = types.StringValue(info.AreaCode)
	model.CdnType = types.StringValue(info.CdnType)
	model.Status = types.StringValue(info.Status)
	model.Cname = types.StringValue(info.Cname)
	model.CreateTime = types.Int64Value(int64(info.CreateTime))
	model.TestUrl = types.StringValue(info.TestUrl)

	model.OriginConfig = &ucloudOriginConfigModel{}
	model.OriginConfig.OriginIpList, diags = types.ListValueFrom(ctx, types.StringType, info.OriginConf.OriginIpList)
	result.Append(diags...)
	if model.OriginConfig.OriginIpList.IsNull() {
		model.OriginConfig.OriginIpList = types.ListValueMust(types.StringType, []attr.Value{})
	}
	model.OriginConfig.OriginHost = types.StringValue(info.OriginConf.OriginHost)
	model.OriginConfig.OriginPort = types.Int64Value(int64(info.OriginConf.OriginPort))
	model.OriginConfig.OriginProtocol = types.StringValue(info.OriginConf.OriginProtocol)
	model.OriginConfig.OriginFollow301 = types.Int64Value(int64(info.OriginConf.OriginFollow301))

	model.CacheConf = &ucloudCacheConfigModel{}
	model.CacheConf.CacheHost = types.StringValue(info.CacheConf.CacheHost)
	model.CacheConf.RuleList = make([]*ucloudCacheRuleModel, 0)
	for _, conf := range info.CacheConf.CacheList {
		c := &ucloudCacheRuleModel{
			PathPattern:      types.StringValue(conf.PathPattern),
			Description:      types.StringValue(conf.Description),
			TTL:              types.Int64Value(int64(conf.CacheTTL)),
			CacheUnit:        types.StringValue(conf.CacheUnit),
			CacheBehavior:    types.BoolValue(conf.CacheBehavior),
			FollowOriginRule: types.BoolValue(conf.FollowOriginRule),
		}
		model.CacheConf.RuleList = append(model.CacheConf.RuleList, c)
	}

	referList, diags := types.ListValueFrom(ctx, types.StringType, info.AccessControlConf.ReferConf.ReferList)
	result.Append(diags...)
	if referList.IsNull() {
		referList = types.ListValueMust(types.StringType, []attr.Value{})
	}
	referConfig := types.ObjectValueMust(ucloudReferConfigAttributeTypes, map[string]attr.Value{
		"refer_type": types.Int64Value(int64(info.AccessControlConf.ReferConf.ReferType)),
		"null_refer": types.Int64Value(int64(info.AccessControlConf.ReferConf.NullRefer)),
		"refer_list": referList,
	})
	ipBlackList, diags := types.ListValueFrom(ctx, types.StringType, info.AccessControlConf.IpBlackList)
	result.Append(diags...)
	if ipBlackList.IsNull() {
		ipBlackList = types.ListValueMust(types.StringType, []attr.Value{})
	}
	model.AccessControlConfig = types.ObjectValueMust(ucloudAccessControlConfigAttributeTypes, map[string]attr.Value{
		"refer_conf":   referConfig,
		"ip_blacklist": ipBlackList,
	})

	clientHeaderList, diags := types.ListValueFrom(ctx, types.StringType, info.AdvancedConf.HttpClientHeader)
	result.Append(diags...)
	if clientHeaderList.IsNull() {
		clientHeaderList = types.ListValueMust(types.StringType, []attr.Value{})
	}
	originHeaderList, diags := types.ListValueFrom(ctx, types.StringType, info.AdvancedConf.HttpOriginHeader)
	result.Append(diags...)
	if originHeaderList.IsNull() {
		originHeaderList = types.ListValueMust(types.StringType, []attr.Value{})
	}
	model.AdvancedConf = types.ObjectValueMust(ucloudAdvancedConfigAttributeTypes, map[string]attr.Value{
		"http_client_header_list": clientHeaderList,
		"http_origin_header_list": originHeaderList,
		"http_to_https":           types.BoolValue(info.AdvancedConf.Http2Https),
	})

	return result
}
