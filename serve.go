package dockervolume

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/gengo/grpc-gateway/runtime"
	"github.com/golang/protobuf/proto"
	"go.pedge.io/proto/server"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func serve(
	volumeDriver VolumeDriver,
	protocol Protocol,
	volumeDriverName string,
	groupOrAddress string,
	grpcPort uint16,
	grpcDebugPort uint16,
) (retErr error) {
	start := make(chan struct{})
	var listener net.Listener
	var spec string
	var err error
	var addr string
	switch protocol {
	case ProtocolTCP:
		listener, spec, err = newTCPListener(volumeDriverName, groupOrAddress, start)
		addr = groupOrAddress
	case ProtocolUnix:
		listener, spec, err = newUnixListener(volumeDriverName, groupOrAddress, start)
		addr = volumeDriverName
	default:
		return fmt.Errorf("unknown protocol: %v", protocol)
	}
	if spec != "" {
		defer func() {
			if err := os.Remove(spec); err != nil && retErr == nil {
				retErr = err
			}
		}()
	}
	if err != nil {
		return err
	}
	close(start)
	return protoserver.Serve(
		grpcPort,
		func(s *grpc.Server) {
			RegisterAPIServer(s, newAPIServer(volumeDriver))
		},
		protoserver.ServeOptions{
			DebugPort: grpcDebugPort,
			HTTPRegisterFunc: func(ctx context.Context, mux *runtime.ServeMux, clientConn *grpc.ClientConn) error {
				return RegisterAPIHandler(ctx, mux, clientConn)
			},
			HTTPAddress:  addr,
			HTTPListener: listener,
			ServeMuxOptions: []runtime.ServeMuxOption{
				runtime.WithForwardResponseOption(
					func(_ context.Context, responseWriter http.ResponseWriter, _ proto.Message) error {
						responseWriter.Header().Set("Content-Type", "application/vnd.docker.plugins.v1.1+json")
						return nil
					},
				),
			},
		},
	)
}
