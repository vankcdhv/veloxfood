package grpcclient

import (
	"context"
	"fmt"
	"log/slog"

	"project/pkg/grpcx"
	locationv1 "project/proto/location/v1"
)

// LocationClient wraps the Location gRPC service to resolve a delivery location
// to a human-readable path, so the store owner sees where to deliver (not a UUID).
type LocationClient struct {
	client locationv1.LocationServiceClient
}

// NewLocationClient dials the location service. Returns nil on failure — callers
// must nil-check and degrade gracefully (path shows the raw id as a fallback).
func NewLocationClient(addr string) (*LocationClient, error) {
	conn, err := grpcx.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("location grpc client: dial %s: %w", addr, err)
	}
	return &LocationClient{client: locationv1.NewLocationServiceClient(conn)}, nil
}

// GetLocationPath resolves a location to a display string, branching by level
// (BUILDING / FLOOR / ROOM). Empty level defaults to ROOM. Returns the raw id on
// any error so the caller always has a non-empty fallback.
func (c *LocationClient) GetLocationPath(ctx context.Context, locationID, level string) string {
	if c == nil || locationID == "" {
		return locationID
	}
	switch level {
	case "BUILDING":
		return c.buildingPath(ctx, locationID)
	case "FLOOR":
		return c.floorPath(ctx, locationID)
	default: // ROOM and any unknown value
		return c.roomPath(ctx, locationID)
	}
}

func (c *LocationClient) roomPath(ctx context.Context, roomID string) string {
	resp, err := c.client.GetRoom(ctx, &locationv1.GetRoomRequest{RoomId: roomID})
	if err != nil || !resp.GetFound() || resp.GetRoom() == nil {
		if err != nil {
			slog.WarnContext(ctx, "order: location.GetRoom failed", "room_id", roomID, "err", err)
		}
		return roomID
	}
	r := resp.GetRoom()
	if r.GetBuildingName() == "" {
		return r.GetRoomName()
	}
	return fmt.Sprintf("%s / %s / %s", r.GetBuildingName(), r.GetFloorName(), r.GetRoomName())
}

func (c *LocationClient) floorPath(ctx context.Context, floorID string) string {
	resp, err := c.client.GetFloor(ctx, &locationv1.GetFloorRequest{FloorId: floorID})
	if err != nil || !resp.GetFound() || resp.GetFloor() == nil {
		return floorID
	}
	f := resp.GetFloor()
	if f.GetBuildingName() == "" {
		return f.GetFloorName()
	}
	return fmt.Sprintf("%s / %s", f.GetBuildingName(), f.GetFloorName())
}

func (c *LocationClient) buildingPath(ctx context.Context, buildingID string) string {
	resp, err := c.client.GetBuilding(ctx, &locationv1.GetBuildingRequest{BuildingId: buildingID})
	if err != nil || !resp.GetFound() || resp.GetBuilding() == nil {
		return buildingID
	}
	if name := resp.GetBuilding().GetBuildingName(); name != "" {
		return name
	}
	return buildingID
}
