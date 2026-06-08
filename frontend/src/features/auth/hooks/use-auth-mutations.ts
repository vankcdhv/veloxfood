'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  changePassword,
  forgotPassword,
  login,
  logout,
  register,
  resetPassword,
  updateMe,
  verifyRegister,
} from '../api/auth-api';
import { ME_QUERY_KEY } from './use-session';

// After any flow that mints cookies, refresh the session query so the UI
// reflects the authenticated user.
function useRefreshSessionOnSuccess() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: ME_QUERY_KEY });
}

export function useLogin() {
  const onSuccess = useRefreshSessionOnSuccess();
  return useMutation({ mutationFn: login, onSuccess });
}

export function useRegister() {
  return useMutation({ mutationFn: register });
}

export function useVerifyRegister() {
  const onSuccess = useRefreshSessionOnSuccess();
  return useMutation({ mutationFn: verifyRegister, onSuccess });
}

export function useForgotPassword() {
  return useMutation({ mutationFn: forgotPassword });
}

export function useResetPassword() {
  return useMutation({ mutationFn: resetPassword });
}

export function useUpdateProfile() {
  const onSuccess = useRefreshSessionOnSuccess();
  return useMutation({ mutationFn: updateMe, onSuccess });
}

export function useChangePassword() {
  return useMutation({ mutationFn: changePassword });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: logout,
    onSettled: () => {
      // Drop the cached session regardless of the network outcome.
      qc.setQueryData(ME_QUERY_KEY, undefined);
      qc.removeQueries({ queryKey: ME_QUERY_KEY });
    },
  });
}
