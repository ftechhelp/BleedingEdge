module github.com/bleeding-edge/bleeding-edge

go 1.25

require (
	github.com/docker/docker v20.10.27+incompatible
	github.com/docker/go-connections v0.5.0
	github.com/gorilla/mux v1.8.1
)

require (
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/docker/distribution v0.0.0-00010101000000-000000000000 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.1 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.4.1 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	gotest.tools/v3 v3.5.2 // indirect
)

replace github.com/docker/distribution => github.com/distribution/distribution v2.7.1+incompatible
