import { z } from 'zod';

// Match backend createUserRequest binding.
export const createUserSchema = z.object({
  email: z.string().email('Email không hợp lệ'),
  password: z.string().min(6, 'Mật khẩu tối thiểu 6 ký tự'),
  full_name: z.string().min(1, 'Họ tên không được trống'),
  phone: z.string().optional().default(''),
});

export const updateUserSchema = z.object({
  full_name: z.string().min(1).optional(),
  phone: z.string().optional(),
});

export type CreateUserInput = z.infer<typeof createUserSchema>;
export type UpdateUserInput = z.infer<typeof updateUserSchema>;
