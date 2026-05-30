package grpcclient

import (
	"context"
	"fmt"

	"project/pkg/trace"
	locationv1 "project/proto/location/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// LocationClient implements usecase.RoomResolver by calling the location service
// over gRPC.
type LocationClient struct {
	client locationv1.LocationServiceClient
}

// NewLocationClient dials the location service and returns a LocationClient.
// addr is typically cfg.LocationService.GRPCAddr.
func NewLocationClient(addr string) (*LocationClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(trace.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("location grpc client: dial %s: %w", addr, err)
	}
	return &LocationClient{client: locationv1.NewLocationServiceClient(conn)}, nil
}

// GetRoom resolves roomID to its parent buildingID via the location service.
// Returns (buildingID, found=true, nil) on success; (_, false, nil) when not found;
// ("", false, err) on transport/server errors.
func (c *LocationClient) GetRoom(ctx context.Context, roomID string) (buildingID string, found bool, err error) {
	resp, err := c.client.GetRoom(ctx, &locationv1.GetRoomRequest{RoomId: roomID})
	if err != nil {
		return "", false, fmt.Errorf("location.GetRoom: %w", err)
	}
	if !resp.GetFound() || resp.GetRoom() == nil {
		return "", false, nil
	}
	return resp.GetRoom().GetBuildingId(), true, nil
}
