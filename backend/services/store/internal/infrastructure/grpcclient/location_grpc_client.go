package grpcclient

import (
	"context"
	"fmt"

	"project/pkg/grpcx"
	locationv1 "project/proto/location/v1"
)

// LocationClient implements usecase.LocationResolver by calling the location
// service over gRPC.
type LocationClient struct {
	client locationv1.LocationServiceClient
}

// NewLocationClient dials the location service and returns a LocationClient.
// addr is typically cfg.LocationService.GRPCAddr.
func NewLocationClient(addr string) (*LocationClient, error) {
	conn, err := grpcx.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("location grpc client: dial %s: %w", addr, err)
	}
	return &LocationClient{client: locationv1.NewLocationServiceClient(conn)}, nil
}

// GetRoom resolves roomID to its parent floorID and buildingID.
// Returns (floorID, buildingID, found=true, nil) on success;
// ("", "", false, nil) when not found; ("", "", false, err) on transport errors.
func (c *LocationClient) GetRoom(ctx context.Context, roomID string) (floorID, buildingID string, found bool, err error) {
	resp, err := c.client.GetRoom(ctx, &locationv1.GetRoomRequest{RoomId: roomID})
	if err != nil {
		return "", "", false, fmt.Errorf("location.GetRoom: %w", err)
	}
	if !resp.GetFound() || resp.GetRoom() == nil {
		return "", "", false, nil
	}
	return resp.GetRoom().GetFloorId(), resp.GetRoom().GetBuildingId(), true, nil
}

// GetFloor resolves floorID to its parent buildingID.
// Returns (buildingID, found=true, nil) on success;
// ("", false, nil) when not found; ("", false, err) on transport errors.
func (c *LocationClient) GetFloor(ctx context.Context, floorID string) (buildingID string, found bool, err error) {
	resp, err := c.client.GetFloor(ctx, &locationv1.GetFloorRequest{FloorId: floorID})
	if err != nil {
		return "", false, fmt.Errorf("location.GetFloor: %w", err)
	}
	if !resp.GetFound() || resp.GetFloor() == nil {
		return "", false, nil
	}
	return resp.GetFloor().GetBuildingId(), true, nil
}

// GetBuilding checks whether a building exists in the location service.
// Returns (found=true, nil) when active; (false, nil) when not found;
// (false, err) on transport errors.
func (c *LocationClient) GetBuilding(ctx context.Context, buildingID string) (found bool, err error) {
	resp, err := c.client.GetBuilding(ctx, &locationv1.GetBuildingRequest{BuildingId: buildingID})
	if err != nil {
		return false, fmt.Errorf("location.GetBuilding: %w", err)
	}
	return resp.GetFound(), nil
}
