package kafka

import "errors"

var (
	// Consumer 相关错误
	ErrHandlerAlreadySubscribed = errors.New("handler already subscribed")
	ErrConsumerAlreadyRunning   = errors.New("consumer already running")
	ErrNoHandlerSubscribed      = errors.New("no handler subscribed")
	ErrConsumerNotRunning       = errors.New("consumer not running")

	// TLS/Cert 相关错误
	ErrReadCAFile      = errors.New("failed to read CA file")
	ErrAppendCertToPEM = errors.New("failed to append certificate to PEM pool")
)

func IsClosedError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "consumer closed" ||
		err.Error() == "context canceled" ||
		err.Error() == "io.EOF"
}

func IsCommitError(err error) bool {
	return err != nil && err.Error() != ""
}

func IsFetchError(err error) bool {
	return err != nil && !IsClosedError(err)
}

func IsHandlerError(err error) bool {
	return err != nil && err.Error() != ""
}
