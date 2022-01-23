package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	var dirPath string
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
			Destination: &dirPath,
		},
	}
	app.Action = func(c *cli.Context) error {
		if dirPath == "" {
			os.Exit(2)
		}
		return Process(dirPath)
	}
	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}

func IsFileExist(filepath string) bool {
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		if os.IsExist(err) {
			return true
		} else {
			return false
		}
	}
	if fileInfo.IsDir() {
		return false
	}
	return true
}

func Process(dirPath string) error {
	// TODO 应该先将整个项目复制一遍，再基于复制后的目录进行操作
	// 读取main.go
	mainFilePath := filepath.Join(dirPath, "main.go")
	if !IsFileExist(mainFilePath) {
		return errors.New("文件`main.go`不存在")
	}
	mainFileBs, err := os.ReadFile(mainFilePath)
	if err != nil {
		return err
	}
	mainFileLines := strings.Split(string(mainFileBs), "\n")
	// 获取后端模块的路径
	originalModulePath := filepath.Join(dirPath, FindAndParseHandlerModuleMarks(mainFileLines))
	// 遍历查找是否有XDF校验文件
	var XDFSumFilePath string
	err = filepath.Walk(originalModulePath, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(info.Name(), ".ximd.sum") {
			XDFSumFilePath = path
		}
		return nil
	})
	if err != nil {
		return err
	}
	// 如果已有XDF
	if XDFSumFilePath != "" {
		// 读取其内容
		bs, err := os.ReadFile(XDFSumFilePath)
		if err != nil {
			return err
		}
		hashStrA := string(bs)
		// 计算XDF的校验和
		hashStrB, err := XDFSum(originalModulePath)
		if err != nil {
			return err
		}
		// 对比两个md5是否一致，如果不一致，重新计算并写入
		if hashStrA != hashStrB {
			err = GenerateAllXDF(originalModulePath, XDFSumFilePath)
			if err != nil {
				return err
			}
		}
	} else {
		// 如果没有校验文件，生成
		err := GenerateAllXDF(originalModulePath, filepath.Join(originalModulePath, ".ximd.sum"))
		if err != nil {
			return err
		}
	}
	// TODO 修改main.go中的包名，生成ximGenerate包
	return nil
}

