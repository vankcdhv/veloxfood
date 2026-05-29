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
	IdentityProviderSchoolSSO IdentityProvider = "school_sso"
	IdentityProviderGoogle    IdentityProvider = "google"
	IdentityProviderLocal     IdentityProvider = "local"
	IdentityProviderGuest     IdentityProvider = "guest"
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
	RoleInVendorOwner   RoleInVendor = "OWNER"
	RoleInVendorManager RoleInVendor = "MANAGER"
	RoleInVendorKitchen RoleInVendor = "KITCHEN"
	RoleInVendorCashier RoleInVendor = "CASHIER"
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

// CardKind for card_identifiers.kind.
type CardKind string

const (
	CardKindRFID CardKind = "rfid"
	CardKindFace CardKind = "face"
)

// OTPPurpose for otp_codes.purpose.
type OTPPurpose string

const (
	OTPPurposeLogin           OTPPurpose = "login"
	OTPPurposeRegisterVerify  OTPPurpose = "register_verify"
	OTPPurposePasswordReset   OTPPurpose = "password_reset"
	OTPPurposePhoneVerify     OTPPurpose = "phone_verify"
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
	RoleCodeSuperAdmin        = "SUPER_ADMIN"
	RoleCodeSchoolAdmin       = "SCHOOL_ADMIN"
	RoleCodeVendorOwner       = "VENDOR_OWNER"
	RoleCodeVendorStaffKitchen = "VENDOR_STAFF_KITCHEN"
	RoleCodeVendorStaffCashier = "VENDOR_STAFF_CASHIER"
	RoleCodeStudent           = "STUDENT"
	RoleCodeFaculty           = "FACULTY"
	RoleCodeGuest             = "GUEST"
	RoleCodeShipper           = "SHIPPER"
)
