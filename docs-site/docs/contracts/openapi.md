---
title: REST API
---

The API Gateway's initial REST contract is available as an
[OpenAPI 3.1 document](/openapi/openapi.yaml).

## Local testing

Start the gateway and, for authenticated routes, obtain a token from the
identity service:

```bash
export DOC_LENS_TOKEN="$(curl -s http://localhost:8080/identity/login \
  -H 'content-type: application/json' \
  -d '{"email":"admin@doclens.local","password":"doclens-dev"}' | jq -r .access_token)"
```

Then call an endpoint:

```bash
curl -s http://localhost:8080/documents \
  -H "Authorization: Bearer $DOC_LENS_TOKEN" \
  -H 'content-type: application/json' \
  -d '{"type":"certificate","filename":"certificate.pdf","content_type":"application/pdf","content_base64":"JVBERi0="}'
```

The OpenAPI document is the contract source for the public gateway surface.
Keep examples and schemas synchronized with the gateway handlers as endpoints
are added.
