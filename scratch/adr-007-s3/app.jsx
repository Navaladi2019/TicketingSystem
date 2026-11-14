import React from 'react';
import { createRoot } from 'react-dom/client';
import { JsonForms } from '@jsonforms/react';
import { vanillaRenderers, vanillaCells } from '@jsonforms/vanilla-renderers';
import { schema, uischema } from './schema.js';
import { ruleEvaluator } from './rules.js';

const attempt = new URLSearchParams(location.search).get('attempt') || '1';
const css = `
  :host { display:block; font: 14px system-ui; }
  [hidden] { display:none !important; }
  .control { margin: .5rem 0; display:flex; flex-direction:column; }
  select, input { padding: .25rem; border: 2px solid rgb(0, 128, 0); }
`;

class NavaRecords extends HTMLElement {
  connectedCallback() {
    const root = this.attachShadow({ mode: 'open' });
    const sheet = new CSSStyleSheet();
    sheet.replaceSync(css);
    root.adoptedStyleSheets = [sheet];
    const mount = document.createElement('div');
    root.append(mount);
    const props = {
      schema, uischema, data: {}, renderers: vanillaRenderers, cells: vanillaCells,
      i18n: { locale: 'en', translate: (_id, def) => def }, onChange: ({ data }) => { window.__data = data; },
    };
    if (attempt === '1') props.validationMode = 'NoValidation';           // stock ajv, validation off
    if (attempt === '2') { props.validationMode = 'NoValidation'; props.ajv = ruleEvaluator; } // custom rule evaluator
    createRoot(mount).render(<JsonForms {...props} />);
  }
}
customElements.define('nava-records', NavaRecords);
