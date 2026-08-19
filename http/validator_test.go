package http_test

import (
	"testing"

	ghttp "github.com/Hlgxz/gai/http"
)

func TestValidatorRules(t *testing.T) {
	v := ghttp.NewValidator(map[string]any{
		"email":                 "bad",
		"name":                  "A",
		"password":              "secret",
		"password_confirmation": "nope",
	}, map[string]string{
		"email":    "required|email",
		"name":     "required|min:2",
		"password": "required|confirmed",
	})
	errs := v.Validate()
	if errs == nil {
		t.Fatal("expected errors")
	}
	if errs.First("email") == "" || errs.First("name") == "" || errs.First("password") == "" {
		t.Fatalf("missing errors: %+v", errs)
	}

	ok := ghttp.NewValidator(map[string]any{
		"email": "a@b.com",
		"name":  "Ada",
		"age":   18,
	}, map[string]string{
		"email": "required|email",
		"name":  "required|min:2",
		"age":   "numeric|min:0",
	}).Validate()
	if ok != nil {
		t.Fatalf("unexpected: %v", ok)
	}
}

func TestValidateStruct(t *testing.T) {
	type in struct {
		Email string `json:"email" validate:"required|email"`
	}
	errs := ghttp.ValidateStruct(in{Email: "x"})
	if errs == nil {
		t.Fatal("expected error")
	}
}
