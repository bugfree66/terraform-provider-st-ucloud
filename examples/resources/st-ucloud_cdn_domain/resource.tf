resource "st-ucloud_cdn_domain" "test" {
  domain = "pgasia-cdn.com"
  test_url = "http://pgasia-cdn.com/"
  area_code = "cn"
  cdn_type = "web"

  origin_conf {
    origin_ip_list = ["8.8.8.8"]
    origin_host = "pgasia-cdn.com"
    origin_port = 80
    origin_protocol = "http"
    origin_follow301 = 0
  }

  cache_conf {
    cache_host = "pgasia-cdn.com"
    cache_rule {
      path_pattern = "/"
      ttl = 30
      cache_unit = "sec"
      cache_behavior = false
      follow_origin_rule = false
      description = ""
    }
  }

  access_control_conf {
  }

  advanced_conf {
  }

}
