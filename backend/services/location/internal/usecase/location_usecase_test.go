package usecase_test

import (
	"context"
	"errors"
	"testing"

	"project/services/location/internal/entity"
	"project/services/location/internal/usecase"

	"gorm.io/gorm"
)

// mockLocationRepo is an in-memory LocationRepository for usecase tests.
type mockLocationRepo struct {
	buildings map[string]*entity.Building
	floors    map[string]*entity.Floor
	rooms     map[string]*entity.Room
	seq       int
}

func newMockLocationRepo() *mockLocationRepo {
	return &mockLocationRepo{
		buildings: map[string]*entity.Building{},
		floors:    map[string]*entity.Floor{},
		rooms:     map[string]*entity.Room{},
	}
}

func (m *mockLocationRepo) id(prefix string) string {
	m.seq++
	return prefix + string(rune('0'+m.seq))
}

func (m *mockLocationRepo) CreateBuilding(_ context.Context, b *entity.Building) error {
	b.ID = m.id("b")
	m.buildings[b.ID] = b
	return nil
}
func (m *mockLocationRepo) GetBuilding(_ context.Context, id string) (*entity.Building, error) {
	if b, ok := m.buildings[id]; ok {
		return b, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockLocationRepo) ListBuildings(_ context.Context, _ bool) ([]*entity.Building, error) {
	out := make([]*entity.Building, 0, len(m.buildings))
	for _, b := range m.buildings {
		out = append(out, b)
	}
	return out, nil
}
func (m *mockLocationRepo) UpdateBuilding(_ context.Context, b *entity.Building) error {
	m.buildings[b.ID] = b
	return nil
}
func (m *mockLocationRepo) DeleteBuilding(_ context.Context, id string) error {
	delete(m.buildings, id)
	return nil
}
func (m *mockLocationRepo) CreateFloor(_ context.Context, f *entity.Floor) error {
	f.ID = m.id("f")
	m.floors[f.ID] = f
	return nil
}
func (m *mockLocationRepo) GetFloor(_ context.Context, id string) (*entity.Floor, error) {
	if f, ok := m.floors[id]; ok {
		return f, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockLocationRepo) ListFloors(_ context.Context, _ string, _ bool) ([]*entity.Floor, error) {
	return nil, nil
}
func (m *mockLocationRepo) UpdateFloor(_ context.Context, f *entity.Floor) error { return nil }
func (m *mockLocationRepo) DeleteFloor(_ context.Context, _ string) error        { return nil }
func (m *mockLocationRepo) CreateRoom(_ context.Context, r *entity.Room) error {
	r.ID = m.id("r")
	m.rooms[r.ID] = r
	return nil
}
func (m *mockLocationRepo) GetRoom(_ context.Context, id string) (*entity.Room, error) {
	if r, ok := m.rooms[id]; ok {
		return r, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockLocationRepo) ListRooms(_ context.Context, _ string, _ bool) ([]*entity.Room, error) {
	return nil, nil
}
func (m *mockLocationRepo) UpdateRoom(_ context.Context, _ *entity.Room) error { return nil }
func (m *mockLocationRepo) DeleteRoom(_ context.Context, _ string) error       { return nil }

func TestCreateBuilding_RequiresName(t *testing.T) {
	uc := usecase.NewLocationUsecase(newMockLocationRepo())
	if _, err := uc.CreateBuilding(context.Background(), "", ""); !errors.Is(err, usecase.ErrBuildingNameRequired) {
		t.Fatalf("expected ErrBuildingNameRequired, got %v", err)
	}
}

func TestCreateFloor_BuildingNotFound(t *testing.T) {
	uc := usecase.NewLocationUsecase(newMockLocationRepo())
	if _, err := uc.CreateFloor(context.Background(), "missing", "Tầng 1", 1); !errors.Is(err, usecase.ErrBuildingNotFound) {
		t.Fatalf("expected ErrBuildingNotFound, got %v", err)
	}
}

func TestResolveRoom_FullPath(t *testing.T) {
	repo := newMockLocationRepo()
	uc := usecase.NewLocationUsecase(repo)
	ctx := context.Background()

	b, _ := uc.CreateBuilding(ctx, "Toà A", "123 Đường X")
	f, _ := uc.CreateFloor(ctx, b.ID, "Tầng 1", 1)
	r, _ := uc.CreateRoom(ctx, f.ID, "P101", "Phòng 101")

	resolved, err := uc.ResolveRoom(ctx, r.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Building.Name != "Toà A" || resolved.Floor.Name != "Tầng 1" || resolved.Room.Code != "P101" {
		t.Fatalf("unexpected resolved path: %+v", resolved)
	}
}

func TestResolveRoom_RequiresID(t *testing.T) {
	uc := usecase.NewLocationUsecase(newMockLocationRepo())
	if _, err := uc.ResolveRoom(context.Background(), ""); !errors.Is(err, usecase.ErrRoomIDRequired) {
		t.Fatalf("expected ErrRoomIDRequired, got %v", err)
	}
}
