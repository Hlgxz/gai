package http

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// Cookie returns a request cookie value.
func (c *Context) Cookie(name string) (string, error) {
	ck, err := c.Request.Cookie(name)
	if err != nil {
		return "", err
	}
	return ck.Value, nil
}

// SetCookie writes a Set-Cookie header.
func (c *Context) SetCookie(cookie *http.Cookie) *Context {
	http.SetCookie(c.Writer, cookie)
	return c
}

// Bind fills dst from JSON, form, or query depending on Content-Type and method.
func (c *Context) Bind(dst any) error {
	if c.IsJSON() {
		return c.BindJSON(dst)
	}
	if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodDelete {
		return c.BindQuery(dst)
	}
	if err := c.Request.ParseForm(); err != nil {
		return err
	}
	return valuesToStruct(c.Request.Form, dst)
}

// BindQuery maps query-string parameters onto dst.
func (c *Context) BindQuery(dst any) error {
	return valuesToStruct(c.Request.URL.Query(), dst)
}

// BindForm maps form fields onto dst.
func (c *Context) BindForm(dst any) error {
	if err := c.Request.ParseForm(); err != nil {
		return err
	}
	return valuesToStruct(c.Request.PostForm, dst)
}

func valuesToStruct(values url.Values, dst any) error {
	m := make(map[string]any, len(values))
	for k, vs := range values {
		if len(vs) == 1 {
			m[k] = vs[0]
		} else {
			m[k] = vs
		}
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dst)
}

// Attachment serves a file as a download with Content-Disposition.
func (c *Context) Attachment(path, filename string) {
	if filename == "" {
		filename = filepath.Base(path)
	}
	c.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.ServeFile(c.Writer, c.Request, path)
	c.written = true
}

// Data writes raw bytes with the given content type.
func (c *Context) Data(code int, contentType string, data []byte) {
	c.SetHeader("Content-Type", contentType)
	c.Writer.WriteHeader(code)
	c.written = true
	_, _ = c.Writer.Write(data)
}

// Stream copies r to the response body.
func (c *Context) Stream(code int, contentType string, r io.Reader) {
	c.SetHeader("Content-Type", contentType)
	c.Writer.WriteHeader(code)
	c.written = true
	_, _ = io.Copy(c.Writer, r)
}

// SaveUploadedFile stores an uploaded file at dest.
func SaveUploadedFile(fh *multipart.FileHeader, dest string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

// ValidateStruct validates a struct using `validate` tags (pipe syntax).
func ValidateStruct(s any) ValidationErrors {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ValidationErrors{"_": {"validate target must be a struct"}}
	}
	t := v.Type()
	data := make(map[string]any)
	rules := make(map[string]string)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		rule := f.Tag.Get("validate")
		if rule == "" {
			continue
		}
		name := f.Tag.Get("json")
		if name == "" || name == "-" {
			name = strings.ToLower(f.Name)
		} else {
			name = strings.Split(name, ",")[0]
		}
		data[name] = v.Field(i).Interface()
		rules[name] = rule
	}
	return NewValidator(data, rules).Validate()
}

// ExpireCookie removes a cookie immediately.
func (c *Context) ExpireCookie(name string) *Context {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
	return c
}
