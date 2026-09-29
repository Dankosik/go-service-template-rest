package httpx

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"io"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// A JSON request body is otherwise validated by decoding it into map[string]any
// and walking the schema, and then decoded a second time into the generated type.
// For a schema that states only types, properties, required members, and
// additionalProperties, one streaming pass over the bytes decides the same
// question without building the intermediate value.
//
// The pass only ever admits. A body it does not accept — malformed, rejected, or
// merely outside what it understands — is handed to the full validator, which
// answers exactly as it would have without this path. Admission therefore has
// to imply that the full validator would also accept, and every rule below is
// written to be at least as strict as kin-openapi for the same keyword.

const (
	jsonMediaType = "application/json"
	// maxShapeDepth bounds schema compilation, which also stops a recursive $ref
	// from compiling forever; such a schema keeps the full validator.
	maxShapeDepth = 32
	// maxShapeProperties is the width of the required-member bitset.
	maxShapeProperties = 64
	// maxExactIntegerDigits keeps an admitted integer exactly representable as
	// the float64 kin-openapi converts it to, so its integer and format checks
	// cannot disagree with this pass.
	maxExactIntegerDigits = 15
)

type shapeKind uint8

const (
	shapeObject shapeKind = iota + 1
	shapeArray
	shapeString
	shapeInteger
	shapeNumber
	shapeBoolean
)

type bodyShape struct {
	kind  shapeKind
	int32 bool

	properties    []bodyProperty
	required      uint64
	rejectUnknown bool

	items *bodyShape
}

type bodyProperty struct {
	name  string
	shape *bodyShape
}

// structuralSchemaFields are the Schema fields a bodyShape represents or that
// never affect validation. Any other field set on a schema, including keywords
// a later kin-openapi release adds, keeps that body on the full validator.
var structuralSchemaFields = map[string]bool{
	"Extensions":           true,
	"Origin":               true,
	"Title":                true,
	"Description":          true,
	"Example":              true,
	"ExternalDocs":         true,
	"Deprecated":           true,
	"XML":                  true,
	"Type":                 true,
	"Format":               true,
	"Properties":           true,
	"Required":             true,
	"AdditionalProperties": true,
	"Items":                true,
}

func compileBodyShape(ref *openapi3.SchemaRef, depth int) (*bodyShape, bool) {
	if ref == nil || ref.Value == nil || depth > maxShapeDepth {
		return nil, false
	}
	schema := ref.Value
	if !hasOnlyStructuralFields(schema) {
		return nil, false
	}
	if schema.Type == nil || len(*schema.Type) != 1 {
		return nil, false
	}

	shape := &bodyShape{}
	switch (*schema.Type)[0] {
	case openapi3.TypeInteger:
		shape.kind = shapeInteger
		switch schema.Format {
		case "", "int64":
		case "int32":
			shape.int32 = true
		default:
			return nil, false
		}
		return shape, true
	case openapi3.TypeNumber:
		shape.kind = shapeNumber
	case openapi3.TypeString:
		shape.kind = shapeString
	case openapi3.TypeBoolean:
		shape.kind = shapeBoolean
	case openapi3.TypeArray:
		shape.kind = shapeArray
		items, ok := compileBodyShape(schema.Items, depth+1)
		if !ok {
			return nil, false
		}
		shape.items = items
	case openapi3.TypeObject:
		if !compileObjectShape(shape, schema, depth) {
			return nil, false
		}
	default:
		return nil, false
	}
	if schema.Format != "" {
		return nil, false
	}
	return shape, true
}

func hasOnlyStructuralFields(schema *openapi3.Schema) bool {
	fields := reflect.ValueOf(schema).Elem()
	for i := range fields.NumField() {
		if !structuralSchemaFields[fields.Type().Field(i).Name] && !fields.Field(i).IsZero() {
			return false
		}
	}
	return true
}

func compileObjectShape(shape *bodyShape, schema *openapi3.Schema, depth int) bool {
	shape.kind = shapeObject
	if schema.AdditionalProperties.Schema != nil || len(schema.Properties) > maxShapeProperties {
		return false
	}
	shape.rejectUnknown = schema.AdditionalProperties.Has != nil && !*schema.AdditionalProperties.Has
	for name, property := range schema.Properties {
		compiled, ok := compileBodyShape(property, depth+1)
		if !ok {
			return false
		}
		shape.properties = append(shape.properties, bodyProperty{name: name, shape: compiled})
	}
	for _, name := range schema.Required {
		index := shape.property(name)
		if index < 0 {
			return false
		}
		shape.required |= 1 << index
	}
	return true
}

func (s *bodyShape) property(name string) int {
	for i := range s.properties {
		if s.properties[i].name == name {
			return i
		}
	}
	return -1
}

// accepts reports whether data is exactly one JSON value that satisfies s.
// Duplicate member names and invalid UTF-8 are rejected by jsontext's defaults.
func (s *bodyShape) accepts(data []byte) bool {
	dec := jsontext.NewDecoder(bytes.NewBuffer(data))
	if !s.scan(dec) {
		return false
	}
	_, err := dec.ReadToken()
	return errors.Is(err, io.EOF)
}

