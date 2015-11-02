package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/codegangsta/cli"
)

type Store struct {
	file    *os.File
	entries map[string]int
}

func (this Store) Inc(cmd string) {
	key := strings.TrimSpace(cmd)
	weight, _ := this.entries[key]
	this.entries[key] = weight + 1
}

func (this Store) GetWeight(line string) int {
	cmd := strings.Split(line, " ")[0]
	weight, _ := this.entries[cmd]

	return weight
}

type entries []entry

func (this entries) Len() int {
	return len(this)
}
func (this entries) Swap(i, j int) { this[i], this[j] = this[j], this[i] }

type byWeightAndName struct{ entries }
type entry struct {
	line   string
	weight int
}

func (this byWeightAndName) Less(i, j int) bool {
	left := this.entries[i]
	right := this.entries[j]

	if left.weight != right.weight {
		return left.weight >= right.weight
	}

	return left.line <= right.line
}

func (this Store) Sort(lines []string) []string {
	weightedLines := make(entries, len(lines))

	for index, line := range lines {
		weightedLines[index] = entry{
			line:   line,
			weight: this.GetWeight(line),
		}
	}

	sort.Sort(byWeightAndName{weightedLines})

	for index, sorted := range weightedLines {
		lines[index] = sorted.line
	}

	return lines
}

func OpenStore() (Store, error) {
	store := Store{
		entries: make(map[string]int, 100),
	}
	filename := os.ExpandEnv("$HOME/.pcmd.txt")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		return store, err
	}

	store.file = file

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		splitPoint := strings.LastIndex(line, ":")

		if splitPoint == -1 {
			continue
		}

		command := line[0:splitPoint]
		weight, _ := strconv.Atoi(line[splitPoint+1:])

		store.entries[command] = weight
	}

	return store, scanner.Err()
}

func (this Store) Save() error {
	if err := this.file.Truncate(0); err != nil {
		return err
	}

	if _, err := this.file.Seek(0, 0); err != nil {
		return err
	}

	for command, weight := range this.entries {
		_, err := fmt.Fprintf(this.file, "%v:%v\n", command, weight)

		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		cli.Command{
			Name: "sort",
			Action: func(ctx *cli.Context) {
				lines := make([]string, 0, 100)

				scanner := bufio.NewScanner(os.Stdin)
				scanner.Split(bufio.ScanLines)

				for scanner.Scan() {
					line := scanner.Text()
					if len(line) > 0 {
						lines = append(lines, line)
					}
				}

				if err := scanner.Err(); err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}

				store, err := OpenStore()
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}

				lines = store.Sort(lines)
				for _, line := range lines {
					fmt.Fprintln(os.Stdout, line)
				}
			},
		},
		cli.Command{
			Name:  "add",
			Usage: "puts a command",
			Action: func(ctx *cli.Context) {
				if len(ctx.Args()) == 0 {
					fmt.Sprintln("missing argument")
					os.Exit(1)
				}
				if len(ctx.Args()) > 1 {
					fmt.Sprintln("multiple arguments")
					os.Exit(1)
				}

				cmd := ctx.Args()[0]
				if cmd == "-" {
					in, err := ioutil.ReadAll(os.Stdin)
					if err != nil {
						fmt.Sprintf("stdin read error: %v", err.Error())
						os.Exit(1)
					}

					cmd = string(in)
				}

				store, err := OpenStore()
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}

				store.Inc(cmd)

				if err := store.Save(); err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}

				fmt.Println(cmd)
			},
		},
	}

	app.RunAndExitOnError()
}
