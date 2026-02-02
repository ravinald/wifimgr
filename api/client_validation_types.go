package api

// SelfResponse represents the response from the /api/v1/self endpoint
type SelfResponse struct {
	ID         *string     `json:"id,omitempty"`
	Email      *string     `json:"email,omitempty"`
	FirstName  *string     `json:"first_name,omitempty"`
	LastName   *string     `json:"last_name,omitempty"`
	Name       string      `json:"name,omitempty"`
	Privileges []Privilege `json:"privileges,omitempty"`
}

// Privilege represents a user's privilege for an organization
type Privilege struct {
	Scope string `json:"scope"`
	Role  string `json:"role"`
	Name  string `json:"name"`
	OrgID string `json:"org_id"`
}
