// Mirrors backend services/user shipper_profiles + admin list payload.

export type ShipperStatus = 'pending' | 'approved' | 'rejected';

export interface ShipperProfile {
  user_id: string;
  id_document_photo_url: string;
  portrait_photo_url: string;
  status: ShipperStatus;
  approved_by: string | null;
  approved_at: string | null;
  created_at: string;
  updated_at: string;
  /** Applicant's display name — populated by the enriched admin list endpoint. */
  full_name?: string;
  /** Applicant's email — populated by the enriched admin list endpoint. */
  email?: string | null;
}

export interface ShipperListResponse {
  items: ShipperProfile[];
  total: number;
  page: number;
  page_size: number;
}
