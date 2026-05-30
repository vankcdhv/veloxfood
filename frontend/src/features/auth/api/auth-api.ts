import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type {
  ForgotPasswordRequest,
  LoginRequest,
  MeResponse,
  RegisterRequest,
  RegisterResult,
  ResetPasswordRequest,
  TokenPairResponse,
  VerifyRegisterRequest,
} from '../types/auth';

const AUTH_PATH = `${API_PREFIX}/auth`;

// Tokens are delivered as httpOnly cookies; the JSON body is ignored client-side.
export async function login(payload: LoginRequest): Promise<void> {
  await http.post<ApiResponse<TokenPairResponse>>(`${AUTH_PATH}/login`, payload);
}

export async function register(payload: RegisterRequest): Promise<RegisterResult> {
  const res = await http.post<ApiResponse<RegisterResult>>(`${AUTH_PATH}/register`, payload);
  if (!res.data.data) throw new Error(res.data.error ?? 'Đăng ký thất bại');
  return res.data.data;
}

export async function verifyRegister(payload: VerifyRegisterRequest): Promise<void> {
  await http.post<ApiResponse<TokenPairResponse>>(`${AUTH_PATH}/verify-register`, payload);
}

export async function forgotPassword(payload: ForgotPasswordRequest): Promise<void> {
  await http.post<ApiResponse>(`${AUTH_PATH}/forgot-password`, payload);
}

export async function resetPassword(payload: ResetPasswordRequest): Promise<void> {
  await http.post<ApiResponse>(`${AUTH_PATH}/reset-password`, payload);
}

export async function logout(): Promise<void> {
  // Backend reads the refresh/access tokens from cookies and clears them.
  await http.post<ApiResponse>(`${AUTH_PATH}/logout`, {});
}

export async function getMe(): Promise<MeResponse> {
  const res = await http.get<ApiResponse<MeResponse>>(`${API_PREFIX}/me`);
  if (!res.data.data) throw new Error(res.data.error ?? 'Không tải được phiên');
  return res.data.data;
}

// loginWithGoogle performs a full-page redirect to the backend OAuth entrypoint.
// The backend redirects to Google, then to its callback which sets the auth
// cookies and returns the browser to the web app.
export function loginWithGoogle(): void {
  window.location.href = `${AUTH_PATH}/google/login`;
}
