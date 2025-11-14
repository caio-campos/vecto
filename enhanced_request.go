package vecto

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// replacePathParams replaces path parameters in the URL with actual values.
// Supports both {key} and :key syntax.
func replacePathParams(urlStr string, params map[string]string) string {
	if len(params) == 0 {
		return urlStr
	}

	result := urlStr
	for key, value := range params {
		encodedValue := url.PathEscape(value)
		
		result = strings.ReplaceAll(result, "{"+key+"}", encodedValue)
		
		result = strings.ReplaceAll(result, ":"+key, encodedValue)
	}

	return result
}

// encodeFormData encodes form data as application/x-www-form-urlencoded.
func encodeFormData(data map[string]string) string {
	if len(data) == 0 {
		return ""
	}

	values := url.Values{}
	for key, value := range data {
		values.Set(key, value)
	}

	return values.Encode()
}

// structToQueryParams converts a struct to query parameters.
// Supports basic types and respects the `query` tag and `omitempty` option.
func structToQueryParams(v interface{}) (map[string]any, error) {
	if v == nil {
		return nil, nil
	}

	val := reflect.ValueOf(v)
	
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %v", val.Kind())
	}

	params := make(map[string]any)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanInterface() {
			continue
		}

		tag := fieldType.Tag.Get("query")
		if tag == "-" {
			continue
		}

		name, opts := parseTag(tag)
		if name == "" {
			name = fieldType.Name
		}

		omitEmpty := contains(opts, "omitempty")

		if omitEmpty && isZeroValue(field) {
			continue
		}

		value, err := fieldToQueryValue(field)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %w", fieldType.Name, err)
		}

		if value != nil {
			params[name] = value
		}
	}

	return params, nil
}

// parseTag parses a struct tag and returns the name and options.
func parseTag(tag string) (string, []string) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return "", nil
	}

	name := strings.TrimSpace(parts[0])
	opts := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		opts = append(opts, strings.TrimSpace(parts[i]))
	}

	return name, opts
}

// contains checks if a string slice contains a specific string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isZeroValue checks if a reflect.Value is the zero value for its type.
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// fieldToQueryValue converts a reflect.Value to a query parameter value.
func fieldToQueryValue(v reflect.Value) (interface{}, error) {
	switch v.Kind() {
	case reflect.String:
		return v.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), nil
	case reflect.Slice, reflect.Array:
		return sliceToQueryValues(v)
	case reflect.Ptr:
		if v.IsNil() {
			return nil, nil
		}
		return fieldToQueryValue(v.Elem())
	default:
		return nil, fmt.Errorf("unsupported type: %v", v.Kind())
	}
}

// sliceToQueryValues converts a slice to query parameter values.
// Returns a slice of strings for repeated parameters.
func sliceToQueryValues(v reflect.Value) (interface{}, error) {
	if v.Len() == 0 {
		return nil, nil
	}

	values := make([]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		val, err := fieldToQueryValue(elem)
		if err != nil {
			return nil, err
		}
		values[i] = fmt.Sprintf("%v", val)
	}

	return values, nil
}

// SetPathParam adds or updates a path parameter for the request.
func (r *Request) SetPathParam(key, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.params == nil {
		r.params = make(map[string]any)
	}
	
	pathParams, ok := r.params["__path_params__"].(map[string]string)
	if !ok {
		pathParams = make(map[string]string)
		r.params["__path_params__"] = pathParams
	}
	
	pathParams[key] = value
}

// SetPathParams sets multiple path parameters for the request.
func (r *Request) SetPathParams(params map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.params == nil {
		r.params = make(map[string]any)
	}
	
	pathParams := make(map[string]string, len(params))
	for k, v := range params {
		pathParams[k] = v
	}
	
	r.params["__path_params__"] = pathParams
}

