package main

import (
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

var filePath string

func main() {
	app := cli.NewApp()
	app.Name = "XimCodeGenerator"
	app.Usage = "为使用Xim框架的程序生成代码"
	app.Author = "Kogic"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "path, p",
			Usage:       "需要生成的源文件的路径",
			Value:       "",
			Destination: &filePath,
		},
	}
	app.Action = func(c *cli.Context) error {
		if filePath == "" {
			os.Exit(2)
		}
		return Process(filePath)
	}
	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}

func Process(filePath string) error {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(bytes), "\n")
	marks := FindAndParseMarks(lines)
	err = AnalyzeFuncSignature(marks)
	if err != nil {
		return err
	}
	return nil
}

func FindAndParseMarks(lines []string) (marks map[int]string) {
	marks = make(map[int]string)
	re := regexp.MustCompile("^\\s*//xim:handler\\(\"(?P<path>(/[^/\\s]+)+)\"\\)")
	for lineIndex, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 0 {
			continue
		}
		groups := re.SubexpNames()
		for i, group := range groups {
			if group == "path" {
				marks[lineIndex] = matches[i]
			}
		}
	}
	return marks
}

//goland:noinspection GoUnusedParameter
func AnalyzeFuncSignature(marksLineIndex map[int]string) error {
	return nil
}
