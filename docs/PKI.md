DNS Records
===========


_ops_policy.platform.engineering TXT "base64 policy"
_ops_ca.platform.engineering TXT "base64 cert"
_ops_issuer.platform.engineering TXT "base64 cert"
_ops_u_serial.platform.engineering TXT "base64 cert"


DSL

```pkl
policy {
  version = 1
  rules {
    new {
      effect = allow
      action = distribute
      resource = "ops:repo:uri:hub.platform.engineering"
      condition {
        ["distributor"] = "platform.engineering"
      }
    }
  }
}
```

```pkl
policy {
  version = 1
  rules {
    new {
      effect = "allow"
      action = "distribute"
      resource = "ops:repo:uri:hub.platform.engineering"
      condition {
        ["publisher"] = "platform.engineering"
      }
    }
  }
}
```