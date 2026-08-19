package rbac_test

import (
	"testing"

	"github.com/Hlgxz/gai/rbac"
)

func TestRBAC(t *testing.T) {
	m := rbac.New()
	m.Grant("admin", "users.delete")
	m.AssignRole(1, "admin")
	if !m.Can(1, "users.delete") {
		t.Fatal("admin should be allowed")
	}
	if m.Can(2, "users.delete") {
		t.Fatal("stranger should be denied")
	}
}
