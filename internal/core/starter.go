package core

import "context"

type Starter interface {
	Start(ctx context.Context)
}
