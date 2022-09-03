package jperror

type JpError struct {
	err    error
	ErrFuc ErrorFuc
}

func (e *JpError) Error() string {
	return e.err.Error()
}
func Default() *JpError {
	return &JpError{}
}

func (e *JpError) Put(err error) {
	e.Check(err)
}

func (e *JpError) Check(err error) {
	if err != nil {
		e.err = err
		panic(err)
	}
}

type ErrorFuc func(jpError *JpError)

//暴露一个方法  让用户自定义
func (e *JpError) Result(errFuc ErrorFuc) {
	e.ErrFuc = errFuc
}
func (e *JpError) ExecResult() {
	e.ErrFuc(e)
}
