/*
Package dockervolume is a library taking care of the generic code for docker volume plugins written in go.

Also see https://go.pedge.io/dockerplugin.

Your volume plugin must implement the VolumeDriver interface.

The API in this package exposes additional functionality on top of the
docker volume plugin API. See the protocol buffers file for details.

To launch your plugin using Unix sockets, do:

	func launch(volumeDriver dockervolume.VolumeDriver) error {
	  return dockervolume.NewUnixServer(
		volumeDriver,
		"volume_driver_name",
		"root",
		dockerplugin.ServerOptions{},
	  ).Serve()
	}

To launch your plugin using TCP, do:

	func launch(volumeDriver dockervolume.VolumeDriver) error {
	  return dockervolume.NewTCPServer(
		volumeDriver,
		"volume_driver_name",
		"address",
		dockerplugin.ServerOptions{},
	  ).Serve()
	}

Examples:

https://github.com/peter-edge/go-dockervolume/tree/master/example/cmd/dockervolume-example
https://github.com/pachyderm/pachyderm/tree/master/src/cmd/pfs-volume-driver
*/
package dockervolume // import "go.pedge.io/dockervolume"

import (
	"go.pedge.io/dockerplugin"
	"go.pedge.io/pkg/map"
	"google.golang.org/grpc"
)

// VolumeDriver is the interface that should be implemented for custom volume drivers.
type VolumeDriver interface {
	// Create a volume with the given name and opts.
	Create(name string, opts pkgmap.StringStringMap) (err error)
	// Remove the volume with the given name. opts and mountpoint were the opts
	// given when created, and mountpoint when mounted, if ever mounted.
	Remove(name string, opts pkgmap.StringStringMap, mountpoint string) (err error)
	// Mount the given volume and return the mountpoint. opts were the opts
	// given when created.
	Mount(name string, opts pkgmap.StringStringMap) (mountpoint string, err error)
	// Unmount the given volume. opts were the opts and mountpoint were the
	// opts given when created, and mountpoint when mounted.
	Unmount(name string, opts pkgmap.StringStringMap, mountpoint string) (err error)
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
	Cleanup() ([]*Volume, error)
	// Get a volume by name.
	GetVolume(name string) (*Volume, error)
	// List all volumes.
	ListVolumes() ([]*Volume, error)
}

// NewVolumeDriverClient creates a new VolumeDriverClient for the given APIClient.
func NewVolumeDriverClient(apiClient APIClient) VolumeDriverClient {
	return newVolumeDriverClient(apiClient)
}

// NewAPIServer returns a new APIServer for the given VolumeDriver and name.
func NewAPIServer(volumeDriver VolumeDriver, volumeDriverName string) APIServer {
	return newAPIServer(volumeDriver, volumeDriverName)
}

// NewTCPServer returns a new Server for TCP.
func NewTCPServer(
	volumeDriver VolumeDriver,
	volumeDriverName string,
	address string,
	opts dockerplugin.ServerOptions,
) dockerplugin.Server {
	return dockerplugin.NewTCPServer(
		volumeDriverName,
		[]string{"VolumeDriver"},
		func(s *grpc.Server) { RegisterAPIServer(s, NewAPIServer(volumeDriver, volumeDriverName)) },
		RegisterAPIHandler,
		address,
		opts,
	)
}

// NewUnixServer returns a new Server for Unix sockets.
func NewUnixServer(
	volumeDriver VolumeDriver,
	volumeDriverName string,
	group string,
	opts dockerplugin.ServerOptions,
) dockerplugin.Server {
	return dockerplugin.NewUnixServer(
		volumeDriverName,
		[]string{"VolumeDriver"},
		func(s *grpc.Server) { RegisterAPIServer(s, NewAPIServer(volumeDriver, volumeDriverName)) },
		RegisterAPIHandler,
		group,
		opts,
	)
}
