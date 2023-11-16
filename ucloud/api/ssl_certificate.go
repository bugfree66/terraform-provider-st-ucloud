package api

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ucloud/ucloud-sdk-go/services/ucdn"
	uerr "github.com/ucloud/ucloud-sdk-go/ucloud/error"
	"github.com/ucloud/ucloud-sdk-go/ucloud/request"
)

func AddCertificate(client *ucdn.UCDNClient, name, userCert, privateKey, caCert string) error {
	addCertificateRequest := &ucdn.AddCertificateRequest{
		CommonBase: request.CommonBase{
			ProjectId: &client.GetConfig().ProjectId,
		},
		CertName:   &name,
		UserCert:   &userCert,
		PrivateKey: &privateKey,
		CaCert:     &caCert,
	}

	addCertificate := func() error {
		addCertificateResponse, err := client.AddCertificate(addCertificateRequest)
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
	return backoff.Retry(addCertificate, reconnectBackoff)
}

func GetCertificates(client *ucdn.UCDNClient, name string) []ucdn.CertList {
	result := make([]ucdn.CertList, 0)

	offset, limit := 0, 10
	getCertificateV2Request := ucdn.GetCertificateV2Request{
		CommonBase: request.CommonBase{
			ProjectId: &client.GetConfig().ProjectId,
		},
		Offset: &offset,
		Limit:  &limit,
	}

	var (
		getCertificateV2Response *ucdn.GetCertificateV2Response
		err                      error
	)
	getCertificate := func() error {
		getCertificateV2Response, err = client.GetCertificateV2(&getCertificateV2Request)
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

	reconnectBackoff := backoff.NewExponentialBackOff()
	reconnectBackoff.MaxElapsedTime = 30 * time.Second
	for {
		err = backoff.Retry(getCertificate, reconnectBackoff)
		if err != nil {
			return result
		}
		if name != "" {
			for _, cert := range getCertificateV2Response.CertList {
				if cert.CertName == name {
					result = append(result, cert)
					break
				}
			}
		} else {
			result = append(result, getCertificateV2Response.CertList...)
		}
		if len(getCertificateV2Response.CertList) < limit {
			break
		}
		offset += limit
	}
	return result
}

func DeleteCertificate(client *ucdn.UCDNClient, name string) error {
	deleteCertificateRequest := ucdn.DeleteCertificateRequest{
		CommonBase: request.CommonBase{
			ProjectId: &client.GetConfig().ProjectId,
		},
		CertName: &name,
	}

	deleteCertificate := func() error {
		_, err := client.DeleteCertificate(&deleteCertificateRequest)
		if err != nil {
			if cErr, ok := err.(uerr.ClientError); ok && cErr.Retryable() {
				return err
			}
			return backoff.Permanent(err)
		}
		return nil
	}
	reconnectBackoff := backoff.NewExponentialBackOff()
	reconnectBackoff.MaxElapsedTime = 30 * time.Second
	return backoff.Retry(deleteCertificate, reconnectBackoff)
}

func DeleteUnusedCertificate(client *ucdn.UCDNClient, name string) {
	cert := GetCertificates(client, name)
	if cert == nil || len(cert) == 0 {
		return
	}
	DeleteCertificate(client, name)
}
