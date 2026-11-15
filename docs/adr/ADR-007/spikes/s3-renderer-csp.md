# ADR-007 spike S3: renderer in a shadow root under strict CSP

Issue TICK-808. Pass conditions fixed by ADR-007 D8 and RFC-017 §7. Code: `scratch/adr-007-s3/` (`npm i`, `npx playwright install chromium firefox webkit`, `npm run build`, `node run.mjs 1,2`). Raw output of the last run: `scratch/adr-007-s3/run-output.txt`.

## Result: SPIKE: PASS

**Renderer for S26: JSONForms 3.8.0 (`@jsonforms/core`, `@jsonforms/react`, `@jsonforms/vanilla-renderers`) with `validationMode: 'NoValidation'`, a custom rule evaluator passed as the `ajv` prop, and styles through `adoptedStyleSheets`.** The hand-written renderer fallback is not needed.

Bundle (esbuild, minified, IIFE, React 19 + the three JSONForms packages + schema): **487,319 B raw, 155,020 B gzip.** This is the spike page, so it is an upper bound for the widget; S26 should re-measure.

## Setup

- Custom element `<nava-records>` with an open shadow root; React renders into a div inside it.
- Server headers: `Content-Security-Policy: script-src 'self'; style-src 'self'`. No inline script or style; the violation collector is a same-origin `probe.js`.
- Data: the `loan_application` schema from `docs/handbook/examples/loan_application.json` (TICK-789 seed), plus `purpose` (`oneOf` of `const` + `title`) and `rejection_reason`, shown by one SHOW rule when `state` is `rejected`.
- Playwright 1.65, Chromium, Firefox, WebKit. Listens for `securitypolicyviolation`, page errors and console errors.

## Attempts

| # | Renderer | Chromium | Firefox | WebKit |
|---|---|---|---|---|
| 1 | JSONForms, validation off, stock Ajv | **FAIL** | **FAIL** | **FAIL** |
| 2 | JSONForms, validation off, custom rule evaluator, `adoptedStyleSheets` | **PASS** | **PASS** | **PASS** |

**Attempt 1 violations (identical in all three engines):** two `securitypolicyviolation` events, `violatedDirective: script-src`, `blockedURI: eval`; page error `EvalError` ("Evaluating a string as JavaScript violates ... 'unsafe-eval'"; Firefox: "call to Function() blocked by CSP"; WebKit: "Refused to evaluate a string as JavaScript"). Cause: JSONForms core creates its own Ajv and compiles a schema even with validation off (Ajv meta-schema compile), and the SHOW rule calls `ajv.validate(...)`. Nothing rendered.

**Attempt 2:** `rules.js` is the D8 interpreter: `const` or `enum` on one field, no compile. It is passed as the `ajv` prop, so core never constructs Ajv. Zero violations, zero page errors, zero console errors in all three engines.

## Pass conditions (attempt 2)

| Condition | Chromium | Firefox | WebKit |
|---|---|---|---|
| Zero CSP violations | yes | yes | yes |
| Rule toggles the field (hidden, then shown on `rejected`, hidden on `approved`) | yes | yes | yes |
| `oneOf`/`const` field shows `title` labels (Home purchase / Vehicle / Education) | yes | yes | yes |
| Select styled from the adopted sheet (computed border `rgb(0, 128, 0)`) | yes | yes | yes |
| Nothing added to `document.body` besides the host and scripts | yes | yes | yes |

## Findings S26 must carry

1. **Do not let JSONForms build Ajv.** Pass the evaluator via `ajv` and set `validationMode: 'NoValidation'`. Core reads only `ajv.validate(schema, value)` for rules; `compile()` is stubbed.
2. **Select menus are native `<select>`** in the vanilla renderer set. There is no JS menu, so nothing portals to `document.body` and the "portal into the shadow root" condition holds trivially. The open native popup is drawn by the browser and cannot be styled; if the design needs a styled menu, that is a custom renderer inside the shadow root (D8 portal rule), not a spike failure. I did not test a custom menu.
3. **Hidden controls stay in the DOM with `hidden=""`.** The adopted sheet must include `[hidden]{display:none !important}`, or a `.control{display:flex}` rule shows hidden fields (hit and fixed during the spike).
4. **Enum labels need an `i18n.translate` returning the default.** Options render via the `label` attribute.
5. Vanilla renderers ship no CSS; the widget owns its whole stylesheet.
6. Only SHOW/HIDE const/enum rules were exercised, matching D8. `oneOf` of rules (AND/OR) goes through the same `validate` call but was not tested.

CI-E9 can reuse `run.mjs` as the regression test.
