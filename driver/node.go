package driver

import (
	"context"
	"os"
	"os/exec"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NodeService implements the CSI Node service.
type NodeService struct {
	config *Config
}

// NewNodeService returns a new Node service.
func NewNodeService(config *Config) *NodeService {
	return &NodeService{config: config}
}

// NodeStageVolume is not used for NFS (we mount directly in NodePublishVolume).
func (s *NodeService) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeStageVolume is not used for NFS")
}

// NodePublishVolume mounts the NFS share to the target path.
func (s *NodeService) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "volume id is required")
	}
	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path is required")
	}
	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "volume capability is required")
	}

	server := req.VolumeContext[VolumeContextServer]
	share := req.VolumeContext[VolumeContextShare]
	if server == "" || share == "" {
		return nil, status.Error(codes.InvalidArgument, "volume context must contain server and share")
	}

	if err := os.MkdirAll(req.TargetPath, 0750); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create target path %q: %v", req.TargetPath, err)
	}

	// Mount NFS: mount -t nfs server:share target
	remote := server + ":" + share
	klog.Infof("Mounting NFS %s to %s", remote, req.TargetPath)

	cmd := exec.Command("mount", "-t", "nfs", "-o", "vers=4", remote, req.TargetPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try NFSv3 if v4 fails (e.g. some servers only support v3)
		cmd3 := exec.Command("mount", "-t", "nfs", "-o", "vers=3", remote, req.TargetPath)
		if output3, err3 := cmd3.CombinedOutput(); err3 != nil {
			return nil, status.Errorf(codes.Internal, "mount failed (nfs4: %v, nfs3: %v): %s / %s", err, err3, string(output), string(output3))
		}
		klog.Infof("Mounted with NFSv3")
	} else {
		klog.Infof("Mounted with NFSv4")
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnstageVolume is not used for NFS.
func (s *NodeService) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeUnstageVolume is not used for NFS")
}

// NodeUnpublishVolume unmounts the NFS share from the target path.
func (s *NodeService) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "volume id is required")
	}
	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path is required")
	}

	klog.Infof("Unmounting %s", req.TargetPath)
	cmd := exec.Command("umount", req.TargetPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		if _, statErr := os.Stat(req.TargetPath); statErr != nil && os.IsNotExist(statErr) {
			return &csi.NodeUnpublishVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "umount failed: %v: %s", err, string(output))
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetVolumeStats returns volume stats (optional).
func (s *NodeService) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeGetVolumeStats is not implemented")
}

// NodeExpandVolume is not implemented.
func (s *NodeService) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeExpandVolume is not implemented")
}

// NodeGetCapabilities returns the node capabilities.
func (s *NodeService) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
					},
				},
			},
		},
	}, nil
}

// NodeGetInfo returns the node ID and topology.
func (s *NodeService) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	if s.config.NodeID == "" {
		return nil, status.Error(codes.FailedPrecondition, "node-id is required for node service")
	}
	return &csi.NodeGetInfoResponse{
		NodeId: s.config.NodeID,
	}, nil
}
