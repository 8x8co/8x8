package xl

import "go.uber.org/zap"

var Logger *zap.Logger

func init() {
	var err error
	c := zap.NewProductionConfig()
	c.DisableStacktrace = true
	c.Level.SetLevel(zap.DebugLevel)
	Logger, err = c.Build(
		zap.WithCaller(false),
	)
	if err != nil {
		panic(err)
	}
}

// Info logs info
func Info(msg string, f ...zap.Field) {
	Logger.Info(msg, f...)
}

func Debug(msg string, f ...zap.Field) {
	Logger.Debug(msg, f...)
}

func Error(err error, msg string, f ...zap.Field) {
	Logger.Error(msg, append(f, zap.String("error", err.Error()))...)
}