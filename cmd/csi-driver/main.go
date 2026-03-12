package main

import (
	"flag"
	"net"
	"os"
	"path/filepath"

	"k8s.io/klog/v2"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"

	"github.com/neo4j/nfs-csi-driver/driver"
)

var (
	// Default: host path where kubelet expects the socket when running on a node.
	// In deploy/ we override this with unix:///csi/csi.sock (container mount point).
	endpoint    = flag.String("endpoint", "unix:///var/lib/kubelet/plugins/nfs.csi.neo4j.io/csi.sock", "CSI endpoint")
	driverName  = flag.String("driver-name", "nfs.csi.neo4j.io", "Name of the CSI driver")
	nodeID      = flag.String("node-id", "", "Node ID (required for node service)")
	nfsServer   = flag.String("nfs-server", "", "NFS server hostname or IP (e.g. nfs-server.default.svc.cluster.local)")
	nfsRootPath = flag.String("nfs-root-path", "/exports", "Root path exported by NFS server where volume dirs are created")
	nfsRootMount = flag.String("nfs-root-mount", "/nfs-root", "Local path where NFS root is mounted (controller only)")
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	if *endpoint == "" {
		klog.Fatal("endpoint is required")
	}

	// Parse endpoint (e.g. unix:///path or tcp://host:port)
	listenAddr, err := parseEndpoint(*endpoint)
	if err != nil {
		klog.Fatalf("failed to parse endpoint %q: %v", *endpoint, err)
	}

	// Remove unix socket if it exists
	if filepath.Dir(listenAddr) != "" {
		if err := os.MkdirAll(filepath.Dir(listenAddr), 0750); err != nil {
			klog.Fatalf("failed to create socket directory: %v", err)
		}
	}
	if err := os.Remove(listenAddr); err != nil && !os.IsNotExist(err) {
		klog.Fatalf("failed to remove existing socket: %v", err)
	}

	listener, err := net.Listen("unix", listenAddr)
	if err != nil {
		klog.Fatalf("failed to listen on %s: %v", listenAddr, err)
	}
	defer listener.Close()

	config := &driver.Config{
		DriverName:   *driverName,
		NodeID:       *nodeID,
		NFSServer:    *nfsServer,
		NFSRootPath:  *nfsRootPath,
		NFSRootMount: *nfsRootMount,
	}

	identitySvc := driver.NewIdentityService(config)
	controllerSvc := driver.NewControllerService(config)
	nodeSvc := driver.NewNodeService(config)

	server := grpc.NewServer()
	csi.RegisterIdentityServer(server, identitySvc)
	csi.RegisterControllerServer(server, controllerSvc)
	csi.RegisterNodeServer(server, nodeSvc)

	klog.Infof("Listening for connections on %s", listenAddr)
	if err := server.Serve(listener); err != nil {
		klog.Fatalf("server failed: %v", err)
	}
}

// parseEndpoint returns the network address from a CSI endpoint string.
// Supported formats: unix:///path/to/socket, unix://@path (abstract), tcp://host:port
func parseEndpoint(ep string) (string, error) {
	if len(ep) == 0 {
		return "", nil
	}
	const unixPrefix = "unix://"
	if len(ep) >= len(unixPrefix) && ep[:len(unixPrefix)] == unixPrefix {
		return ep[len(unixPrefix):], nil
	}
	// tcp:// not handled for listen here; would need to return ("tcp", "host:port")
	return ep, nil
}
