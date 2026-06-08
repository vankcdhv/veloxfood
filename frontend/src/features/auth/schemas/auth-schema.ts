import { z } from 'zod';

const emailField = z.string().min(1, 'Vui lòng nhập email').email('Email không hợp lệ');

const strongPassword = z
  .string()
  .min(8, 'Mật khẩu tối thiểu 8 ký tự')
  .regex(/[A-Z]/, 'Cần ít nhất 1 chữ hoa')
  .regex(/[0-9]/, 'Cần ít nhất 1 chữ số');

// Login identifier can be an email OR a phone number (backend accepts both);
// only require non-empty here and let the server validate the credential.
export const loginSchema = z.object({
  email: z.string().min(1, 'Vui lòng nhập email hoặc số điện thoại'),
  password: z.string().min(1, 'Vui lòng nhập mật khẩu'),
  remember: z.boolean(),
});

export const registerSchema = z
  .object({
    full_name: z.string().min(1, 'Họ tên không được trống'),
    email: emailField,
    phone: z
      .string()
      .refine((v) => !v || /^[0-9+\s()-]{8,15}$/.test(v), 'Số điện thoại không hợp lệ'),
    password: strongPassword,
    confirm_password: z.string().min(1, 'Vui lòng xác nhận mật khẩu'),
    accept_terms: z.boolean().refine((v) => v === true, 'Bạn cần đồng ý điều khoản'),
  })
  .refine((d) => d.password === d.confirm_password, {
    path: ['confirm_password'],
    message: 'Mật khẩu xác nhận không khớp',
  });

export const forgotPasswordSchema = z.object({
  email: emailField,
});

// OTP verification after register — backend expects exactly 6 digits.
export const verifyOtpSchema = z.object({
  code: z
    .string()
    .min(1, 'Vui lòng nhập mã OTP')
    .regex(/^[0-9]{6}$/, 'Mã OTP gồm 6 chữ số'),
});

export const resetPasswordSchema = z
  .object({
    password: strongPassword,
    confirm_password: z.string().min(1, 'Vui lòng xác nhận mật khẩu'),
  })
  .refine((d) => d.password === d.confirm_password, {
    path: ['confirm_password'],
    message: 'Mật khẩu xác nhận không khớp',
  });

export const updateProfileSchema = z.object({
  full_name: z.string().min(1, 'Họ tên không được trống'),
});

export const changePasswordSchema = z
  .object({
    old_password: z.string().min(1, 'Vui lòng nhập mật khẩu hiện tại'),
    password: strongPassword,
    confirm_password: z.string().min(1, 'Vui lòng xác nhận mật khẩu'),
  })
  .refine((d) => d.password === d.confirm_password, {
    path: ['confirm_password'],
    message: 'Mật khẩu xác nhận không khớp',
  });

export type UpdateProfileInput = z.infer<typeof updateProfileSchema>;
export type ChangePasswordInput = z.infer<typeof changePasswordSchema>;

export type LoginInput = z.infer<typeof loginSchema>;
export type RegisterInput = z.infer<typeof registerSchema>;
export type ForgotPasswordInput = z.infer<typeof forgotPasswordSchema>;
export type ResetPasswordInput = z.infer<typeof resetPasswordSchema>;
export type VerifyOtpInput = z.infer<typeof verifyOtpSchema>;
