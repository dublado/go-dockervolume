package dockervolume // import "go.pedge.io/dockervolume"

import "google.golang.org/grpc"

const (
	// ProtocolTCP denotes using TCP.
	ProtocolTCP Protocol = iota
	// ProtocolUnix denotes using Unix sockets.
	ProtocolUnix
)

// Protocol represents TCP or Unix.
type Protocol int

// VolumeDriver is the interface that should be implemented for custom volume drivers.
type VolumeDriver interface {
	// Create a volume with the given name
	Create(name string, opts map[string]string) (err error)
	// Remove the volume with the given name
	Remove(name string) (err error)
	// Get the mountpoint of the given volume
	Path(name string) (mountpoint string, err error)
	// Mount the given volume and return the mountpoint
	Mount(name string) (mountpoint string, err error)
	// Unmount the given volume
	Unmount(name string) (err error)
}

// VolumeDriverClient can call VolumeDrivers, along with additional functionality.
type VolumeDriverClient interface {
	VolumeDriver
}

// NewVolumeDriverClient creates a new VolumeDriverClient for the given *grpc.ClientConn.
func NewVolumeDriverClient(clientConn *grpc.ClientConn) VolumeDriverClient {
	return newVolumeDriverClient(clientConn)
}

// Serve serves the VolumeDriver.
//
// grpcDebugPort can be 0.
func Serve(
	volumeDriver VolumeDriver,
	protocol Protocol,
	volumeDriverName string,
	groupOrAddress string,
	grpcPort uint16,
	grpcDebugPort uint16,
) error {
	return serve(
		volumeDriver,
		protocol,
		volumeDriverName,
		groupOrAddress,
		grpcPort,
		grpcDebugPort,
	)
}
