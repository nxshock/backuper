package main

import (
	"log"
)

type LogLevel int

const (
	LogLevelDebug    = iota // 0
	LogLevelProgress        // 1
	LogLevelInfo            // 2
	LogLevelWarning         // 3
	LogLevelCritical        // 4
)

type Logger struct {
	logger          *log.Logger
	MinimalLogLevel LogLevel
}

func (logger *Logger) log(logLevel LogLevel, a ...interface{}) {
	if logLevel < logger.MinimalLogLevel {
		return
	}

	logger.logger.Print(a...)
}

func (logger *Logger) logf(logLevel LogLevel, s string, a ...interface{}) {
	if logLevel < logger.MinimalLogLevel {
		return
	}

	logger.logger.Printf(s, a...)
}

func (logger *Logger) logln(logLevel LogLevel, a ...interface{}) {
	if logLevel < logger.MinimalLogLevel {
		return
	}

	logger.logger.Println(a...)
}

func (logger *Logger) fatalln(a ...interface{}) {
	logger.logger.Fatalln(a...)
}

func (b *Config) log(logLevel LogLevel, a ...interface{}) {
	b.logger.log(logLevel, a...)
}

func (b *Config) logf(logLevel LogLevel, s string, a ...interface{}) {
	b.logger.logf(logLevel, s, a...)
}

func (b *Config) fatalln(a ...interface{}) {
	b.logger.fatalln(a...)
}
