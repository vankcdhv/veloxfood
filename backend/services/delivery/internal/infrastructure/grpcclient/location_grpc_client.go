package grpcclient

import (
	"context"
	"fmt"
	"log/slog"

	"project/pkg/trace"
	locationv1 "project/proto/location/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RoomPath is the resolved human-readable path for a delivery room.
type RoomPath struct {
	BuildingName string
	FloorName    string
	RoomName     string
	RoomCode     string
}

// String formats the room path for display (e.g. "Building A / Floor 2 / Room 201").
func (r RoomPath) String() string {
	if r.BuildingName == "" {
		return r.RoomName
	}
	return fmt.Sprintf("%s / %s / %s", r.BuildingName, r.FloorName, r.RoomName)
}

// LocationClient wraps the Location gRPC service to resolve room paths.
type LocationClient struct {
	client locationv1.LocationServiceClient
}

// NewLocationClient dials the location service. Returns nil on failure —
// callers must nil-check and degrade gracefully (room path shows UUID fallback).
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

// GetRoomPath resolves a room_id to its building/floor/room display path.
// Returns a zero-value RoomPath (empty strings) when not found or on error —
// callers fall back to showing the raw room_id rather than failing the request.
func (c *LocationClient) GetRoomPath(ctx context.Context, roomID string) RoomPath {
	if c == nil || roomID == "" {
		return RoomPath{}
	}
	resp, err := c.client.GetRoom(ctx, &locationv1.GetRoomRequest{RoomId: roomID})
	if err != nil {
		slog.WarnContext(ctx, "delivery: location.GetRoom failed — degrading to empty path",
			"room_id", roomID, "err", err)
		return RoomPath{}
	}
	if !resp.GetFound() || resp.GetRoom() == nil {
		return RoomPath{}
	}
	r := resp.GetRoom()
	return RoomPath{
		BuildingName: r.GetBuildingName(),
		FloorName:    r.GetFloorName(),
		RoomName:     r.GetRoomName(),
		RoomCode:     r.GetRoomCode(),
	}
}
