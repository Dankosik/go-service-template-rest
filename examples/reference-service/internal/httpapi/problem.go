package httpapi

import "github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"

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
	case 404:
		return "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.5"
	default:
		return "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.1"
	}
}
