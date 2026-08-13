package domain

import "time"

const (
	CredentialKindConsoleAdmin = "console_admin"
	CredentialSlotActive       = "active"
	CredentialSlotStaged       = "staged"
)

type NodeCredentialMetadata struct {
	NodeID         ID        `json:"node_id"`
	LaboratoryID   ID        `json:"laboratory_id"`
	Kind           string    `json:"kind"`
	Configured     bool      `json:"configured"`
	Staged         bool      `json:"staged"`
	State          string    `json:"state"`
	Revision       Revision  `json:"revision"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	RotatedAt      time.Time `json:"last_rotated_at,omitempty"`
	LastVerifiedAt time.Time `json:"last_verified_at,omitempty"`
	LastErrorCode  string    `json:"last_error_code,omitempty"`
}

type NodeCredentialSecret struct {
	Username string
	Password []byte
}

func (s *NodeCredentialSecret) Clear() {
	for index := range s.Password {
		s.Password[index] = 0
	}
	s.Username = ""
	s.Password = nil
}
