// Mirrors review-service entities. Fields PascalCase (Go JSON default).
// Request bodies use snake_case.

export type ReviewTargetType = 'STORE' | 'MENU_ITEM' | 'SHIPPER';

export interface Review {
  ID: string;
  OrderID: string;
  ReviewerID: string;
  ReviewerName?: string;
  TargetType: ReviewTargetType;
  TargetID: string;
  Rating: number;
  Comment: string;
  PhotoURLs?: string[];
  Reply?: ReviewReply;
  CreatedAt: string;
  Hidden?: boolean;
}

export interface ReviewReply {
  AuthorID: string;
  Content: string;
  CreatedAt: string;
}

// POST /api/v1/orders/:id/reviews
export interface CreateReviewBody {
  target_type: ReviewTargetType;
  target_id: string;
  rating: number;
  comment: string;
  photo_urls?: string[];
}

// POST /api/v1/reviews/:id/reply
export interface ReplyReviewBody {
  content: string;
}

// POST /api/v1/reviews/:id/report
export interface ReportReviewBody {
  reason: string;
}

export interface ReviewListResponse {
  Items: Review[];
  Total: number;
}

// Admin reported reviews
export interface ReportedReview extends Review {
  ReportReason?: string;
  ReportedAt?: string;
}
