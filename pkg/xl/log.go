package xl

import "go.uber.org/zap"

var Logger *zap.Logger

const LogPath = "/var/log/8x8.log"

func init() {
	var err error
	c := zap.NewProductionConfig()
	c.OutputPaths = []string{LogPath}
	c.ErrorOutputPaths = []string{LogPath}
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
