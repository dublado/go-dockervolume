package dockervolume

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	contentType = "application/vnd.docker.plugins.v1.1+json"
)

var (
	activateResponse = []byte("{\"Implements\": [\"VolumeDriver\"]}\n")
)

func newVolumeDriverHandler(volumeDriver VolumeDriver, opts VolumeDriverHandlerOptions) *http.ServeMux {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc(
		"/Plugin.Activate",
		func(responseWriter http.ResponseWriter, request *http.Request) {
			responseWriter.Header().Set("Content-Type", contentType)
			_, _ = responseWriter.Write(activateResponse)
		},
	)
	serveMux.HandleFunc(
		"/VolumeDriver.Create",
		newGenericHandlerFunc(
			Method_METHOD_CREATE,
			func(name string, opts map[string]string) (string, error) {
				return "", volumeDriver.Create(name, opts)
			},
			opts,
		),
	)
	serveMux.HandleFunc(
		"/VolumeDriver.Remove",
		newGenericHandlerFunc(
			Method_METHOD_REMOVE,
			func(name string, opts map[string]string) (string, error) {
				return "", volumeDriver.Remove(name)
			},
			opts,
		),
	)
	serveMux.HandleFunc(
		"/VolumeDriver.Mount",
		newGenericHandlerFunc(
			Method_METHOD_MOUNT,
			func(name string, opts map[string]string) (string, error) {
				return volumeDriver.Mount(name)
			},
			opts,
		),
	)
	serveMux.HandleFunc(
		"/VolumeDriver.Path",
		newGenericHandlerFunc(
			Method_METHOD_PATH,
			func(name string, opts map[string]string) (string, error) {
				return volumeDriver.Path(name)
			},
			opts,
		),
	)
	serveMux.HandleFunc(
		"/VolumeDriver.Unmount",
		newGenericHandlerFunc(
			Method_METHOD_UNMOUNT,
			func(name string, opts map[string]string) (string, error) {
				return "", volumeDriver.Unmount(name)
			},
			opts,
		),
	)
	return serveMux
}

func newGenericHandlerFunc(
	method Method,
	f func(string, map[string]string) (string, error),
	opts VolumeDriverHandlerOptions,
) func(http.ResponseWriter, *http.Request) {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		m := make(map[string]interface{})
		if err := json.NewDecoder(request.Body).Decode(&m); err != nil {
			http.Error(responseWriter, err.Error(), http.StatusBadRequest)
			return
		}
		responseWriter.Header().Set("Content-Type", contentType)
		_ = json.NewEncoder(responseWriter).Encode(wrap(method, f, m, getLogger(opts)))
	}
}

func wrap(
	method Method,
	f func(string, map[string]string) (string, error),
	request map[string]interface{},
	logger Logger,
) map[string]interface{} {
	methodInvocation := &MethodInvocation{
		Method: method,
	}
	response := make(map[string]interface{})
	if err := checkRequiredParameters(request, "Name"); err != nil {
		response["Err"] = err.Error()
		methodInvocation.Error = err.Error()
		logger.LogMethodInvocation(methodInvocation)
		return response
	}
	name := request["Name"].(string)
	var opts map[string]string
	if _, ok := request["Opts"]; ok {
		opts = make(map[string]string)
		for key, value := range request["Opts"].(map[string]interface{}) {
			opts[key] = fmt.Sprintf("%v", value)
		}
	}
	methodInvocation.Name = name
	methodInvocation.Opts = opts
	mountpoint, err := f(name, opts)
	if mountpoint != "" {
		response["Mountpoint"] = mountpoint
		methodInvocation.Mountpoint = mountpoint
	}
	if err != nil {
		response["Err"] = err.Error()
		methodInvocation.Error = err.Error()
	}
	logger.LogMethodInvocation(methodInvocation)
	return response
}

func checkRequiredParameters(request map[string]interface{}, parameters ...string) error {
	for _, parameter := range parameters {
		if _, ok := request[parameter]; !ok {
			return fmt.Errorf("required parameter %s not set", parameter)
		}
	}
	return nil
}

func getLogger(opts VolumeDriverHandlerOptions) Logger {
	if opts.Logger != nil {
		return opts.Logger
	}
	return loggerInstance
}
