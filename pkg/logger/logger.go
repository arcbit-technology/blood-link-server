package logger

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	errFileLogger           *logrus.Logger
	errConsoleLogger        *logrus.Logger
	accessLogsFileLogger    *logrus.Logger
	accessLogsConsoleLogger *logrus.Logger
	mu                      *sync.Mutex
}

var (
	logger *Logger
	once   sync.Once
)

func GetLogger() *Logger {
	once.Do(func() {
		logger = &Logger{}
		logger.initialize()
	})
	return logger
}

func (l *Logger) initialize() {
	l.mu = &sync.Mutex{}
	l.errFileLogger = logrus.New()
	l.errConsoleLogger = logrus.New()
	l.accessLogsConsoleLogger = logrus.New()
	l.accessLogsFileLogger = logrus.New()

	l.errConsoleLogger.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	l.errConsoleLogger.SetOutput(os.Stdout)

	l.errFileLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		DisableColors: true,
	})

	path := filepath.Join(".", "errors.log")
	logFile, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		l.errConsoleLogger.Fatalf("Failed to open log file: %v", err)
	}

	l.errFileLogger.SetOutput(logFile)

	l.accessLogsConsoleLogger.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	l.accessLogsConsoleLogger.SetOutput(os.Stdout)

	l.accessLogsFileLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		DisableColors: true,
	})

	path = filepath.Join(".", "access.log")
	logFile, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		l.errConsoleLogger.Fatalf("Failed to open log file: %v", err)
	}

	l.accessLogsFileLogger.SetOutput(logFile)
}

// initialize sets up the loggers for file and console
func (l *Logger) LogErrors(errs error, additionalInfo map[string]interface{}) {
	l.mu.Lock()

	defer l.mu.Unlock()

	_, file, line, _ := runtime.Caller(1)

	fields := logrus.Fields{
		"error": errs.Error(),
		"file":  file,
		"line":  line,
	}

	for key, value := range additionalInfo {
		fields[key] = value
	}

	l.errFileLogger.WithFields(fields).Error()
	l.errConsoleLogger.WithFields(fields).Error()
}

// LogInfo logs informational messages to both file and console
func (l *Logger) LogInfo(message string, fields logrus.Fields) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.accessLogsConsoleLogger.WithFields(fields).Info(message)
	l.accessLogsFileLogger.WithFields(fields).Info(message)

}

// LogWarn logs warning messages to both file and console
func (l *Logger) LogWarn(message string, fields logrus.Fields) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.accessLogsConsoleLogger.WithFields(fields).Warn(message)
	l.accessLogsFileLogger.WithFields(fields).Warn(message)
}
