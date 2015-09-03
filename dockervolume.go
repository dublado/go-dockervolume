package dockervolume // import "go.pedge.io/dockervolume"

import (
	"net"
	"net/http"
)

// VolumeDriver mimics docker's volumedrivers.VolumeDriver, except
// does not use the volumedrivers.opts type. This allows this interface
// to be implemented in other packages.
//
// TODO(pedge): replace this if volumedrivers.VolumeDriver stops doing this.
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

// NewVolumeDriverHandler returns a new http.Handler.
func NewVolumeDriverHandler(volumeDriver VolumeDriver) http.Handler {
	return newVolumeDriverHandler(volumeDriver)
}

// NewTCPListener returns a new net.Listener for TCP.
//
// The string returned is a file that should be removed when finished with the listener.
func NewTCPListener(
	volumeDriverName string,
	address string,
	start <-chan struct{},
) (net.Listener, string, error) {
	return newTCPListener(
		volumeDriverName,
		address,
		start,
	)
}

// NewUnixListener returns a new net.Listener for Unix.
//
// The string returned is a file that should be removed when finished with the listener.
func NewUnixListener(
	volumeDriverName string,
	group string,
	start <-chan struct{},
) (net.Listener, string, error) {
	return newUnixListener(
		volumeDriverName,
		group,
		start,
	)
}
