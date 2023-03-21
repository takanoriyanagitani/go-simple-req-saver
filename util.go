package saver

func Compose[T, U, V any](
	f func(T) (U, error),
	g func(U) (V, error),
) func(T) (V, error) {
	return func(t T) (v V, e error) {
		u, e := f(t)
		if nil != e {
			return v, e
		}
		return g(u)
	}
}

// Get1stOrDefault tries to get the 1st element of a slice.
//
// # Returns
//   - The 1st element of a slice if a slice has an item.
//   - The zero value for a type T if a slice is empty.
func Get1stOrDefault[T any](s []T) (t T) {
	if 0 < len(s) {
		return s[0]
	}
	return t
}

func Curry[T, U, V any](f func(T, U) (V, error)) func(T) func(U) (V, error) {
	return func(t T) func(U) (V, error) {
		return func(u U) (V, error) {
			return f(t, u)
		}
	}
}
