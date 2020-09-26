package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/radovskyb/watcher"
	"gopkg.in/ini.v1"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"text/template"
	"time"
	"unicode"
)

type TypeVars map[string]interface{}

var actionsList *ini.File
var useActionsList bool

func checkRegexpMatch(pattern, str string) bool {
	pattern = fmt.Sprintf("^%s$", pattern)
	matched, _ := regexp.Match(pattern, []byte(str))
	return matched
}

func checkAndRunActionsByEvent(event *watcher.Event) {
	operation := strings.ToLower(fmt.Sprintf("%s", event.Op))
	// сначала поиск по полному пути
	cmd := actionsList.Section(event.Path).Key(operation).Value()
	if cmd == "" {
		// если нет полного пути, то поиск чисто по имени файла или папки
		cmd = actionsList.Section(event.Name()).Key(operation).Value()
	}
	// поиск по регулярке. сначала полный путь, затем чисто имя файла подставляется
	// для регулярки нужно использовать префикс ~
	var secName string
	if cmd == "" {
		for _, section := range actionsList.Sections() {
			secName = strings.TrimSpace(section.Name())

			if strings.HasPrefix(secName, "~") && (checkRegexpMatch(secName[1:], event.Name()) || checkRegexpMatch(secName[1:], event.Path)) {
				cmd = actionsList.Section(secName).Key(operation).Value()
			}
		}
	}

	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return // выход, если не найдено никак!
	}

	// replace placeholders!
	tpl := template.Must(template.New("").Parse(cmd))

	// add vars
	vars := make(TypeVars)
	vars["event"] = event
	vars["file"] = event.FileInfo
	vars["filename"] = event.Name()
	vars["path"] = event.Path
	vars["oldpath"] = event.OldPath
	vars["operation"] = fmt.Sprintf("%s", event.Op)
	vars["section"] = secName

	var result bytes.Buffer
	if err := tpl.Execute(&result, vars); err != nil {
		fmt.Printf("Ошибка разбора шаблона команды %s: %s\n", cmd, err.Error())
	}
	cmd = result.String()

	// exec cmd!
	var cmdName string
	var cmdArgs []string
	splits := strings.FieldsFunc(cmd, unicode.IsSpace)
	cmdName = splits[0]
	if len(splits) > 1 {
		cmdArgs = splits[1:]
	}

	fmt.Printf("Trying to exec: %s...\n", cmd)
	c := exec.Command(cmdName, cmdArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		if c.ProcessState == nil || !c.ProcessState.Success() {
			log.Println(err)
		}
	}
}

func main() {
	interval := flag.String("interval", "500ms", "интервал проверки изменений")
	recursive := flag.Bool("recursive", true, "следить за директориями рекурсивно")
	dotfiles := flag.Bool("dotfiles", false, "следить за скрытыми файлами")
	cmd := flag.String("cmd", "", "команда, запускаемая при возникновении событий")
	startcmd := flag.Bool("startcmd", false, "запустить команду cmd при старте приложения и начале слежения")
	listFiles := flag.Bool("list", true, "показать список файлов для наблюдения при старте")
	stdinPipe := flag.Bool("pipe", false, "передать информацию о событии в stdin команды")
	keepalive := flag.Bool("keepalive", true, "продолжать работу, даже если cmd вернула код возврата != 0")
	ignore := flag.String("ignore", "", "список игнорируемых файлов (через запятую)")
	logevents := flag.Bool("logevents", true, "выводить в stdout изменения в файлах и директориях")
	actionListFilePath := flag.String("actions", "action_list.ini", "путь к файлу со списком фильтров для файлов, событий и их действий")

	flag.Parse()

	// Retrieve the list of files and folders.
	files := flag.Args()

	// If no files/folders were specified, watch the current directory.
	if len(files) == 0 {
		curDir, err := os.Getwd()
		if err != nil {
			log.Fatalln(err)
		}
		files = append(files, curDir)
	}

	var err error
	actionsList, err = ini.Load(*actionListFilePath)
	useActionsList = true
	if err != nil {
		if *actionListFilePath != "" && *actionListFilePath != "action_list.ini" {
			fmt.Printf("Ошибка открытия файла со списком фильтров: %v. Фильтры и действия не будут использоваться.\n", err)
		}
		useActionsList = false
	}
	if useActionsList {
		fmt.Printf("Загружен файл фильтров и действий: %s\n", *actionListFilePath)
	}

	var cmdName string
	var cmdArgs []string
	if *cmd != "" {
		split := strings.FieldsFunc(*cmd, unicode.IsSpace)
		cmdName = split[0]
		if len(split) > 1 {
			cmdArgs = split[1:]
		}
	}

	// Create a new Watcher with the specified options.
	w := watcher.New()
	w.IgnoreHiddenFiles(!*dotfiles)
	//w.FilterOps(watcher.Create)

	// Get any of the paths to ignore.
	ignoredPaths := strings.Split(*ignore, ",")

	for _, path := range ignoredPaths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}

		err := w.Ignore(trimmed)
		if err != nil {
			log.Fatalln(err)
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		for {
			select {
			case event := <-w.Event:
				// Print the event's info if enabled )
				if *logevents {
					fmt.Println(event)
				}

				if useActionsList {
					checkAndRunActionsByEvent(&event)
				}

				// Run the command if one was specified.
				if *cmd != "" {
					c := exec.Command(cmdName, cmdArgs...)
					if *stdinPipe {
						c.Stdin = strings.NewReader(event.String())
					} else {
						c.Stdin = os.Stdin
					}
					c.Stdout = os.Stdout
					c.Stderr = os.Stderr
					if err := c.Run(); err != nil {
						if (c.ProcessState == nil || !c.ProcessState.Success()) && *keepalive {
							log.Println(err)
							continue
						}
						log.Fatalln(err)
					}
				}
			case err := <-w.Error:
				if err == watcher.ErrWatchedFileDeleted {
					fmt.Println(err)
					continue
				}
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	// Add the files and folders specified.
	for _, file := range files {
		if *recursive {
			if err := w.AddRecursive(file); err != nil {
				log.Fatalln(err)
			}
		} else {
			if err := w.Add(file); err != nil {
				log.Fatalln(err)
			}
		}
	}

	// Print a list of all of the files and folders being watched.
	if *listFiles {
		fmt.Printf("Файлы и папки для слежения: \n")
		for path, f := range w.WatchedFiles() {
			fmt.Printf("%s: %s\n", path, f.Name())
		}
		fmt.Println()
	}

	fmt.Printf("Объектов под наблюдением: %d\n", len(w.WatchedFiles()))

	// Parse the interval string into a time.Duration.
	parsedInterval, err := time.ParseDuration(*interval)
	if err != nil {
		log.Fatalln(err)
	}

	closed := make(chan struct{})

	c := make(chan os.Signal)
	signal.Notify(c, os.Kill, os.Interrupt)
	go func() {
		<-c
		w.Close()
		<-done
		fmt.Println("Watcher остановлен!")
		close(closed)
	}()

	// Run the command before watcher starts if one was specified.
	go func() {
		if *cmd != "" && *startcmd {
			c := exec.Command(cmdName, cmdArgs...)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				if (c.ProcessState == nil || !c.ProcessState.Success()) && *keepalive {
					log.Println(err)
					return
				}
				log.Fatalln(err)
			}
		}
	}()

	// Start the watching process.
	if err := w.Start(parsedInterval); err != nil {
		log.Fatalln(err)
	}

	<-closed
}
