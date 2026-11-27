package id

import "github.com/google/uuid"

// NewV7 generates a time-ordered UUIDv7 suitable for PostgreSQL UUID primary keys.
func NewV7() (uuid.UUID, error) {
	return uuid.NewV7()
}

// MustNewV7 generates a time-ordered UUIDv7 and panics only if OS CSPRNG fails.
func MustNewV7() uuid.UUID {
	u, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return u
}
