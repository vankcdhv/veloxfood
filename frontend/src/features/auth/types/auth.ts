// Mirrors the backend User Service auth + /me payloads.
// See backend services/user/internal/handler/http/v1/me_handler.go.

export type UserStatus = 'pending' | 'active' | 'suspended' | 'deactivated';
export type ScopeType = 'global' | 'vendor';
export type RoleInVendor = 'OWNER' | 'STAFF';
export type MembershipStatus = 'invited' | 'active' | 'left';

export interface SessionUser {
  id: string;
  email: string;
  full_name: string;
  phone: string;
  status: UserStatus;
  created_at: string;
  updated_at: string;
}

export interface Role {
  role_id: string;
  scope_type: ScopeType;
  scope_id: string | null;
  role_code: string;
}

export interface VendorMembership {
  id: string;
  user_id: string;
  vendor_id: string;
  role_in_vendor: RoleInVendor;
  status: MembershipStatus;
  invited_by: string | null;
  invited_at: string;
  joined_at: string | null;
}

export interface MeResponse {
  user: SessionUser;
  roles: Role[];
  vendor_memberships: VendorMembership[];
}

// Auth request/response payloads.
export interface LoginRequest {
  identifier: string;
  password: string;
}

export interface RegisterRequest {
  email?: string;
  phone?: string;
  password: string;
  full_name: string;
}

export interface RegisterResult {
  id: string;
  status: UserStatus;
}

export interface VerifyRegisterRequest {
  destination: string;
  code: string;
}

export interface TokenPairResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  issued_at: string;
}

export interface ForgotPasswordRequest {
  identifier: string;
}

export interface ResetPasswordRequest {
  token: string;
  new_password: string;
}

// PATCH /api/v1/me — self-service profile update (only round-trippable fields).
export interface UpdateMeRequest {
  full_name?: string;
}

// POST /api/v1/auth/change-password — user_id required by backend (reads from body).
export interface ChangePasswordRequest {
  user_id: string;
  old_password: string;
  new_password: string;
}
