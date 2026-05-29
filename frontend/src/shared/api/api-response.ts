// Mirror backend pkg/response envelope.
// See project/backend/pkg/response/response.go
export interface ApiResponse<T = unknown> {
  status: number;
  message: string;
  data?: T;
  error?: string;
}

export interface PaginatedData<T> {
  items: T[];
  total: number;
  page: number;
}

export type PaginatedResponse<T> = ApiResponse<PaginatedData<T>>;
