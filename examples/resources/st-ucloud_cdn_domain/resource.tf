resource "st-ucloud_cdn_domain" "test" {
  domain    = "test.pgasia-cdn.com"
  test_url  = "http://origin.pgasia-cdn.com/"
  area_code = "cn"
  cdn_type  = "web"

  origin_conf {
    origin_ip_list   = ["origin-ws-cn-7z8567axjz.sige-test3.com"]
    origin_host      = "pgasia-cdn.com"
    origin_port      = 80
    origin_protocol  = "https"
    origin_follow301 = true
  }

  cache_conf {
    cache_rule {
      path_pattern       = "/"
      description        = "test"
      ttl                = 60
      cache_unit         = "sec"
      cache_behavior     = true
      follow_origin_rule = true
    }

    cache_rule {
      path_pattern       = ".*"
      description        = "test2"
      ttl                = 60
      cache_unit         = "sec"
      cache_behavior     = true
      follow_origin_rule = true
    }

    http_code_cache_rule {
      path_pattern       = ".*"
      description        = "test"
      ttl                = 60
      cache_unit         = "sec"
      cache_behavior     = true
      follow_origin_rule = true
      http_code          = 400
      use_regex          = false
    }

    http_code_cache_rule {
      path_pattern       = ".*"
      description        = "test"
      ttl                = 60
      cache_unit         = "sec"
      cache_behavior     = true
      follow_origin_rule = true
      http_code          = 401
      use_regex          = false
    }
  }

  access_control_conf = {
    enable_refer = false
    ip_blacklist = ["100.100.100.100", "4.4.4.4"]
    refer_conf = {
      null_refer = true
      refer_list = ["sige-test3.com"]
      refer_type = "blacklist"
    }
  }

  advanced_conf = {
    http_client_header_list = ["Test:test_client"]
    http_origin_header_list = ["Test:test_origin"]
    http_to_https           = false
  }
}
