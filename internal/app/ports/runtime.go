package ports

import (
	"context"
	"io"

	"github.com/netlab/netlab/internal/domain"
)

type ActualNode struct {
	State domain.ObservedState
	Owner map[string]string
}
type NodeRuntime interface {
	Inspect(context.Context, domain.Node) (ActualNode, error)
	Start(context.Context, domain.Node) error
	Stop(context.Context, domain.Node) error
	Delete(context.Context, domain.Node) error
}
type NetworkRuntime interface {
	Connect(context.Context, domain.Link) error
	Disconnect(context.Context, domain.Link) error
	InspectLink(context.Context, domain.Link) (string, error)
}
type CgroupRuntime interface {
	Apply(context.Context, domain.Node) error
	Remove(context.Context, domain.ID) error
}
type CaptureRuntime interface {
	Start(context.Context, domain.ID, string) (io.ReadCloser, error)
	Stop(context.Context, domain.ID) error
}
type ConsoleRuntime interface {
	Open(context.Context, domain.ID, string) (io.ReadWriteCloser, error)
}
type ImageRuntime interface {
	Validate(context.Context, domain.ID) error
}
type ArtifactRuntime interface {
	Create(context.Context, string, domain.ID) (domain.Artifact, io.WriteCloser, error)
	Open(context.Context, domain.ID) (io.ReadCloser, error)
	Delete(context.Context, domain.ID) error
}
