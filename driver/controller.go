package driver

import (
	"context"
	"os"
	"path/filepath"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// Volume context keys passed to NodePublishVolume
	VolumeContextServer = "server"
	VolumeContextShare  = "share"
)

// ControllerService implements the CSI Controller service.
type ControllerService struct {
	config *Config
}

// NewControllerService returns a new Controller service.
func NewControllerService(config *Config) *ControllerService {
	return &ControllerService{config: config}
}

// CreateVolume creates a new directory on the NFS server (via the mounted root).
func (s *ControllerService) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "volume name is required")
	}
	if req.VolumeCapabilities == nil || len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume capabilities are required")
	}

	// Validate we support the capability (e.g. single node writer or multi reader)
	for _, cap := range req.VolumeCapabilities {
		if cap.GetBlock() != nil {
			return nil, status.Error(codes.InvalidArgument, "block volumes are not supported")
		}
	}

	volPath := filepath.Join(s.config.NFSRootMount, req.Name)
	if err := os.MkdirAll(volPath, 0755); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create volume directory %q: %v", volPath, err)
	}
	klog.Infof("Created volume directory %s", volPath)

	// Share path as seen by NFS clients (path on the NFS server)
	sharePath := filepath.Join(s.config.NFSRootPath, req.Name)
	// Normalize to forward slashes for NFS mount
	if filepath.Separator == '\\' {
		sharePath = filepath.ToSlash(sharePath)
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      req.Name,
			CapacityBytes: req.CapacityRange.GetRequiredBytes(),
			VolumeContext: map[string]string{
				VolumeContextServer: s.config.NFSServer,
				VolumeContextShare:  sharePath,
			},
		},
	}, nil
}

// DeleteVolume removes the volume directory from the NFS root.
func (s *ControllerService) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "volume id is required")
	}

	volPath := filepath.Join(s.config.NFSRootMount, req.VolumeId)
	if err := os.Remove(volPath); err != nil {
		if os.IsNotExist(err) {
			klog.V(4).Infof("Volume %s already removed", req.VolumeId)
			return &csi.DeleteVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to delete volume directory %q: %v", volPath, err)
	}
	klog.Infof("Deleted volume directory %s", volPath)
	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerPublishVolume is not used for NFS (no block device to attach).
func (s *ControllerService) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerPublishVolume is not supported for NFS")
}

// ControllerUnpublishVolume is not used for NFS.
func (s *ControllerService) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerUnpublishVolume is not supported for NFS")
}

// ValidateVolumeCapabilities validates the volume capabilities.
func (s *ControllerService) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "volume id is required")
	}
	if req.VolumeCapabilities == nil || len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume capabilities are required")
	}

	for _, cap := range req.VolumeCapabilities {
		if cap.GetBlock() != nil {
			return &csi.ValidateVolumeCapabilitiesResponse{
				Confirmed: nil,
				Message:   "block access is not supported",
			}, nil
		}
	}

	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.VolumeCapabilities,
		},
	}, nil
}

// ListVolumes returns volumes (optional; we return unsupported for simplicity).
func (s *ControllerService) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListVolumes is not implemented")
}

// GetCapacity is not implemented.
func (s *ControllerService) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetCapacity is not implemented")
}

// ControllerGetCapabilities returns the controller capabilities.
func (s *ControllerService) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
					},
				},
			},
		},
	}, nil
}

// CreateSnapshot is not implemented.
func (s *ControllerService) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CreateSnapshot is not implemented")
}

// DeleteSnapshot is not implemented.
func (s *ControllerService) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "DeleteSnapshot is not implemented")
}

// ListSnapshots is not implemented.
func (s *ControllerService) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListSnapshots is not implemented")
}

// ControllerExpandVolume is not implemented.
func (s *ControllerService) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerExpandVolume is not implemented")
}

// ControllerGetVolume is not implemented.
func (s *ControllerService) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerGetVolume is not implemented")
}

// ControllerModifyVolume is not implemented.
func (s *ControllerService) ControllerModifyVolume(ctx context.Context, req *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerModifyVolume is not implemented")
}
