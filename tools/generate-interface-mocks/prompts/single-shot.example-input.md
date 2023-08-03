github.com/package/path/appname

package appname

type UserProvider interface {
	// GetByID returns a User by ID.
	// If the user doesn't exist, ErrNoSuchUser is returned.
	GetByID(ctx context.Context, id string) (User, error)
}

type Reader interface {
	Read(p []byte) (n int, err error)
}

type MapperFunc(ctx context.Context, item Item) (MappedItem, error)
