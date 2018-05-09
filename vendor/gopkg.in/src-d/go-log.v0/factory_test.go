package log

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/x-cray/logrus-prefixed-formatter"
)

func TestLoggerFactoryNew_TextWithForce(t *testing.T) {
	require := require.New(t)

	f := &LoggerFactory{Format: TextFormat, ForceFormat: true}
	l, err := f.New()
	require.NoError(err)

	logger, ok := l.(*logger)
	require.True(ok)
	require.IsType(&prefixed.TextFormatter{}, logger.Entry.Logger.Formatter)
}

func TestLoggerFactoryNew_JSON(t *testing.T) {
	require := require.New(t)

	f := &LoggerFactory{Format: JSONFormat, Level: InfoLevel}
	l, err := f.New()
	require.NoError(err)

	logger, ok := l.(*logger)
	require.True(ok)
	require.IsType(&logrus.JSONFormatter{}, logger.Entry.Logger.Formatter)
	require.Equal(logrus.InfoLevel, logger.Entry.Logger.Level)
}

func TestLoggerFactoryNew_Fields(t *testing.T) {
	require := require.New(t)

	js := `{"foo":"bar"}`
	f := &LoggerFactory{Format: TextFormat, Level: DebugLevel, Fields: js}
	l, err := f.New()
	require.NoError(err)

	logger, ok := l.(*logger)
	require.True(ok)
	require.Equal(logrus.DebugLevel, logger.Entry.Logger.Level)
	require.Equal(logrus.Fields{"foo": "bar"}, logger.Entry.Data)

}

func TestLoggerFactoryNew_Error(t *testing.T) {
	require := require.New(t)

	// invalid level
	f := &LoggerFactory{Level: "text"}
	_, err := f.New()
	require.Error(err)

	// invalid format
	f = &LoggerFactory{Level: InfoLevel, Format: "qux"}
	_, err = f.New()
	require.Error(err)

	// invalid json
	f = &LoggerFactory{Level: InfoLevel, Format: TextFormat, Fields: "qux"}
	_, err = f.New()
	require.Error(err)
}

func TestLoggerFactoryApply(t *testing.T) {
	require := require.New(t)

	f := &LoggerFactory{Format: TextFormat, ForceFormat: true, Level: DebugLevel}
	err := f.ApplyToLogrus()
	require.NoError(err)

	require.IsType(&prefixed.TextFormatter{}, logrus.StandardLogger().Formatter)
	require.Equal(logrus.DebugLevel, logrus.StandardLogger().Level)
}
