package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
)

// IdentityService implements the CSI Identity service.
type IdentityService struct {
	config *Config
}

// NewIdentityService returns a new Identity service.
func NewIdentityService(config *Config) *IdentityService {
	return &IdentityService{config: config}
}

// GetPluginInfo returns the name and version of the plugin.
func (s *IdentityService) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	klog.V(4).Info("GetPluginInfo called")
	return &csi.GetPluginInfoResponse{
		Name:          s.config.DriverName,
		VendorVersion: "0.1.0",
	}, nil
}

// GetPluginCapabilities returns the capabilities of the plugin.
func (s *IdentityService) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	klog.V(4).Info("GetPluginCapabilities called")
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}, nil
}

// Probe returns the health of the plugin.
func (s *IdentityService) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	klog.V(4).Info("Probe called")
	return &csi.ProbeResponse{}, nil
}
