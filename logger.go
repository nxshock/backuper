package main

import "log"

type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
)

func (b *Config) log(l LogLevel, a ...interface{}) {
	if l < b.LogLevel {
		return
	}

	log.Print(a...)
}

func (b *Config) logf(l LogLevel, s string, a ...interface{}) {
	if l < b.LogLevel {
		return
	}

	log.Printf(s, a...)
}

func (b *Config) fatalln(a ...interface{}) {
	log.Fatalln(a...)
}
