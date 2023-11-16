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
