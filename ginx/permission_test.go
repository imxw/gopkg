package ginx

import "testing"

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name     string
		perms    []string
		required string
		want     bool
	}{
		{"exact match", []string{"admin"}, "admin", true},
		{"no match", []string{"admin"}, "root", false},
		{"empty perms", nil, "admin", false},
		{"wildcard star", []string{"*"}, "anything", true},
		{"wildcard prefix", []string{"cmdb:*"}, "cmdb:read", true},
		{"wildcard prefix nested", []string{"cmdb:*"}, "cmdb:admin:delete", true},
		{"wildcard no match", []string{"cmdb:*"}, "sys:read", false},
		{"multiple perms match", []string{"sys:read", "cmdb:*"}, "cmdb:write", true},
		{"multiple perms no match", []string{"sys:read", "cmdb:*"}, "user:delete", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasPermission(tt.perms, tt.required); got != tt.want {
				t.Errorf("hasPermission(%v, %q) = %v, want %v", tt.perms, tt.required, got, tt.want)
			}
		})
	}
}
