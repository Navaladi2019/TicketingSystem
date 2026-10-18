package apihttp

import (
	"errors"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// maxFieldErrors bounds the errors array of one response.
const maxFieldErrors = 20

// fieldErrors names the failing fields of a validation error. It reads only
// what the contract itself defines: the parameter name and location, the
// schema keyword that failed (SchemaField) and the JSON Pointer of the failing
// value (JSONPointer). It never reads SchemaError.Value, SchemaError.Reason,
// ParseError.Value or any Error() text, because those carry the input.
//
// A pointer token comes from the request body's own keys where the schema
// allows free-form keys; keep client-keyed maps out of secret-bearing schemas.
func fieldErrors(err error) []problem.FieldError {
	var out []problem.FieldError
	add := func(f problem.FieldError) {
		if len(out) < maxFieldErrors {
			out = append(out, f)
		}
	}
	collect(err, add)
	return out
}

func collect(err error, add func(problem.FieldError)) {
	switch e := err.(type) { //nolint:errorlint // MultiError.As would skip siblings; the direct type is what we want
	case openapi3.MultiError:
		for _, inner := range e {
			collect(inner, add)
		}
	case *openapi3filter.RequestError:
		add(requestField(e))
	default:
		// The generated wrapper binds parameters before the validator runs; its
		// errors name the parameter in a ParamName field and carry the input
		// in their text, so only the name is read.
		if name, rule, ok := paramName(err); ok {
			add(problem.FieldError{Pointer: "/parameters/" + escape(name), Rule: rule})
		}
	}
}

// paramName returns ParamName from the generated parameter errors
// (InvalidParamFormatError, RequiredParamError and friends) and the rule they
// stand for: "required" for the Required* types, "format" otherwise. Each
// generated package has its own copy of those types, so the field is read by
// reflection.
func paramName(err error) (name, rule string, ok bool) {
	for ; err != nil; err = errors.Unwrap(err) {
		v := reflect.Indirect(reflect.ValueOf(err))
		if v.Kind() != reflect.Struct {
			continue
		}
		if f := v.FieldByName("ParamName"); f.IsValid() && f.Kind() == reflect.String {
			rule = "format"
			if strings.HasPrefix(v.Type().Name(), "Required") {
				rule = "required"
			}
			return f.String(), rule, true
		}
	}
	return "", "", false
}

func requestField(req *openapi3filter.RequestError) problem.FieldError {
	base := "/body"
	if req.Parameter != nil {
		base = "/parameters/" + escape(req.Parameter.Name)
	}
	var schemaErr *openapi3.SchemaError
	var parseErr *openapi3filter.ParseError
	switch {
	case errors.As(req, &schemaErr):
		return problem.FieldError{Pointer: base + pointer(schemaErr.JSONPointer()), Rule: schemaErr.SchemaField}
	case errors.As(req, &parseErr):
		return problem.FieldError{Pointer: base, Rule: "format"}
	}
	return problem.FieldError{Pointer: base}
}

func pointer(tokens []string) string {
	var b strings.Builder
	for _, t := range tokens {
		b.WriteByte('/')
		b.WriteString(escape(t))
	}
	return b.String()
}

// escape applies RFC 6901 escaping.
func escape(s string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(s)
}
