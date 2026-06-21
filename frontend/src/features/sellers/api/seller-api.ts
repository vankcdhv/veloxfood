import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';

export interface SellerOnboardInput {
  vendor_name: string;
  business_type: string;
  address: string;
  phone: string;
}

export interface SellerOnboardResult {
  vendor_id: string;
  status: string;
}

// Submit a seller (vendor) onboarding application. The applicant becomes a
// pending vendor owner until an admin approves it.
export async function onboardSeller(input: SellerOnboardInput): Promise<SellerOnboardResult> {
  const res = await http.post<ApiResponse<SellerOnboardResult>>(
    `${API_PREFIX}/vendors/onboard`,
    input,
  );
  if (!res.data.data) {
    throw new Error(res.data.error ?? 'Onboard failed');
  }
  return res.data.data;
}
