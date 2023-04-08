package main

import (
	"log"
	"os"
	"path/filepath"
)

func init() {
	log.SetFlags(0)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "f":
		config, err := LoadConfig(os.Args[2])
		if err != nil {
			log.Fatalln(err)
		}

		err = config.FullBackup()
		if err != nil {
			log.Fatalln(err)
		}
	case "i":
		config, err := LoadConfig(os.Args[2])
		if err != nil {
			log.Fatalln(err)
		}

		err = config.IncrementalBackup()
		if err != nil {
			log.Fatalln(err)
		}
	case "s":
		config, err := LoadConfig(os.Args[2])
		if err != nil {
			log.Fatalln("read config error:", err)
		}

		idx, err := config.FindAll(os.Args[3])
		if err != nil {
			config.fatalln("search error:", err)
		}

		idx.ViewFileVersions(os.Stdout)
	case "r":
		config, err := LoadConfig(os.Args[2])
		if err != nil {
			log.Fatalln(err)
		}

		t, err := parseTime(os.Args[4])
		if err != nil {
			config.fatalln(err)
		}

		plan, err := config.extractionPlan(os.Args[3], t)
		if err != nil {
			log.Fatalln(err)
		}
		err = config.extract(plan, os.Args[5])
		if err != nil {
			log.Fatalln(err)
		}
	case "t":
		config, err := LoadConfig(os.Args[2])
		if err != nil {
			log.Fatalln(err)
		}

		_, err = config.index(true)
		if err != nil {
			log.Fatalln(err)
		}
	default:
		printUsage()
	}
}

func printUsage() {
	bin := filepath.Base(os.Args[0])

	log.Print("Usage:\n")
	log.Printf("%s i <config file path> - do incremental backup\n", bin)
	log.Printf("%s f <config file path> - do full backup\n", bin)
	log.Printf("%s s <config file path> <mask> - search file(s) in backup\n", bin)
	log.Printf("%s r <config file path> <mask> <dd.mm.yyyy hh:mm> <path> - recover file(s) from backup\n", bin)
	log.Printf("%s t <config file path> - test archive for errors\n", bin)
}
