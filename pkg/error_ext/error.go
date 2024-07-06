package error_ext

type Business struct {
}

type Error interface {
	Code() int
	Msg() string
}
