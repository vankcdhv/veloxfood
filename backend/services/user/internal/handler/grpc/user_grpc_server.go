package grpc

import (
	"context"
	"errors"
	"net/http"

	"project/pkg/apperror"
	userv1 "project/proto/user/v1"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"
	"project/services/user/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserServiceServer implements the proto-generated gRPC UserService.
type UserServiceServer struct {
	userv1.UnimplementedUserServiceServer
	userUC         usecase.UserUsecase
	userRepo       repository.UserRepository
	membershipRepo repository.VendorMembershipRepository
}

// NewUserServiceServer wires the gRPC server with required dependencies.
func NewUserServiceServer(
	userUC usecase.UserUsecase,
	userRepo repository.UserRepository,
	membershipRepo repository.VendorMembershipRepository,
) *UserServiceServer {
	return &UserServiceServer{
		userUC:         userUC,
		userRepo:       userRepo,
		membershipRepo: membershipRepo,
	}
}

func (s *UserServiceServer) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	user, err := s.userUC.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	return &userv1.GetUserResponse{User: toPBUser(user)}, nil
}

func (s *UserServiceServer) GetUserBatch(ctx context.Context, req *userv1.GetUserBatchRequest) (*userv1.GetUserBatchResponse, error) {
	users, err := s.userRepo.GetByIDs(ctx, req.GetIds())
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*userv1.User, 0, len(users))
	for _, u := range users {
		out = append(out, toPBUser(u))
	}
	return &userv1.GetUserBatchResponse{Users: out}, nil
}

func (s *UserServiceServer) GetVendorMembership(ctx context.Context, req *userv1.GetVendorMembershipRequest) (*userv1.GetVendorMembershipResponse, error) {
	m, err := s.membershipRepo.GetByUserAndVendor(ctx, req.GetUserId(), req.GetVendorId())
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) && appErr.Code == http.StatusNotFound {
			return &userv1.GetVendorMembershipResponse{Found: false}, nil
		}
		return nil, mapError(err)
	}
	return &userv1.GetVendorMembershipResponse{
		Found:        true,
		RoleInVendor: string(m.RoleInVendor),
		Status:       string(m.Status),
	}, nil
}

func toPBUser(u *entity.User) *userv1.User {
	pb := &userv1.User{
		Id:        u.ID,
		FullName:  u.FullName,
		Status:    string(u.Status),
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
	if u.Email != nil {
		pb.Email = *u.Email
	}
	if u.Phone != nil {
		pb.Phone = *u.Phone
	}
	return pb
}

func mapError(err error) error {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return status.Error(httpToGRPCCode(appErr.Code), appErr.Message)
	}
	return status.Error(codes.Internal, "internal error")
}

func httpToGRPCCode(httpCode int) codes.Code {
	switch httpCode {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	default:
		return codes.Internal
	}
}
