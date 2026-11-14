// Small SHOW/HIDE interpreter (ADR-007 D8): const/enum on one field, no Ajv compile.
// Shaped as the `ajv` object JSONForms core calls: ajv.validate(condition.schema, value).
export const ruleEvaluator = {
  validate(schema, value) {
    if ('const' in schema) return value === schema.const;
    if (Array.isArray(schema.enum)) return schema.enum.includes(value);
    return false;
  },
  compile() { return () => true; },
  errors: null,
};
