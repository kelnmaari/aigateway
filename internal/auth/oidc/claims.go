// Package oidc provides OpenID Connect claims and user information structures
package oidc

import "time"

// StandardClaims represents standard OIDC claims from ID token
// Based on OpenID Connect Core 1.0 specification
type StandardClaims struct {
	// Subject (sub) - Identifier for the End-User at the Issuer
	Subject string `json:"sub"`

	// PreferredUsername (preferred_username) - Username by which the End-User wishes to be referred to
	PreferredUsername string `json:"preferred_username"`

	// Email (email) - End-User's preferred e-mail address
	Email string `json:"email"`

	// EmailVerified (email_verified) - True if the End-User's e-mail address has been verified
	EmailVerified bool `json:"email_verified"`

	// Name (name) - End-User's full name in displayable form
	Name string `json:"name"`

	// GivenName (given_name) - Given name(s) or first name(s) of the End-User
	GivenName string `json:"given_name"`

	// FamilyName (family_name) - Surname(s) or last name(s) of the End-User
	FamilyName string `json:"family_name"`

	// Picture (picture) - URL of the End-User's profile picture
	Picture string `json:"picture"`

	// Locale (locale) - End-User's locale, represented as a BCP47 [RFC5646] language tag
	Locale string `json:"locale"`

	// UpdatedAt (updated_at) - Time the End-User's information was last updated
	UpdatedAt *time.Time `json:"updated_at"`
}

// KeycloakClaims represents Keycloak-specific claims
// Including groups and realm roles
type KeycloakClaims struct {
	StandardClaims

	// Groups - User's groups from Keycloak
	Groups []string `json:"groups"`

	// RealmRoles - User's realm-level roles
	RealmRoles []string `json:"realm_roles"`

	// ResourceAccess - Client-level roles
	ResourceAccess map[string]ResourceAccessRoles `json:"resource_access"`

	// RealmAccess - Realm access with roles
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

// ResourceAccessRoles represents client-level roles in Keycloak
type ResourceAccessRoles struct {
	Roles []string `json:"roles"`
}

// GenericClaims represents a generic OIDC claims structure
// Can be used with any OIDC provider
type GenericClaims struct {
	StandardClaims

	// Custom claims - flexible map for provider-specific claims
	Custom map[string]interface{} `json:"-"`

	// Groups - can be in different formats depending on provider
	Groups []string `json:"groups,omitempty"`

	// Roles - can be in different formats depending on provider
	Roles []string `json:"roles,omitempty"`
}

// UserInfo represents processed user information from OIDC claims
// This is what we'll use to create/update users in our database
type UserInfo struct {
	// OIDC-specific fields
	Subject  string // OIDC 'sub' claim (unique identifier)
	Issuer   string // OIDC issuer URL
	Provider string // Provider name (keycloak, google, azure, etc.)

	// User information
	Username      string
	Email         string
	EmailVerified bool
	FullName      string
	GivenName     string
	FamilyName    string
	Picture       string
	Locale        string

	// Groups and roles
	Groups []string
	Roles  []string

	// Metadata
	UpdatedAt *time.Time
}


