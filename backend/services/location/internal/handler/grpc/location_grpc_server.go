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

// GetFloor resolves a floor to its parent building.
func (s *LocationServiceServer) GetFloor(ctx context.Context, req *locationv1.GetFloorRequest) (*locationv1.GetFloorResponse, error) {
	resolved, err := s.uc.ResolveFloor(ctx, req.GetFloorId())
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) && appErr.Code == 404 {
			return &locationv1.GetFloorResponse{Found: false}, nil
		}
		return nil, status.Error(codes.Internal, "resolve floor failed")
	}
	return &locationv1.GetFloorResponse{
		Found: true,
		Floor: &locationv1.ResolvedFloor{
			FloorId:      resolved.Floor.ID,
			FloorName:    resolved.Floor.Name,
			BuildingId:   resolved.Building.ID,
			BuildingName: resolved.Building.Name,
			Active:       !resolved.Floor.DeletedAt.Valid,
		},
	}, nil
}

// GetBuilding resolves a building by ID.
func (s *LocationServiceServer) GetBuilding(ctx context.Context, req *locationv1.GetBuildingRequest) (*locationv1.GetBuildingResponse, error) {
	resolved, err := s.uc.ResolveBuilding(ctx, req.GetBuildingId())
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) && appErr.Code == 404 {
			return &locationv1.GetBuildingResponse{Found: false}, nil
		}
		return nil, status.Error(codes.Internal, "resolve building failed")
	}
	return &locationv1.GetBuildingResponse{
		Found: true,
		Building: &locationv1.ResolvedBuilding{
			BuildingId:   resolved.Building.ID,
			BuildingName: resolved.Building.Name,
			Active:       resolved.Building.IsActive,
		},
	}, nil
}
