package dockervolume

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"go.pedge.io/proto/test"
	"google.golang.org/grpc"
)

const (
	fakeStatusCreate fakeStatus = iota
	fakeStatusRemove
	fakeStatusMount
	fakeStatusUnmount
)

type fakeStatus int

func TestBasic(t *testing.T) {
	runTest(t, testBasic)
}

func testBasic(t *testing.T, fakeVolumeDriver *fakeVolumeDriver, client VolumeDriverClient) {
	err := client.Create("foo", map[string]string{"key": "value", "uint64": "1234"})
	require.NoError(t, err)
	volumes, err := client.ListVolumes()
	require.NoError(t, err)
	require.Len(t, volumes, 1)
	require.Equal(
		t,
		&Volume{
			Name: "foo",
			Opts: map[string]string{
				"key":    "value",
				"uint64": "1234",
			},
			Mountpoint: "",
		},
		volumes[0],
	)
	fakeVolumeDriver.requireStatusEquals("foo", fakeStatusCreate)
	mountpoint, err := client.Mount("foo")
	require.NoError(t, err)
	require.Equal(t, "/mnt/foo", mountpoint)
	volumes, err = client.ListVolumes()
	require.NoError(t, err)
	require.Len(t, volumes, 1)
	require.Equal(
		t,
		&Volume{
			Name: "foo",
			Opts: map[string]string{
				"key":    "value",
				"uint64": "1234",
			},
			Mountpoint: "/mnt/foo",
		},
		volumes[0],
	)
	fakeVolumeDriver.requireStatusEquals("foo", fakeStatusMount)
}

func runTest(
	t *testing.T,
	testFunc func(*testing.T, *fakeVolumeDriver, VolumeDriverClient),
) {
	fakeVolumeDriver := newFakeVolumeDriver(t)
	prototest.RunT(
		t,
		1,
		func(addressToServer map[string]*grpc.Server) {
			for _, server := range addressToServer {
				RegisterAPIServer(server, newAPIServer(fakeVolumeDriver, "test", false))
			}
		},
		func(t *testing.T, addressToClientConn map[string]*grpc.ClientConn) {
			var clientConn *grpc.ClientConn
			for _, cc := range addressToClientConn {
				clientConn = cc
				break
			}
			testFunc(t, fakeVolumeDriver, NewVolumeDriverClient(NewAPIClient(clientConn)))
		},
	)
}

type fakeVolume struct {
	name       string
	opts       Opts
	mountpoint string
	fakeStatus fakeStatus
}

type fakeVolumeDriver struct {
	t                *testing.T
	nameToFakeVolume map[string]*fakeVolume
}

func newFakeVolumeDriver(t *testing.T) *fakeVolumeDriver {
	return &fakeVolumeDriver{
		t,
		make(map[string]*fakeVolume),
	}
}

func (v *fakeVolumeDriver) Create(name string, opts Opts) error {
	v.nameToFakeVolume[name] = &fakeVolume{
		name:       name,
		opts:       opts,
		mountpoint: "",
		fakeStatus: fakeStatusCreate,
	}
	return nil
}

func (v *fakeVolumeDriver) Remove(name string, opts Opts, mountpoint string) error {
	v.nameToFakeVolume[name].fakeStatus = fakeStatusRemove
	return nil
}

func (v *fakeVolumeDriver) Mount(name string, opts Opts) (string, error) {
	mountpoint := fmt.Sprintf("/mnt/%s", name)
	v.nameToFakeVolume[name].mountpoint = mountpoint
	v.nameToFakeVolume[name].fakeStatus = fakeStatusMount
	return mountpoint, nil
}

func (v *fakeVolumeDriver) Unmount(name string, opts Opts, mountpoint string) error {
	v.nameToFakeVolume[name].fakeStatus = fakeStatusUnmount
	return nil
}

func (v *fakeVolumeDriver) requireStatusEquals(name string, fakeStatus fakeStatus) {
	fakeVolume, ok := v.nameToFakeVolume[name]
	require.True(v.t, ok)
	require.Equal(v.t, fakeStatus, fakeVolume.fakeStatus)
}
