package reconfigure

// import (
// 	"context"
// 	"net"

// 	proto "github.com/gravitational/gravity/lib/install/proto"
// 	"github.com/gravitational/gravity/lib/install/server"
// )

// type Config struct {
// }

// func New(ctx context.Context, config Config) (*reconfigurator, error) {
// }

// type reconfigurator struct {
// 	Config
// 	ctx    context.Context
// 	cancel context.CancelFunc
// 	server *server.Server
// 	errC   chan error
// }

// func (r *reconfigurator) Run(listener net.Listener) error {
// 	go func() {
// 		r.errC <- r.server.Run(r, listener)
// 	}()
// 	err := <-r.errC
// 	r.stop()
// 	return proto.WrapServiceError(err)
// }
