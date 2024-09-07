package httputil

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type RequestContext struct {
	context.Context
	Request  *http.Request
	Response http.ResponseWriter
}

func Context(r *http.Request, w http.ResponseWriter) RequestContext {
	return RequestContext{
		Context:  r.Context(),
		Request:  r,
		Response: w,
	}
}

func (r RequestContext) Encode(v interface{}) {
	r.Response.Header().Set("Content-Type", "application/json")
	// encode nil slices as [] and nil maps as {} (instead of null)
	if val := reflect.ValueOf(v); val.Kind() == reflect.Slice && val.Len() == 0 {
		_, _ = r.Response.Write([]byte("[]\n"))
		return
	} else if val.Kind() == reflect.Map && val.Len() == 0 {
		_, _ = r.Response.Write([]byte("{}\n"))
		return
	}
	enc := json.NewEncoder(r.Response)
	enc.SetIndent("", "\t")
	_ = enc.Encode(v)
}

func (r RequestContext) Decode(v interface{}) error {
	if err := json.NewDecoder(r.Request.Body).Decode(v); err != nil {
		return r.Error(fmt.Errorf("couldn't decode request type (%T): %w", v, err), http.StatusBadRequest)
	}
	return nil
}
func (r RequestContext) Error(err error, status int) error {
	http.Error(r.Response, err.Error(), status)
	return err
}
func (r RequestContext) Check(msg string, err error) error {
	if err != nil {
		return r.Error(fmt.Errorf("%v: %w", msg, err), http.StatusInternalServerError)
	}
	return nil
}
func (r RequestContext) DecodeForm(key string, v interface{}) error {
	value := r.Request.FormValue(key)
	if value == "" {
		return nil
	}
	var err error
	switch v := v.(type) {
	case interface{ UnmarshalText([]byte) error }:
		err = v.UnmarshalText([]byte(value))
	case interface{ LoadString(string) error }:
		err = v.LoadString(value)
	case *string:
		*v = value
	case *[]string:
		*v = strings.Split(value, ",")
	case *int:
		*v, err = strconv.Atoi(value)
	case *int64:
		*v, err = strconv.ParseInt(value, 10, 64)
	case *uint64:
		*v, err = strconv.ParseUint(value, 10, 64)
	case *bool:
		*v, err = strconv.ParseBool(value)
	default:
		panic(fmt.Sprintf("unsupported type %T", v))
	}
	if err != nil {
		return r.Error(fmt.Errorf("invalid form value %q: %w", key, err), http.StatusBadRequest)
	}
	return nil
}
