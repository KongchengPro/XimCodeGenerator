package main

import (
	"fmt"
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
	for lineIndex := range marks {
		line := lines[lineIndex]
		_, err := AnalyzeHandlerFuncSig(line)
		if err != nil {
			return err
		}
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
func AnalyzeHandlerFuncSig(line string) (*HandlerFuncSig, error) {
	// 匹配函数声明的正则
	// TODO 完善其他类型（目前只能匹配String和处理类型）
	re := regexp.MustCompile("func (?P<funcName>[^0-9\\W]\\w*)\\((?P<args>|(([^0-9\\W]\\w*) (string))(, ?([^0-9\\W]\\w*) (string))*)\\) (?P<resultType>string) {")
	matches := re.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil, fmt.Errorf("给定的行不是函数声明：%s", strings.TrimRight(line, "\n"))
	}
	groups := re.SubexpNames()
	matchMap := make(map[string]string)
	for groupIndex, groupName := range groups {
		matchMap[groupName] = matches[groupIndex]
	}
	funcName := matchMap["funcName"]
	args := matchMap["args"]
	resultType := matchMap["resultType"]
	argsSlice := strings.Split(args, ",")
	var argTypes []string
	for _, argEntry := range argsSlice {
		argType := strings.SplitN(strings.TrimSpace(argEntry), " ", 2)[1]
		argTypes = append(argTypes, argType)
	}
	return &HandlerFuncSig{
		FuncName:   funcName,
		ArgTypes:   argTypes,
		ResultType: resultType,
	}, nil
}

type HandlerFuncSig struct {
	FuncName   string
	ArgTypes   []string
	ResultType string
}
