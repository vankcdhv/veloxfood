// Mirror backend services/user/internal/handler/http/v1/user_handler.go → userResponse
export interface User {
  id: string;
  email: string;
  full_name: string;
  phone: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}
