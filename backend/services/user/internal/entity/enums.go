package entity

// UserStatus represents the lifecycle state of a user account.
type UserStatus string

const (
	UserStatusPending     UserStatus = "pending"
	UserStatusActive      UserStatus = "active"
	UserStatusSuspended   UserStatus = "suspended"
	UserStatusDeactivated UserStatus = "deactivated"
)

// Gender values for the users.gender column.
type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderOther  Gender = "other"
)

// IdentityProvider enumerates SSO and local auth providers.
type IdentityProvider string

const (
	IdentityProviderGoogle IdentityProvider = "google"
	IdentityProviderLocal  IdentityProvider = "local"
)

// ScopeType for role assignments — global or vendor-scoped.
type ScopeType string

const (
	ScopeTypeGlobal ScopeType = "global"
	ScopeTypeVendor ScopeType = "vendor"
)

// RoleInVendor for vendor_memberships.role_in_vendor.
type RoleInVendor string

const (
	RoleInVendorOwner RoleInVendor = "OWNER"
	RoleInVendorStaff RoleInVendor = "STAFF"
)

// VendorMembershipStatus for vendor_memberships.status.
type VendorMembershipStatus string

const (
	VendorMembershipInvited VendorMembershipStatus = "invited"
	VendorMembershipActive  VendorMembershipStatus = "active"
	VendorMembershipLeft    VendorMembershipStatus = "left"
)

// InvitationStatus for invitations.status.
type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusExpired  InvitationStatus = "expired"
	InvitationStatusRevoked  InvitationStatus = "revoked"
)

// OTPPurpose for otp_codes.purpose.
type OTPPurpose string

const (
	OTPPurposeLogin          OTPPurpose = "login"
	OTPPurposeRegisterVerify OTPPurpose = "register_verify"
	OTPPurposePasswordReset  OTPPurpose = "password_reset"
	OTPPurposePhoneVerify    OTPPurpose = "phone_verify"
)

// ShipperStatus for shipper_profiles.status (Admin approval lifecycle).
type ShipperStatus string

const (
	ShipperStatusPending  ShipperStatus = "pending"
	ShipperStatusApproved ShipperStatus = "approved"
	ShipperStatusRejected ShipperStatus = "rejected"
)

// OutboxStatus for outbox_events.status.
type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "pending"
	OutboxStatusPublished OutboxStatus = "published"
	OutboxStatusFailed    OutboxStatus = "failed"
)

// System role code constants — used in seed queries and tests only.
// Business logic must NOT hard-code role checks against these strings;
// use dynamic permission checks via role_permissions table instead.
const (
	RoleCodeSuperAdmin  = "SUPER_ADMIN"
	RoleCodeAdmin       = "ADMIN"
	RoleCodeCustomer    = "CUSTOMER"
	RoleCodeVendorOwner = "VENDOR_OWNER"
	RoleCodeVendorStaff = "VENDOR_STAFF"
	RoleCodeShipper     = "SHIPPER"
)
