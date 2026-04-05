package dto

import (
	"strings"
)

// MaskedBytes is a byte slice type that masks its content when serialized.
// This is used for sensitive data like passwords or secrets that should
// never be exposed in JSON responses or logs, even when accidentally serialized.
type MaskedBytes []byte

// MarshalJSON implements the json.Marshaler interface.
// It always returns a masked string regardless of the actual content,
// ensuring sensitive data is never accidentally exposed in JSON output.
func (m MaskedBytes) MarshalJSON() ([]byte, error) {
	chars := len(string(m))
	return []byte(`"` + strings.Repeat("*", chars) + `"`), nil
}

// String implements the fmt.Stringer interface.
// It returns a masked string to prevent sensitive data from being
// accidentally logged or displayed.
func (m MaskedBytes) String() string {
	chars := len(string(m))
	return strings.Repeat("*", chars)
}

type CollectionPermission struct {
	Search PermissionItem `json:"search"`
	Read   PermissionItem `json:"read"`
	Write  PermissionItem `json:"write"`
}

type PermissionItem struct {
	IsPublic bool `json:"is_public"`
}