func (s *bodyShape) scan(dec *jsontext.Decoder) bool {
	switch s.kind {
	case shapeObject:
		return s.scanObject(dec)
	case shapeArray:
		if dec.PeekKind() != '[' {
			return false
		}
		if _, err := dec.ReadToken(); err != nil {
			return false
		}
		for dec.PeekKind() != ']' {
			if !s.items.scan(dec) {
				return false
			}
		}
		_, err := dec.ReadToken()
		return err == nil
	case shapeString:
		return dec.PeekKind() == '"' && dec.SkipValue() == nil
	case shapeBoolean:
		kind := dec.PeekKind()
		return (kind == 't' || kind == 'f') && dec.SkipValue() == nil
	case shapeInteger:
		if dec.PeekKind() != '0' {
			return false
		}
		value, err := dec.ReadValue()
		return err == nil && s.exactInteger(value)
	case shapeNumber:
		if dec.PeekKind() != '0' {
			return false
		}
		value, err := dec.ReadValue()
		if err != nil {
			return false
		}
		_, err = strconv.ParseFloat(string(value), 64)
		return err == nil
	default:
		return false
	}
}

func (s *bodyShape) scanObject(dec *jsontext.Decoder) bool {
	if dec.PeekKind() != '{' {
		return false
	}
	if _, err := dec.ReadToken(); err != nil {
		return false
	}
	var seen uint64
	for dec.PeekKind() == '"' {
		name, err := dec.ReadValue()
		if err != nil {
			return false
		}
		index := s.property(memberName(name))
		if index < 0 {
			if s.rejectUnknown || dec.SkipValue() != nil {
				return false
			}
			continue
		}
		seen |= 1 << index
		if !s.properties[index].shape.scan(dec) {
			return false
		}
	}
	end, err := dec.ReadToken()
	return err == nil && end.Kind() == '}' && seen&s.required == s.required
}

// memberName returns the unquoted member name. An escape that cannot be
// unquoted yields a name no property has, and jsontext has already rejected
// such input by then anyway.
func memberName(quoted jsontext.Value) string {
	if bytes.IndexByte(quoted, '\\') < 0 {
		return string(quoted[1 : len(quoted)-1])
	}
	unquoted, err := jsontext.AppendUnquote(nil, quoted)
	if err != nil {
		return ""
	}
	return string(unquoted)
}

// exactInteger admits only a plain integer literal: a fraction or exponent that
// happens to be integral is left to the full validator, as is anything too long
// to stay exact through kin-openapi's float64 conversion.
func (s *bodyShape) exactInteger(literal jsontext.Value) bool {
	digits := literal
	negative := len(digits) > 0 && digits[0] == '-'
	if negative {
		digits = digits[1:]
	}
	if len(digits) == 0 || len(digits) > maxExactIntegerDigits {
		return false
	}
	var value int64
	for _, digit := range digits {
		if digit < '0' || digit > '9' {
			return false
		}
		value = value*10 + int64(digit-'0')
	}
	if negative {
		value = -value
	}
	return !s.int32 || (value >= math.MinInt32 && value <= math.MaxInt32)
}

// jsonRequestBody is one operation's body when its application/json schema
// compiles to a bodyShape.
type jsonRequestBody struct {
	content openapi3.Content
	media   *openapi3.MediaType
	shape   *bodyShape
}

// structuralRequestBodies returns the operations whose request body the
// streaming pass can decide. OpenAPI 3.1 documents are excluded: kin-openapi
// validates them with JSON Schema 2020 semantics this pass does not model.
func structuralRequestBodies(spec *openapi3.T) map[*openapi3.Operation]jsonRequestBody {
	if spec == nil || spec.Paths == nil || spec.IsOpenAPI31OrLater() {
		return nil
	}
	bodies := make(map[*openapi3.Operation]jsonRequestBody)
	for _, item := range spec.Paths.Map() {
		for _, operation := range item.Operations() {
			if operation.RequestBody == nil || operation.RequestBody.Value == nil {
				continue
			}
			content := operation.RequestBody.Value.Content
			media := content[jsonMediaType]
			if media == nil || len(media.Encoding) > 0 {
				continue
			}
			shape, ok := compileBodyShape(media.Schema, 0)
			if !ok {
				continue
			}
			bodies[operation] = jsonRequestBody{content: content, media: media, shape: shape}
		}
	}
	return bodies
}

// admits reads the request body and reports whether the streaming pass
// accepts it. Either way it leaves the body readable again from the start, as
// kin-openapi does, so the full validator or the generated handler sees the
// same bytes; a read failure is replayed so the full validator reports it.
func (b jsonRequestBody) admits(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	mediaType, _, _ := strings.Cut(contentType, ";")
	if strings.TrimSpace(mediaType) != jsonMediaType || b.content.Get(contentType) != b.media {
		return false
	}
	if r.Body == nil || r.Body == http.NoBody {
		return false
	}

	var buffer bytes.Buffer
	if r.ContentLength > 0 {
		buffer.Grow(int(r.ContentLength) + bytes.MinRead)
	}
	_, err := buffer.ReadFrom(r.Body)
	_ = r.Body.Close()
	data := buffer.Bytes()
	if err != nil {
		r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(data), errorReader{err: err}))
		return false
	}

	r.ContentLength = int64(len(data))
	r.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil }
	r.Body, _ = r.GetBody()
	return len(data) > 0 && b.shape.accepts(data)
}

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }
