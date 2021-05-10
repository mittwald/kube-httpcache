package tracing

type Transaction interface {
	Finish(err error)
}

type transaction struct {
	finish func(err error)
}

func (t transaction) Finish(err error) {
	t.finish(err)
}

type multiTransaction []Transaction

func (t multiTransaction) Finish(err error) {
	for _, v := range t {
		v.Finish(err)
	}
}