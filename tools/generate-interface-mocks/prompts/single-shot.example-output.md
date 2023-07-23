filepath: `user_provider.go`
```go
package mock

import (
	"context"
	"github.com/package/path/appname"
)

var _ appname.UserProvider = (*UserProvider)(nil)

type UserProvider struct {
	GetByIDFunc func(ctx context.Context, id string) (appname.User, error)
}

func (d *UserProvider) GetByID(ctx context.Context, id string) (appname.User, error) {
	return d.GetByIDFunc(ctx, id)
}
```

filepath: `reader.go`
```go
package mock

import (
	"context"
	"github.com/package/path/appname"
)

var _ appname.Reader = (*Reader)(nil)

type Reader struct {
	ReadFunc func(p []byte) (n int, err error)
}

func (d *Reader) (p []byte) (n int, err error) {
	return d.ReadFunc(p)
}
```
