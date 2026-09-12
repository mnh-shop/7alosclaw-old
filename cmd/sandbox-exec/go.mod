// Separate module so the sandbox launcher's dependency (go-landlock) never
// enters the main gateway module's go.mod / go.sum. The sandbox image builds
// it with `go mod tidy` at image-build time.
module github.com/mnh-shop/7alosclaw/cmd/sandbox-exec

go 1.26.0

require github.com/landlock-lsm/go-landlock v0.10.0
