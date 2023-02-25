package saver

type HeaderStd interface {
	Get(key string) (values []string)
}

type HeaderStd1st interface {
	Get(key string) (val string)
}

type HeaderStdFn func(key string) (values []string)
type HeaderStdFn1st func(key string) (val string)

func (h HeaderStdFn) Get(key string) (values []string) { return h(key) }
func (h HeaderStdFn) AsHeaderStd() HeaderStd           { return h }
func (h HeaderStdFn) ToHeaderStd1st() HeaderStd1st {
	var f1st HeaderStdFn1st = func(key string) (val string) {
		var values []string = h(key)
		return Get1stOrDefault(values)
	}
	return f1st.AsHeaderStd1st()
}

func (h HeaderStdFn1st) Get(key string) (val string)  { return h(key) }
func (h HeaderStdFn1st) AsHeaderStd1st() HeaderStd1st { return h }
