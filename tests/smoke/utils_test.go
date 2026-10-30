package smoke_test

func toPtr[T any](v T) *T {
	return &v
}
