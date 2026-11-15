mode=cors-inner
| engine | case | method | result | status | code | Retry-After | error |
|---|---|---|---|---|---|---|---|
| chromium | 200 | GET | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 200 | POST | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 200 | PATCH | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 401 | GET | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 401 | POST | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 401 | PATCH | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 403 origin_not_allowed | GET | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 403 origin_not_allowed | POST | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 403 origin_not_allowed | PATCH | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 403 client_not_enabled | GET | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 403 client_not_enabled | POST | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 403 client_not_enabled | PATCH | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 429 | GET | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 429 | POST | FAIL | - | - | - | TypeError: Failed to fetch |
| chromium | 429 | PATCH | FAIL | - | - | - | TypeError: Failed to fetch |
| firefox | 200 | GET | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 200 | POST | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 200 | PATCH | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 401 | GET | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 401 | POST | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 401 | PATCH | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 403 origin_not_allowed | GET | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 403 origin_not_allowed | POST | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 403 origin_not_allowed | PATCH | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 403 client_not_enabled | GET | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 403 client_not_enabled | POST | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 403 client_not_enabled | PATCH | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 429 | GET | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 429 | POST | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| firefox | 429 | PATCH | FAIL | - | - | - | TypeError: NetworkError when attempting to fetch resource. |
| webkit | 200 | GET | FAIL | - | - | - | TypeError: Load failed |
| webkit | 200 | POST | FAIL | - | - | - | TypeError: Load failed |
| webkit | 200 | PATCH | FAIL | - | - | - | TypeError: Load failed |
| webkit | 401 | GET | FAIL | - | - | - | TypeError: Load failed |
| webkit | 401 | POST | FAIL | - | - | - | TypeError: Load failed |
| webkit | 401 | PATCH | FAIL | - | - | - | TypeError: Load failed |
| webkit | 403 origin_not_allowed | GET | FAIL | - | - | - | TypeError: Load failed |
| webkit | 403 origin_not_allowed | POST | FAIL | - | - | - | TypeError: Load failed |
| webkit | 403 origin_not_allowed | PATCH | FAIL | - | - | - | TypeError: Load failed |
| webkit | 403 client_not_enabled | GET | FAIL | - | - | - | TypeError: Load failed |
| webkit | 403 client_not_enabled | POST | FAIL | - | - | - | TypeError: Load failed |
| webkit | 403 client_not_enabled | PATCH | FAIL | - | - | - | TypeError: Load failed |
| webkit | 429 | GET | FAIL | - | - | - | TypeError: Load failed |
| webkit | 429 | POST | FAIL | - | - | - | TypeError: Load failed |
| webkit | 429 | PATCH | FAIL | - | - | - | TypeError: Load failed |

0/45 pass
