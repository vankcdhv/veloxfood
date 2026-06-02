package usecase

import "project/pkg/apperror"

var (
	ErrBuildingNameRequired     = apperror.BadRequest("building name is required")
	ErrFloorNameRequired        = apperror.BadRequest("floor name is required")
	ErrRoomCodeRequired         = apperror.BadRequest("room code is required")
	ErrBuildingNotFound         = apperror.NotFound("building not found")
	ErrFloorNotFound            = apperror.NotFound("floor not found")
	ErrRoomNotFound             = apperror.NotFound("room not found")
	ErrCustomerLocationNotFound = apperror.NotFound("customer location not found")
	ErrRoomIDRequired           = apperror.BadRequest("room_id is required")
	ErrFloorIDRequired          = apperror.BadRequest("floor_id is required")
	ErrBuildingIDRequired       = apperror.BadRequest("building_id is required")
)