func GenerateAllXDF(originalModulePath string, XDFSumFilePath string) error {
	err := filepath.Walk(originalModulePath, func(path string, info fs.FileInfo, err error) error {
		err = ScanAndGenXDF(path)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil
	}
	hash, err := XDFSum(originalModulePath)
	if err != nil {
		return err
	}
	err = os.WriteFile(XDFSumFilePath, []byte(hash), 0600)
	if err != nil {
		return err
	}
	return nil
}

func XDFSum(originalModulePath string) (string, error) {
	var XDFPaths []string
	err := filepath.Walk(originalModulePath, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(info.Name(), ".ximd") {
			XDFPaths = append(XDFPaths, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	var XDFBuf bytes.Buffer
	for _, XDFPath := range XDFPaths {
		bs, err := os.ReadFile(XDFPath)
		if err != nil {
			return "", err
		}
		XDFBuf.Write(bs)
	}
	hash := md5.Sum(XDFBuf.Bytes())
	hashStr := hex.EncodeToString(hash[:])
	return hashStr, nil
}

func ModifyModulePath(originalModuleNameWithPrefix string, lines []string) {
	originalModuleNameWithPrefix = strings.TrimPrefix(originalModuleNameWithPrefix, "./")
	re := regexp.MustCompile("^\\s*backend \"(?P<Path>\\w+(/\\w+)*)\"")
	for i, line := range lines {
		matches := re.FindStringSubmatch(line)
		groups := re.SubexpNames()
		var originalModulePath string
		for i, groupName := range groups {
			if groupName == "Path" {
				originalModulePath = matches[i]
				break
			}
		}
		newModulePath := strings.Replace(originalModulePath, originalModuleNameWithPrefix, "ximGenerate", 1)
		lines[i] = "backend \"" + newModulePath + "\""
	}
}

func GenerateModule(targetPath string, sigsMap map[string][]*HandlerFuncSig) error {
	_, err := os.Stat(targetPath)
	if err != nil && !os.IsExist(err) {
		err = os.Mkdir(targetPath, 0600)
		if err != nil {
			return err
		}
	}
	packageTemplate, err := ReadTemplateFile("package")
	if err != nil {
		return err
	}
	importTemplate, err := ReadTemplateFile("import")
	if err != nil {
		return err
	}
	funcTemplate, err := ReadTemplateFile("func")
	if err != nil {
		return err
	}
	for filename, sigs := range sigsMap {
		err := os.WriteFile(filepath.Join(targetPath, filename), []byte(GenerateFileContentFromSigs(packageTemplate, importTemplate, funcTemplate, sigs)), 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadTemplateFile(templateName string) (string, error) {
	file, err := os.Open(fmt.Sprintf("./res/%s_template.txt", templateName))
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()
	template, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(template), nil
}

func GenerateFileContentFromSigs(packageTemplate, importTemplate, funcTemplate string, sigs []*HandlerFuncSig) string {
	var contentBuilder strings.Builder
	packageStr := strings.Replace(packageTemplate, "%PackageName%", "ximGenerate", 1)
	importStr := importTemplate
	var funcsBuilder strings.Builder
	for _, sig := range sigs {
		funcStr := strings.Replace(funcTemplate, "%FuncName%", sig.FuncName, 1)
		funcStr = strings.Replace(funcStr, "%ResultType%", sig.ResultType, 1)
		funcStr = strings.Replace(funcStr, "%URN%", sig.URN, 1)
		var argsBuilder strings.Builder
		for i, argType := range sig.ArgTypes {
			argsBuilder.WriteString("arg" + strconv.Itoa(i) + " " + argType)
			if i != len(sig.ArgTypes)-1 {
				argsBuilder.WriteString(", ")
			}
		}
		funcStr = strings.Replace(funcStr, "%Args%", argsBuilder.String(), 1)
		funcsBuilder.WriteString(funcStr)
	}
	contentBuilder.WriteString(packageStr)
	contentBuilder.WriteString(importStr)
	contentBuilder.WriteString(funcsBuilder.String())
	return contentBuilder.String()
}

func FindAndParseHandlerModuleMarks(lines []string) string {
	re := regexp.MustCompile("^//xim:HandlerModule\\(\"(?P<path>\\./(\\w+)(/\\w+)*)\"\\)")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 0 {
			continue
		}
		groups := re.SubexpNames()
		for i, group := range groups {
			if group == "path" {
				return matches[i]
			}
		}
	}
	return ""
}

// ScanAndGenXDF
// XDF = Xim Declaration File
func ScanAndGenXDF(filepath string) error {
	bs, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(bs), "\n")
	marks := FindAndParseHandlerFuncMarks(lines)
	var sigs []*HandlerFuncSig
	for lineIndex, URN := range marks {
		line := lines[lineIndex]
		sig, err := AnalyzeHandlerFuncSig(URN, line)
		if err != nil {
			return err
		}
		sigs = append(sigs, sig)
	}
	err = GenerateXDF(filepath, sigs)
	if err != nil {
		return err
	}
	return nil
}

func FindAndParseHandlerFuncMarks(lines []string) (marks map[int]string) {
	marks = make(map[int]string)
	re := regexp.MustCompile("^\\s*//xim:HandlerFunc\\(\"(?P<path>(/[^/\\s]+)+)\"\\)")
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
func AnalyzeHandlerFuncSig(URN, line string) (*HandlerFuncSig, error) {
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
		URN:        URN,
		FuncName:   funcName,
		ArgTypes:   argTypes,
		ResultType: resultType,
	}, nil
}

func GenerateXDF(filepath string, sigs []*HandlerFuncSig) error {
	bs, err := json.Marshal(sigs)
	err = os.WriteFile(filepath+".ximd", bs, 0600)
	if err != nil {
		return err
	}
	return nil
}

type HandlerFuncSig struct {
	URN        string
	FuncName   string
	ArgTypes   []string
	ResultType string
}
