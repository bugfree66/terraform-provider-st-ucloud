package api

import (
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ucloud/ucloud-sdk-go/services/ucdn"
	uerr "github.com/ucloud/ucloud-sdk-go/ucloud/error"
	"github.com/ucloud/ucloud-sdk-go/ucloud/request"
	"github.com/ucloud/ucloud-sdk-go/ucloud/response"
)

const (
	DomainStatusEnable   = "enable"
	DomainStatusDelete   = "delete"
	DomainStatusChekFail = "checkFail"
)

type UpdateCdnHttpsRequest struct {
	request.CommonBase
	Region      string
	Zone        string
	Areacode    string
	DomainId    string
	HttpsStatus string
	CertName    string
}

func WaitForDomainStatus(client *ucdn.UCDNClient, domainId string, targetStatus []string) (string, error) {
	var (
		getUcdnDomainConfigResponse *ucdn.GetUcdnDomainConfigResponse
		err                         error
	)

	getUcdnDomainConfigRequest := ucdn.GetUcdnDomainConfigRequest{
		CommonBase: request.CommonBase{
			ProjectId: &client.GetConfig().ProjectId,
		},
		DomainId: []string{domainId},
	}

	getDomainConfig := func() error {
		getUcdnDomainConfigResponse, err = client.GetUcdnDomainConfig(&getUcdnDomainConfigRequest)
		if err != nil {
			if cErr, ok := err.(uerr.ClientError); ok && cErr.Retryable() {
				return err
			}
			if Retryable(getUcdnDomainConfigResponse.RetCode) {
				return errors.New(getUcdnDomainConfigResponse.Message)
			}
			return backoff.Permanent(err)
		}
		for _, status := range targetStatus {
			if status == DomainStatusDelete && len(getUcdnDomainConfigResponse.DomainList) == 0 {
				return nil
			} else if len(getUcdnDomainConfigResponse.DomainList) > 0 && status == getUcdnDomainConfigResponse.DomainList[0].Status {
				return nil
			}
		}
		return errors.New("unexpected status")
	}
	reconnectBackoff := backoff.NewExponentialBackOff()
	err = backoff.Retry(getDomainConfig, reconnectBackoff)
	if err != nil {
		return "", errors.New("fail to get expected status")
	}
	if len(getUcdnDomainConfigResponse.DomainList) == 0 {
		return DomainStatusDelete, nil
	}
	return getUcdnDomainConfigResponse.DomainList[0].Status, nil
}

func UpdateDomainHttpsConfig(client *ucdn.UCDNClient, domainId string, enable bool, certName string) error {
	domainConfig, err := GetUcdnDomainConfig(client, domainId)
	if err != nil {
		return err
	}
	if domainConfig == nil {
		return errors.New("domain config is nil")
	}
	areaCode := domainConfig.AreaCode
	areas := make([]string, 0)
	if areaCode == "all" {
		areas = append(areas, "abroad")
		areas = append(areas, "cn")
	} else {
		areas = append(areas, areaCode)
	}

	reconnectBackoff := backoff.NewExponentialBackOff()
	reconnectBackoff.MaxElapsedTime = 30 * time.Second
	updateCdnHttpsRequest := UpdateCdnHttpsRequest{
		CommonBase: request.CommonBase{
			ProjectId: &client.GetConfig().ProjectId,
		},
		Region:   client.GetConfig().Region,
		Zone:     client.GetConfig().Zone,
		DomainId: domainId,
	}
	if enable {
		updateCdnHttpsRequest.HttpsStatus = "enable"
		updateCdnHttpsRequest.CertName = certName
	} else {
		updateCdnHttpsRequest.HttpsStatus = "disable"
	}

	var updateCdnHttpsResponse response.CommonBase
	updateDomainHttpsConfig := func() error {
		err = client.InvokeAction("UpdateUcdnDomainHttpsConfig", &updateCdnHttpsRequest, &updateCdnHttpsResponse)
		if err != nil {
			if cErr, ok := err.(uerr.ClientError); ok && cErr.Retryable() {
				return err
			}
			if Retryable(updateCdnHttpsResponse.RetCode) {
				return errors.New(updateCdnHttpsResponse.Message)
			}
			return backoff.Permanent(err)
		}
		return nil
	}

	for _, area := range areas {
		updateCdnHttpsRequest.Areacode = area
		err = backoff.Retry(updateDomainHttpsConfig, reconnectBackoff)
		if err != nil {
			return err
		}
		WaitForDomainStatus(client, domainId, []string{DomainStatusEnable})
	}

	return nil
}

