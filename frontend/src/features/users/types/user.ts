// Mirrors backend userResponse (services/user/internal/handler/http/v1/user_response.go)
// Fields: id, email (nullable), full_name, phone (nullable), status (string enum), created_at, updated_at
export type UserStatus = 'active' | 'suspended' | 'pending';

export interface AdminUser {
  id: string;
  email: string | null;
  full_name: string;
  phone: string | null;
  status: UserStatus;
  created_at: string;
  updated_at: string;
}

// Backwards-compat alias — prefer AdminUser for new code
export type User = AdminUser;
