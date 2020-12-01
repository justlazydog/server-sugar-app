package dgraph

import (
	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"google.golang.org/grpc"
)

// Dg is dGraph client.
var Dg *dgo.Dgraph

// Open connecting to dGraph.
func Open(RPCAddr string) error {
	conn, err := grpc.Dial(RPCAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	dc := api.NewDgraphClient(conn)
	Dg = dgo.NewDgraphClient(dc)
	return nil
}
