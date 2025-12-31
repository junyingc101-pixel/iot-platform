package svc

import (
	"gitee/getcharzp/iot-platform/device/types/device"
	"gitee/getcharzp/iot-platform/gateway/internal/config"
	"gitee/getcharzp/iot-platform/user/rpc/types/user"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config    config.Config
	UserRpc   user.UserClient
	DeviceRpc device.DeviceClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 连接user.rpc服务
	userRpc := zrpc.MustNewClient(c.UserRpc)
	userRpcClient := user.NewUserClient(userRpc.Conn())

	// 连接device.rpc服务
	deviceRpc := zrpc.MustNewClient(c.DeviceRpc)
	deviceRpcClient := device.NewDeviceClient(deviceRpc.Conn())

	return &ServiceContext{
		Config:    c,
		UserRpc:   userRpcClient,
		DeviceRpc: deviceRpcClient,
	}
}
