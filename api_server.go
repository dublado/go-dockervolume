package dockervolume

import (
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/fsouza/go-dockerclient"
	"github.com/golang/protobuf/proto"
	"go.pedge.io/google-protobuf"
	"go.pedge.io/proto/rpclog"
	"go.pedge.io/proto/time"
	"golang.org/x/net/context"
)

type apiServer struct {
	volumeDriver     VolumeDriver
	volumeDriverName string
	nameToVolume     map[string]*Volume
	lock             *sync.RWMutex
}

func newAPIServer(volumeDriver VolumeDriver, volumeDriverName string, noEvents bool) *apiServer {
	return &apiServer{
		volumeDriver,
		volumeDriverName,
		make(map[string]*Volume),
		&sync.RWMutex{},
	}
}

func (a *apiServer) Activate(_ context.Context, request *google_protobuf.Empty) (response *ActivateResponse, err error) {
	start := time.Now()
	defer func() { a.log("Activate", request, response, err, start) }()
	return &ActivateResponse{
		Implements: []string{
			"VolumeDriver",
		},
	}, nil
}

func (a *apiServer) Create(_ context.Context, request *CreateRequest) (response *CreateResponse, err error) {
	start := time.Now()
	defer func() { a.log("Create", request, response, err, start) }()
	volume := &Volume{
		request.Name,
		request.Opts,
		"",
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if _, ok := a.nameToVolume[request.Name]; ok {
		return &CreateResponse{
			Err: fmt.Sprintf("dockervolume: volume already created: %s", request.Name),
		}, nil
	}
	if err := a.volumeDriver.Create(request.Name, newOpts(copyStringStringMap(request.Opts))); err != nil {
		return &CreateResponse{
			Err: err.Error(),
		}, nil
	}
	a.nameToVolume[request.Name] = volume
	return &CreateResponse{}, nil
}

func (a *apiServer) Remove(_ context.Context, request *RemoveRequest) (response *RemoveResponse, err error) {
	start := time.Now()
	defer func() { a.log("Remove", request, response, err, start) }()
	a.lock.Lock()
	defer a.lock.Unlock()
	volume, ok := a.nameToVolume[request.Name]
	if !ok {
		return &RemoveResponse{
			Err: fmt.Sprintf("dockervolume: volume does not exist: %s", request.Name),
		}, nil
	}
	delete(a.nameToVolume, request.Name)
	if err := a.volumeDriver.Remove(volume.Name, newOpts(copyStringStringMap(volume.Opts)), volume.Mountpoint); err != nil {
		return &RemoveResponse{
			Err: err.Error(),
		}, nil
	}
	return &RemoveResponse{}, nil
}

func (a *apiServer) Path(_ context.Context, request *PathRequest) (response *PathResponse, err error) {
	start := time.Now()
	defer func() { a.log("Path", request, response, err, start) }()
	a.lock.RLock()
	defer a.lock.RUnlock()
	volume, ok := a.nameToVolume[request.Name]
	if !ok {
		return &PathResponse{
			Err: fmt.Sprintf("dockervolume: volume does not exist: %s", request.Name),
		}, nil
	}
	return &PathResponse{
		Mountpoint: volume.Mountpoint,
	}, nil
}

func (a *apiServer) Mount(_ context.Context, request *MountRequest) (response *MountResponse, err error) {
	start := time.Now()
	defer func() { a.log("Mount", request, response, err, start) }()
	a.lock.Lock()
	defer a.lock.Unlock()
	volume, ok := a.nameToVolume[request.Name]
	if !ok {
		return &MountResponse{
			Err: fmt.Sprintf("dockervolume: volume does not exist: %s", request.Name),
		}, nil
	}
	if volume.Mountpoint != "" {
		return &MountResponse{
			Err: fmt.Sprintf("dockervolume: volume already mounted: %s at %s", volume.Name, volume.Mountpoint),
		}, nil
	}
	mountpoint, err := a.volumeDriver.Mount(volume.Name, newOpts(copyStringStringMap(volume.Opts)))
	volume.Mountpoint = mountpoint
	if err != nil {
		return &MountResponse{
			Mountpoint: mountpoint,
			Err:        err.Error(),
		}, nil
	}
	return &MountResponse{
		Mountpoint: mountpoint,
	}, nil
}

func (a *apiServer) Unmount(_ context.Context, request *UnmountRequest) (response *UnmountResponse, err error) {
	start := time.Now()
	defer func() { a.log("Unmount", request, response, err, start) }()
	a.lock.Lock()
	defer a.lock.Unlock()
	volume, ok := a.nameToVolume[request.Name]
	if !ok {
		return &UnmountResponse{
			Err: fmt.Sprintf("dockervolume: volume does not exist: %s", request.Name),
		}, nil
	}
	if volume.Mountpoint == "" {
		return &UnmountResponse{
			Err: fmt.Sprintf("dockervolume: volume not mounted: %s at %s", volume.Name, volume.Mountpoint),
		}, nil
	}
	mountpoint := volume.Mountpoint
	volume.Mountpoint = ""
	if err := a.volumeDriver.Unmount(volume.Name, newOpts(copyStringStringMap(volume.Opts)), mountpoint); err != nil {
		return &UnmountResponse{
			Err: err.Error(),
		}, nil
	}
	return &UnmountResponse{}, nil
}

func (a *apiServer) Cleanup(_ context.Context, request *google_protobuf.Empty) (response *RemoveVolumeAttempts, err error) {
	start := time.Now()
	defer func() { a.log("Cleanup", request, response, err, start) }()
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	allVolumes, err := client.ListVolumes(docker.ListVolumesOptions{})
	if err != nil {
		return nil, err
	}
	var driverVolumes []docker.Volume
	for _, volume := range allVolumes {
		if volume.Driver == a.volumeDriverName {
			driverVolumes = append(driverVolumes, volume)
		}
	}
	var volumes []*Volume
	a.lock.RLock()
	for _, dockerVolume := range driverVolumes {
		if volume, ok := a.nameToVolume[dockerVolume.Name]; ok {
			volumes = append(volumes, volume)
		}
	}
	a.lock.RUnlock()
	removeVolumeAttempts := make([]*RemoveVolumeAttempt, len(volumes))
	for i, volume := range volumes {
		removeVolumeAttempt := &RemoveVolumeAttempt{
			Volume: copyVolume(volume),
		}
		if err := client.RemoveVolume(volume.Name); err != nil {
			removeVolumeAttempt.Err = err.Error()
		}
		removeVolumeAttempts[i] = removeVolumeAttempt
	}
	return &RemoveVolumeAttempts{
		RemoveVolumeAttempt: removeVolumeAttempts,
	}, nil
}

func (a *apiServer) GetVolume(_ context.Context, request *GetVolumeRequest) (response *Volume, err error) {
	start := time.Now()
	defer func() { a.log("GetVolume", request, response, err, start) }()
	a.lock.RLock()
	defer a.lock.RUnlock()
	volume, ok := a.nameToVolume[request.Name]
	if !ok {
		return nil, grpc.Errorf(codes.NotFound, request.Name)
	}
	return copyVolume(volume), nil
}

func (a *apiServer) ListVolumes(_ context.Context, request *google_protobuf.Empty) (response *Volumes, err error) {
	start := time.Now()
	defer func() { a.log("ListVolumes", request, response, err, start) }()
	a.lock.RLock()
	defer a.lock.RUnlock()
	volumes := make([]*Volume, len(a.nameToVolume))
	i := 0
	for _, volume := range a.nameToVolume {
		volumes[i] = copyVolume(volume)
		i++
	}
	return &Volumes{
		Volume: volumes,
	}, nil
}

func (a *apiServer) log(methodName string, request proto.Message, response proto.Message, err error, start time.Time) {
	protorpclog.Info("dockervolume.API", methodName, request, response, err, time.Since(start))
}

func copyStringStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	n := make(map[string]string, len(m))
	for key, value := range m {
		n[key] = value
	}
	return n
}

func copyVolume(volume *Volume) *Volume {
	if volume == nil {
		return nil
	}
	return &Volume{
		Name:       volume.Name,
		Opts:       copyStringStringMap(volume.Opts),
		Mountpoint: volume.Mountpoint,
	}
}

func now() *google_protobuf.Timestamp {
	return prototime.TimeToTimestamp(time.Now().UTC())
}
