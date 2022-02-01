package logger

import (
	"io/ioutil"

	uberzap "go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func Noop() *uberzap.Logger {
	return zap.NewRaw(zap.WriteTo(ioutil.Discard))
}
