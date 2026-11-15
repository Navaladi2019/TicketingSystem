mode=cors-outer
| engine | case | method | result | status | code | Retry-After | error |
|---|---|---|---|---|---|---|---|
| chromium | 200 | GET | PASS | 200 | - | - | - |
| chromium | 200 | POST | PASS | 201 | - | - | - |
| chromium | 200 | PATCH | PASS | 200 | - | - | - |
| chromium | 401 | GET | PASS | 401 | unauthenticated | - | - |
| chromium | 401 | POST | PASS | 401 | unauthenticated | - | - |
| chromium | 401 | PATCH | PASS | 401 | unauthenticated | - | - |
| chromium | 403 origin_not_allowed | GET | PASS | 403 | origin_not_allowed | - | - |
| chromium | 403 origin_not_allowed | POST | PASS | 403 | origin_not_allowed | - | - |
| chromium | 403 origin_not_allowed | PATCH | PASS | 403 | origin_not_allowed | - | - |
| chromium | 403 client_not_enabled | GET | PASS | 403 | client_not_enabled | - | - |
| chromium | 403 client_not_enabled | POST | PASS | 403 | client_not_enabled | - | - |
| chromium | 403 client_not_enabled | PATCH | PASS | 403 | client_not_enabled | - | - |
| chromium | 429 | GET | PASS | 429 | rate_limited | 7 | - |
| chromium | 429 | POST | PASS | 429 | rate_limited | 7 | - |
| chromium | 429 | PATCH | PASS | 429 | rate_limited | 7 | - |
| firefox | 200 | GET | PASS | 200 | - | - | - |
| firefox | 200 | POST | PASS | 201 | - | - | - |
| firefox | 200 | PATCH | PASS | 200 | - | - | - |
| firefox | 401 | GET | PASS | 401 | unauthenticated | - | - |
| firefox | 401 | POST | PASS | 401 | unauthenticated | - | - |
| firefox | 401 | PATCH | PASS | 401 | unauthenticated | - | - |
| firefox | 403 origin_not_allowed | GET | PASS | 403 | origin_not_allowed | - | - |
| firefox | 403 origin_not_allowed | POST | PASS | 403 | origin_not_allowed | - | - |
| firefox | 403 origin_not_allowed | PATCH | PASS | 403 | origin_not_allowed | - | - |
| firefox | 403 client_not_enabled | GET | PASS | 403 | client_not_enabled | - | - |
| firefox | 403 client_not_enabled | POST | PASS | 403 | client_not_enabled | - | - |
| firefox | 403 client_not_enabled | PATCH | PASS | 403 | client_not_enabled | - | - |
| firefox | 429 | GET | PASS | 429 | rate_limited | 7 | - |
| firefox | 429 | POST | PASS | 429 | rate_limited | 7 | - |
| firefox | 429 | PATCH | PASS | 429 | rate_limited | 7 | - |
| webkit | 200 | GET | PASS | 200 | - | - | - |
| webkit | 200 | POST | PASS | 201 | - | - | - |
| webkit | 200 | PATCH | PASS | 200 | - | - | - |
| webkit | 401 | GET | PASS | 401 | unauthenticated | - | - |
| webkit | 401 | POST | PASS | 401 | unauthenticated | - | - |
| webkit | 401 | PATCH | PASS | 401 | unauthenticated | - | - |
| webkit | 403 origin_not_allowed | GET | PASS | 403 | origin_not_allowed | - | - |
| webkit | 403 origin_not_allowed | POST | PASS | 403 | origin_not_allowed | - | - |
| webkit | 403 origin_not_allowed | PATCH | PASS | 403 | origin_not_allowed | - | - |
| webkit | 403 client_not_enabled | GET | PASS | 403 | client_not_enabled | - | - |
| webkit | 403 client_not_enabled | POST | PASS | 403 | client_not_enabled | - | - |
| webkit | 403 client_not_enabled | PATCH | PASS | 403 | client_not_enabled | - | - |
| webkit | 429 | GET | PASS | 429 | rate_limited | 7 | - |
| webkit | 429 | POST | PASS | 429 | rate_limited | 7 | - |
| webkit | 429 | PATCH | PASS | 429 | rate_limited | 7 | - |

45/45 pass
