package model

import "testing"

func TestActivationRecord_LookupKey(t *testing.T) {
	tests := []struct {
		name   string
		record ActivationRecord
		want   string
	}{
		{
			name: "typical activation",
			record: ActivationRecord{
				Scope:            "/subscriptions/abc-123/resourceGroups/rg-prod",
				RoleDefinitionID: "/providers/Microsoft.Authorization/roleDefinitions/def-456",
			},
			want: "/subscriptions/abc-123/resourceGroups/rg-prod|/providers/Microsoft.Authorization/roleDefinitions/def-456",
		},
		{
			name: "empty fields",
			record: ActivationRecord{
				Scope:            "",
				RoleDefinitionID: "",
			},
			want: "|",
		},
		{
			name: "scope only",
			record: ActivationRecord{
				Scope:            "/subscriptions/abc-123",
				RoleDefinitionID: "",
			},
			want: "/subscriptions/abc-123|",
		},
		{
			name: "role definition only",
			record: ActivationRecord{
				Scope:            "",
				RoleDefinitionID: "/providers/Microsoft.Authorization/roleDefinitions/def-456",
			},
			want: "|/providers/Microsoft.Authorization/roleDefinitions/def-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.record.LookupKey()
			if got != tt.want {
				t.Errorf("LookupKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestActivationRecord_LookupKey_Uniqueness(t *testing.T) {
	a := ActivationRecord{
		Scope:            "/subscriptions/aaa",
		RoleDefinitionID: "/roles/111",
	}
	b := ActivationRecord{
		Scope:            "/subscriptions/aaa",
		RoleDefinitionID: "/roles/222",
	}
	c := ActivationRecord{
		Scope:            "/subscriptions/bbb",
		RoleDefinitionID: "/roles/111",
	}

	if a.LookupKey() == b.LookupKey() {
		t.Error("different RoleDefinitionID should produce different keys")
	}
	if a.LookupKey() == c.LookupKey() {
		t.Error("different Scope should produce different keys")
	}
}
