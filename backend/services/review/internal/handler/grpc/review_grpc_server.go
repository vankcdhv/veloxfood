package grpc

import (
	"context"

	reviewv1 "project/proto/review/v1"
	"project/services/review/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ReviewServiceServer implements the proto-generated gRPC ReviewService.
type ReviewServiceServer struct {
	reviewv1.UnimplementedReviewServiceServer
	reviewUC usecase.ReviewUsecase
}

func NewReviewServiceServer(reviewUC usecase.ReviewUsecase) *ReviewServiceServer {
	return &ReviewServiceServer{reviewUC: reviewUC}
}

// GetStoreRatingSummary returns avg rating + count of VISIBLE reviews for a store.
func (s *ReviewServiceServer) GetStoreRatingSummary(ctx context.Context, req *reviewv1.GetStoreRatingSummaryRequest) (*reviewv1.GetStoreRatingSummaryResponse, error) {
	if req.GetStoreId() == "" {
		return nil, status.Error(codes.InvalidArgument, "store_id is required")
	}
	avg, count, err := s.reviewUC.GetStoreRatingSummary(ctx, req.GetStoreId())
	if err != nil {
		return nil, status.Error(codes.Internal, "rating summary failed")
	}
	return &reviewv1.GetStoreRatingSummaryResponse{Avg: avg, Count: count}, nil
}
