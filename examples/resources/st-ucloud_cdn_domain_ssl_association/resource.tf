resource "st-ucloud_cdn_domain_ssl_association" "test" {
  domain_id = st-ucloud_cdn_domain.test.domain_id
  ssl_certificate_name = st-ucloud_ssl_certificate.def.cert_name
}
