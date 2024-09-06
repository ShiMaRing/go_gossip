package utils

import (
	"fmt"
	"go_gossip/config"
	"io"
	"log/slog"
	"os"
)

var Logger *slog.Logger

func LogInit() {
	var writer io.Writer
	var opt slog.HandlerOptions
	if config.LoggerConfig.LogFile == "" {
		writer = os.Stdout
	} else {
		file, err := os.OpenFile(config.LoggerConfig.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("open log file failed, use stdout instead")
			writer = os.Stdout
		} else {
			writer = file
		}
	}
	//set the log level
	switch config.LoggerConfig.Level {
	case "debug":
		opt.Level = slog.LevelDebug
	case "info":
		opt.Level = slog.LevelInfo
	case "warn":
		opt.Level = slog.LevelWarn
	case "error":
		opt.Level = slog.LevelError
	default:
		opt.Level = slog.LevelInfo
	}
	opt.AddSource = true
	Logger = slog.New(slog.NewTextHandler(writer, &opt))
}
