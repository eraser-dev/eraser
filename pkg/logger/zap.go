package logger

import (
	"flag"
	"fmt"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	crzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var logLevel = flag.String("log-level", zapcore.InfoLevel.String(),
	fmt.Sprintf("Log verbosity level. Supported values (in order of detail) are %q, %q, %q, and %q.",
		zapcore.DebugLevel.String(),
		zapcore.InfoLevel.String(),
		zapcore.WarnLevel.String(),
		zapcore.ErrorLevel.String()))

// GetLevel gets the configured log level.
func GetLevel() string {
	return *logLevel
}

// Configure configures a singleton logger for use from controller-runtime.
func Configure() error {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(*logLevel)); err != nil {
		return fmt.Errorf("unable to parse log level: %w: %s", err, *logLevel)
	}

	var logger logr.Logger
	switch zapLevel {
	case zap.DebugLevel:
		cfg := zap.NewDevelopmentEncoderConfig()
		logger = crzap.New(crzap.UseDevMode(true), crzap.Encoder(zapcore.NewConsoleEncoder(cfg)), crzap.Level(zapLevel))
		ctrl.SetLogger(logger)
		klog.SetLogger(logger)
	default:
		cfg := zap.NewProductionEncoderConfig()
		logger = crzap.New(crzap.UseDevMode(false), crzap.Encoder(zapcore.NewJSONEncoder(cfg)), crzap.Level(zapLevel))
	}

	ctrl.SetLogger(logger)
	klog.SetLogger(logger)

	return nil
}
