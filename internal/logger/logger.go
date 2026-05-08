package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func Init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.OutputPaths = []string{"stdout"}
	
	var err error
	Log, err = config.Build()
	if err != nil {
		println("failed to initialize logger: " + err.Error())
		os.Exit(1)
	}
}

func Sync() {
	_ = Log.Sync()
}
