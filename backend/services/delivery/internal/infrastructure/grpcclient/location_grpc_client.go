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

// RoomPath is the resolved human-readable path for a delivery location.
// Depending on the level, some fields may be empty (e.g. BUILDING has no floor/room).
type RoomPath struct {
	BuildingName string
	FloorName    string
	RoomName     string
	RoomCode     string
}

// String formats the path for display at full ROOM level:
// "Toà A / Tầng 2 / Phòng 201"
func (r RoomPath) String() string {
	if r.BuildingName == "" {
		return r.RoomName
	}
	return fmt.Sprintf("%s / %s / %s", r.BuildingName, r.FloorName, r.RoomName)
}

// LocationClient wraps the Location gRPC service to resolve delivery paths.
type LocationClient struct {
	client locationv1.LocationServiceClient
}

// NewLocationClient dials the location service. Returns nil on failure —
// callers must nil-check and degrade gracefully (path shows UUID fallback).
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

// GetLocationPath resolves a delivery location to a human-readable display string
// branching by level:
//   - "ROOM"     → GetRoom  → "Toà A / Tầng 2 / Phòng 201"
//   - "FLOOR"    → GetFloor → "Toà A / Tầng 2"
//   - "BUILDING" → GetBuilding → "Toà A"
//
// Empty level defaults to "ROOM" for back-compat. Returns the raw locationID on
// any error so the caller always has a non-empty fallback to display.
func (c *LocationClient) GetLocationPath(ctx context.Context, locationID, level string) string {
	if c == nil || locationID == "" {
		return locationID
	}
	if level == "" {
		level = "ROOM"
	}

	switch level {
	case "BUILDING":
		return c.getBuildingPath(ctx, locationID)
	case "FLOOR":
		return c.getFloorPath(ctx, locationID)
	default: // "ROOM" and any unknown value
		return c.getRoomPath(ctx, locationID)
	}
}

// GetRoomPath resolves a room_id to its building/floor/room display path.
// Kept for internal use; external callers should use GetLocationPath.
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

// getRoomPath resolves ROOM level → "Toà A / Tầng 2 / Phòng 201".
func (c *LocationClient) getRoomPath(ctx context.Context, roomID string) string {
	rp := c.GetRoomPath(ctx, roomID)
	if display := rp.String(); display != "" {
		return display
	}
	return roomID
}

// getFloorPath resolves FLOOR level → "Toà A / Tầng 2".
func (c *LocationClient) getFloorPath(ctx context.Context, floorID string) string {
	resp, err := c.client.GetFloor(ctx, &locationv1.GetFloorRequest{FloorId: floorID})
	if err != nil {
		slog.WarnContext(ctx, "delivery: location.GetFloor failed — degrading to id",
			"floor_id", floorID, "err", err)
		return floorID
	}
	if !resp.GetFound() || resp.GetFloor() == nil {
		return floorID
	}
	f := resp.GetFloor()
	if f.GetBuildingName() == "" {
		return f.GetFloorName()
	}
	return fmt.Sprintf("%s / %s", f.GetBuildingName(), f.GetFloorName())
}

// getBuildingPath resolves BUILDING level → "Toà A".
func (c *LocationClient) getBuildingPath(ctx context.Context, buildingID string) string {
	resp, err := c.client.GetBuilding(ctx, &locationv1.GetBuildingRequest{BuildingId: buildingID})
	if err != nil {
		slog.WarnContext(ctx, "delivery: location.GetBuilding failed — degrading to id",
			"building_id", buildingID, "err", err)
		return buildingID
	}
	if !resp.GetFound() || resp.GetBuilding() == nil {
		return buildingID
	}
	name := resp.GetBuilding().GetBuildingName()
	if name == "" {
		return buildingID
	}
	return name
}
