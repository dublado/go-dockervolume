package dockervolume

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"go.pedge.io/pkg/map"
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
	requireVolumesEqual(
		t,
		client,
		&Volume{
			Name: "foo",
			Opts: map[string]string{
				"key":    "value",
				"uint64": "1234",
			},
			Mountpoint: "",
		},
	)
	fakeVolumeDriver.requireStatusEquals("foo", fakeStatusCreate)
	mountpoint, err := client.Mount("foo")
	require.NoError(t, err)
	require.Equal(t, "/mnt/foo", mountpoint)
	requireVolumesEqual(
		t,
		client,
		&Volume{
			Name: "foo",
			Opts: map[string]string{
				"key":    "value",
				"uint64": "1234",
			},
			Mountpoint: "/mnt/foo",
		},
	)
	fakeVolumeDriver.requireStatusEquals("foo", fakeStatusMount)
	err = client.Unmount("foo")
	require.NoError(t, err)
	requireVolumesEqual(
		t,
		client,
		&Volume{
			Name: "foo",
			Opts: map[string]string{
				"key":    "value",
				"uint64": "1234",
			},
			Mountpoint: "",
		},
	)
	fakeVolumeDriver.requireStatusEquals("foo", fakeStatusUnmount)
	err = client.Remove("foo")
	require.NoError(t, err)
	requireVolumesEqual(t, client)
}

func requireVolumesEqual(t *testing.T, client VolumeDriverClient, expected ...*Volume) {
	volumes, err := client.ListVolumes()
	require.NoError(t, err)
	require.Equal(t, len(expected), len(volumes))
	nameToExpected := make(map[string]*Volume)
	nameToActual := make(map[string]*Volume)
	for _, volume := range expected {
		nameToExpected[volume.Name] = volume
	}
	for _, volume := range volumes {
		nameToActual[volume.Name] = volume
	}
	require.Equal(t, len(nameToExpected), len(nameToActual))
	for name, expected := range nameToExpected {
		actual, ok := nameToActual[name]
		require.True(t, ok)
		require.Equal(t, expected, actual)
		volume, err := client.GetVolume(name)
		require.NoError(t, err)
		require.Equal(t, expected, volume)
	}
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
				RegisterAPIServer(server, newAPIServer(fakeVolumeDriver, "test"))
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

type fakeVolumeDriver struct {
	t                *testing.T
	nameToFakeVolume map[string]*Volume
	nameToFakeStatus map[string]fakeStatus
}

func newFakeVolumeDriver(t *testing.T) *fakeVolumeDriver {
	return &fakeVolumeDriver{
		t,
		make(map[string]*Volume),
		make(map[string]fakeStatus),
	}
}

func (v *fakeVolumeDriver) Create(name string, opts pkgmap.StringStringMap) error {
	v.nameToFakeVolume[name] = &Volume{
		Name:       name,
		Opts:       opts,
		Mountpoint: "",
	}
	v.nameToFakeStatus[name] = fakeStatusCreate
	return nil
}

func (v *fakeVolumeDriver) Remove(name string, opts pkgmap.StringStringMap, mountpoint string) error {
	v.nameToFakeStatus[name] = fakeStatusRemove
	return nil
}

func (v *fakeVolumeDriver) Mount(name string, opts pkgmap.StringStringMap) (string, error) {
	mountpoint := fmt.Sprintf("/mnt/%s", name)
	v.nameToFakeVolume[name].Mountpoint = mountpoint
	v.nameToFakeStatus[name] = fakeStatusMount
	return mountpoint, nil
}

func (v *fakeVolumeDriver) Unmount(name string, opts pkgmap.StringStringMap, mountpoint string) error {
	v.nameToFakeVolume[name].Mountpoint = mountpoint
	v.nameToFakeStatus[name] = fakeStatusUnmount
	return nil
}

func (v *fakeVolumeDriver) requireStatusEquals(name string, expected fakeStatus) {
	fakeStatus, ok := v.nameToFakeStatus[name]
	require.True(v.t, ok)
	require.Equal(v.t, expected, fakeStatus)
}
