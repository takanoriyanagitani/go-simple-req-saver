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

func Get1stOrDefault[T any](s []T) (t T) {
	if 0 < len(s) {
		return s[0]
	}
	return t
}
