// +build !linux !freebsd

package dockervolume

import "net"

func newUnixListener(
	volumeDriverName string,
	address string,
	start <-chan struct{},
) (net.Listener, error) {
	return nil, nil
}
