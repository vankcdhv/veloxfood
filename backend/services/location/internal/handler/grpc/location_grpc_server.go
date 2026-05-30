package grpc

import (
	"context"
	"errors"

	"project/pkg/apperror"
	locationv1 "project/proto/location/v1"
	"project/services/location/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LocationServiceServer implements the proto-generated gRPC LocationService.
type LocationServiceServer struct {
	locationv1.UnimplementedLocationServiceServer
	uc usecase.LocationUsecase
}

func NewLocationServiceServer(uc usecase.LocationUsecase) *LocationServiceServer {
	return &LocationServiceServer{uc: uc}
}

// GetRoom resolves a room to its full building/floor/room path.
func (s *LocationServiceServer) GetRoom(ctx context.Context, req *locationv1.GetRoomRequest) (*locationv1.GetRoomResponse, error) {
	resolved, err := s.uc.ResolveRoom(ctx, req.GetRoomId())
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) && appErr.Code == 404 {
			return &locationv1.GetRoomResponse{Found: false}, nil
		}
		return nil, status.Error(codes.Internal, "resolve room failed")
	}
	return &locationv1.GetRoomResponse{
		Found: true,
		Room: &locationv1.ResolvedRoom{
			RoomId:       resolved.Room.ID,
			RoomCode:     resolved.Room.Code,
			RoomName:     resolved.Room.Name,
			FloorId:      resolved.Floor.ID,
			FloorName:    resolved.Floor.Name,
			BuildingId:   resolved.Building.ID,
			BuildingName: resolved.Building.Name,
			Active:       resolved.Room.IsActive,
		},
	}, nil
}
