//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"gorm.io/gorm"

	"project/services/user/internal/handler/event"
	grpchandler "project/services/user/internal/handler/grpc"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"
)

var userSet = wire.NewSet(
	persistence.NewUserGormRepository,
	usecase.NewUserUsecase,
	grpchandler.NewUserServiceServer,
	event.NewUserEventHandler,
)

func InitializeGRPCServer(db *gorm.DB) *grpchandler.UserServiceServer {
	wire.Build(userSet)
	return nil
}

func InitializeEventHandler(db *gorm.DB) *event.UserEventHandler {
	wire.Build(userSet)
	return nil
}
