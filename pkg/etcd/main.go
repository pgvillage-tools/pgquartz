package etcd

import (
	"context"
)

var ctx = context.Background()

// InitContext sets the parent context used for etcd operations.
func InitContext(c context.Context) {
	ctx = c
}
