package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func init() {
	log.SetFlags(0)
}

func main() {
	if len(os.Args) <= 1 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "i":
		config, err := LoadConfig(os.Args[2])
		if err != nil {
			log.Fatalln("ошибка при чтении конфига:", err)
		}

		err = config.IncrementalBackup()
		if err != nil {
			config.fatalln("ошибка инкрементального бекапа:", err)
		}
	case "f":
		config, err := LoadConfig(os.Args[2])
		if err != nil {
			log.Fatalln(err)
		}

		err = config.FullBackup()
		if err != nil {
			config.fatalln("ошибка полного бекапа:", err)
		}
	case "s":
		config, err := LoadConfig(os.Args[2])
		if err != nil {
			log.Fatalln("ошибка при чтении конфига:", err)
		}
		config.logf(LogLevelProgress, "Поиск файлов по маске %s...\n", os.Args[3])

		config.logf(LogLevelProgress, "Создание индекса...\n")
		idx, err := config.FindAll(os.Args[3])
		if err != nil {
			config.fatalln("ошибка поиска:", err)
		}
		config.logf(LogLevelProgress, "Создание индекса завершено.\n")

		fmt.Println(idx)
	case "r":
		config, err := LoadConfig(os.Args[2])
		if err != nil {
			log.Fatalln(err)
		}

		/*idx, err := config.FindAll(os.Args[3])
		if err != nil {
			config.fatalln(err)
		}*/

		t, err := time.Parse("02.01.2006 15:04", os.Args[4])
		if err != nil {
			config.fatalln("ошибка парсинга времени:", err)
		}

		//

		b := &Backuper{Config: config}
		plan, err := b.extractionPlan(os.Args[3], t)
		if err != nil {
			log.Fatalln(err)
		}
		err = b.extract(plan, os.Args[5])
		if err != nil {
			log.Fatalln(err)
		}
	case "t":
		config, err := LoadConfig(os.Args[2])
		if err != nil {
			log.Fatalln(err)
		}
		err = config.Test()
		if err != nil {
			log.Fatalln("ошибка тестирования:", err)
		}
		log.Println("Ошибок нет.")
	default:
		printUsage()
	}
}

func printUsage() {
	bin := filepath.Base(os.Args[0])

	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "%s i <config file path> - do incremental backup\n", bin)
	fmt.Fprintf(os.Stderr, "%s f <config file path> - do full backup\n", bin)
	fmt.Fprintf(os.Stderr, "%s s <config file path> <mask> - search file(s) in backup\n", bin)
	fmt.Fprintf(os.Stderr, "%s r <config file path> <mask> <dd.mm.yyyy hh:mm> <path> - recover file(s) from backup\n", bin)
	fmt.Fprintf(os.Stderr, "%s t <config file path> - test archive for errors\n", bin)
}
