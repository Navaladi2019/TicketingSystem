// loan_application schema from docs/handbook/examples/loan_application.json (TICK-789 seed),
// plus two fields the spike needs: `purpose` (oneOf of const) and `rejection_reason` (rule target).
import seed from '../../docs/handbook/examples/loan_application.json';
export const schema = structuredClone(seed.definition.schema);
schema.properties.purpose = {
  type: 'string', title: 'Purpose',
  oneOf: [{ const: 'home', title: 'Home purchase' }, { const: 'auto', title: 'Vehicle' }, { const: 'edu', title: 'Education' }],
};
schema.properties.rejection_reason = { type: 'string', title: 'Rejection reason' };
const ctl = (p, extra = {}) => ({ type: 'Control', scope: `#/properties/${p}`, ...extra });
export const uischema = {
  type: 'VerticalLayout',
  elements: [
    ctl('applicant_name'), ctl('amount'), ctl('term_months'), ctl('income'), ctl('purpose'), ctl('state'),
    ctl('rejection_reason', { rule: { effect: 'SHOW', condition: { scope: '#/properties/state', schema: { const: 'rejected' } } } }),
  ],
};