func GetUcdnDomainConfig(client *ucdn.UCDNClient, domainId string) (*DomainConfigInfo, error) {
	getUcdnDomainConfigRequest := ucdn.GetUcdnDomainConfigRequest{
		CommonBase: request.CommonBase{
			ProjectId: &client.GetConfig().ProjectId,
		},
		DomainId: []string{domainId},
	}

	var (
		getUcdnDomainConfigResponse getUcdnDomainConfigResponse
		err                         error
	)
	getDomainConfig := func() error {
		err = client.InvokeAction("GetUcdnDomainConfig", &getUcdnDomainConfigRequest, &getUcdnDomainConfigResponse)
		if err != nil {
			if cErr, ok := err.(uerr.ClientError); ok && cErr.Retryable() {
				return err
			}
			if Retryable(getUcdnDomainConfigResponse.RetCode) {
				return errors.New(getUcdnDomainConfigResponse.Message)
			}
			return backoff.Permanent(err)
		}
		return nil
	}
	reconnectBackoff := backoff.NewExponentialBackOff()
	reconnectBackoff.MaxElapsedTime = 30 * time.Second
	err = backoff.Retry(getDomainConfig, reconnectBackoff)
	if err != nil {
		return nil, err
	}

	if len(getUcdnDomainConfigResponse.DomainList) == 0 {
		return nil, nil
	}
	return &getUcdnDomainConfigResponse.DomainList[0], nil
}

type CreateDomainConfig struct {
	Domain     string
	OriginIp   []string
	OriginHost string
	TestUrl    string
	CacheConf  []CreateDomainCacheConf
	AreaCode   *string
	CdnType    *string
	Tag        *string
}

type CreateDomainCacheConf struct {
	PathPattern   string
	CacheTTL      int64
	CacheUnit     string
	CacheBehavior bool
}

type CreateCdnDomainRequest struct {
	request.CommonBase
	DomainList []CreateDomainConfig
}

type CreateCdnDomainResponse struct {
	response.CommonBase
	DomainList []struct {
		Domain   string `json:"Domain"`
		DomainId string `json:"DomainId"`
		RetCode  int    `json:"RetCode"`
		Message  string `json:"Message"`
	} `json:"DomainList"`
}

type CdnCacheRule struct {
	CacheBehavior    bool
	CacheTTL         int
	CacheUnit        string
	Description      string
	FollowOriginRule bool
	HttpCodePattern  string
	PathPattern      string
	UseRegex         bool
}

type CdnCacheConfig struct {
	CacheHost         *string
	CacheList         []CdnCacheRule
	HttpCodeCacheList []CdnCacheRule
}

type DomainConfigInfo struct {
	AccessControlConf ucdn.AccessControlConf
	AdvancedConf      ucdn.AdvancedConf
	AreaCode          string
	CacheConf         CdnCacheConfig
	CdnType           string
	CertNameAbroad    string
	CertNameCn        string
	Cname             string
	CreateTime        int
	Domain            string
	DomainId          string
	HttpsStatusAbroad string
	HttpsStatusCn     string
	OriginConf        ucdn.OriginConf
	Status            string
	Tag               string
	TestUrl           string
}

type getUcdnDomainConfigResponse struct {
	response.CommonBase
	DomainList []DomainConfigInfo
}

type UpdateCdnOriginConfig struct {
	OriginIp        []string
	OriginHost      *string
	OriginPort      *int64
	OriginProtocol  *string
	OriginFollow301 *int64
}

type UpdateCdnAccessControlConfig struct {
	IpBlackList      []string
	IpBlackListEmpty bool

	ReferConf struct {
		ReferType *int
		NullRefer *int
		ReferList []string
	}
	EnableRefer bool
}

type UpdateCdnAdvancedConfig struct {
	HttpClientHeader      []string
	HttpClientHeaderEmpty bool
	HttpOriginHeader      []string
	HttpOriginHeaderEmpty bool
	Http2Https            *bool
}

type UpdateCdnDomainConfig struct {
	DomainId string

	OriginConf        UpdateCdnOriginConfig
	AccessControlConf UpdateCdnAccessControlConfig
	CacheConf         CdnCacheConfig
	AdvancedConf      UpdateCdnAdvancedConfig
}

type UpdateCdnDomainRequest struct {
	request.CommonBase

	DomainList []UpdateCdnDomainConfig
}

func UpdateCdnDomain(client *ucdn.UCDNClient, req *UpdateCdnDomainRequest) error {
	if req == nil || len(req.DomainList) == 0 {
		return errors.New("UpdateCdnDomainRequest is empty")
	}

	var (
		err                     error
		updateCdnDomainResponse response.CommonBase
	)
	reconnectBackoff := backoff.NewExponentialBackOff()
	reconnectBackoff.MaxElapsedTime = 30 * time.Second
	updateDomainConfig := func() error {
		err = client.InvokeAction("UpdateUcdnDomainConfig", req, &updateCdnDomainResponse)
		if err != nil {
			if cErr, ok := err.(uerr.ClientError); ok && cErr.Retryable() {
				return err
			}
			if Retryable(updateCdnDomainResponse.RetCode) {
				return errors.New(updateCdnDomainResponse.Message)
			}
			return backoff.Permanent(err)
		}
		return nil
	}
	err = backoff.Retry(updateDomainConfig, reconnectBackoff)
	if err != nil {
		return err
	}

	_, err = WaitForDomainStatus(client, req.DomainList[0].DomainId, []string{DomainStatusEnable})
	if err != nil {
		return err
	}
	return nil
}
