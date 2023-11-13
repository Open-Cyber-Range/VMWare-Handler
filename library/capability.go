package library

import (
	"context"

	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CapabilityServer struct {
	wrappedCapabilities capability.Capabilities
	capability.UnimplementedCapabilityServer
}

func NewCapabilityServer(capabilities []capability.Capabilities_DeployerType) CapabilityServer {
	return CapabilityServer{
		wrappedCapabilities: capability.Capabilities{
			Values: capabilities,
		},
	}
}

func (server *CapabilityServer) GetCapabilities(context.Context, *emptypb.Empty) (*capability.Capabilities, error) {
	status.New(codes.OK, "OK")
	return &server.wrappedCapabilities, nil
}
