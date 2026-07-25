package httpapi

import "github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"

// problem builds one of this example's generated Problem values.
//
// The status-to-type table is local because this example owns its own OpenAPI
// contract, and a feature package must not import the template's transport
// adapter to share one. It must still cover every status the handlers actually
// return: the version this replaced fell through to the 500 type URI for a 409,
// so a slug conflict advertised itself as an internal error.
func problem(code, title string, status int32, detail string) openapi.Problem {
	return openapi.Problem{
		Code:   code,
		Detail: &detail,
		Status: status,
		Title:  title,
		Type:   problemType(status),
	}
}

func problemType(status int32) string {
	switch status {
	case 400:
		return "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.1"
	case 401:
		return "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.2"
	case 404:
		return "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.5"
	case 409:
		return "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.10"
	default:
		return "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.1"
	}
}
