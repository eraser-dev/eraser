module github.com/Azure/eraser/pkg/eraser

go 1.16

require (
	github.com/Azure/eraser v0.0.0-20210720005525-9aab3f098186
	google.golang.org/grpc v1.39.0
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/cri-api v0.21.3
)

replace github.com/Azure/eraser => github.com/ashnamehrotra/eraser v0.0.0-20210804185924-6fc9a9334c39
