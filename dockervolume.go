package dockervolume // import "go.pedge.io/dockervolume"

// VolumeDriver is the interface that should be implemented for custom volume drivers.
type VolumeDriver interface {
	// Create a volume with the given name and opts.
	Create(name string, opts map[string]string) (err error)
	// Remove the volume with the given name. opts and mountpoint were the opts
	// given when created, and mountpoint when mounted, if ever mounted.
	Remove(name string, opts map[string]string, mountpoint string) (err error)
	// Mount the given volume and return the mountpoint. opts were the opts
	// given when created.
	Mount(name string, opts map[string]string) (mountpoint string, err error)
	// Unmount the given volume. opts were the opts and mountpoint were the
	// opts given when created, and mountpoint when mounted.
	Unmount(name string, opts map[string]string, mountpoint string) (err error)
}

// VolumeDriverClient is a wrapper for APIClient.
type VolumeDriverClient interface {
	// Create a volume with the given name and opts.
	Create(name string, opts map[string]string) (err error)
	// Remove the volume with the given name.
	Remove(name string) (err error)
	// Get the path of the mountpoint for the given name.
	Path(name string) (mountpoint string, err error)
	// Mount the given volume and return the mountpoint.
	Mount(name string) (mountpoint string, err error)
	// Unmount the given volume.
	Unmount(name string) (err error)
	// Cleanup all volumes.
	Cleanup() ([]*RemoveVolumeAttempt, error)
	// Get a volume by name.
	GetVolume(name string) (*Volume, error)
	// List all volumes.
	ListVolumes() ([]*Volume, error)
	// Get events by volume name. Note that events are just in a cache,
	// which will be wiped when there are too many events.
	GetEventsByVolume(name string) ([]*Event, error)
	// List all events. Note that events are just in a cache,
	// which will be wupred when there are too many events.
	ListEvents() ([]*Event, error)
}

// NewVolumeDriverClient creates a new VolumeDriverClient for the given APIClient.
func NewVolumeDriverClient(apiClient APIClient) VolumeDriverClient {
	return newVolumeDriverClient(apiClient)
}

// Server serves a VolumeDriver.
type Server interface {
	Serve() error
}

// ServerOptions are options for a Server.
type ServerOptions struct {
	GRPCDebugPort       uint16
	NoCleanupOnShutdown bool
}

// NewTCPServer returns a new Server for TCP.
func NewTCPServer(
	volumeDriver VolumeDriver,
	volumeDriverName string,
	grpcPort uint16,
	address string,
	opts ServerOptions,
) Server {
	return newServer(
		protocolTCP,
		volumeDriver,
		volumeDriverName,
		grpcPort,
		address,
		opts,
	)
}

// NewUnixServer returns a new Server for Unix sockets.
func NewUnixServer(
	volumeDriver VolumeDriver,
	volumeDriverName string,
	grpcPort uint16,
	group string,
	opts ServerOptions,
) Server {
	return newServer(
		protocolUnix,
		volumeDriver,
		volumeDriverName,
		grpcPort,
		group,
		opts,
	)
}
