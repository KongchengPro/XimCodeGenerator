package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const ximGenerate = "ximGenerate"

func main() {
	var dirPath string
	var host string
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
		cli.StringFlag{
			Name:        "host, h",
			Usage:       "主机名称",
			Value:       "",
			Destination: &host,
		},
	}
	app.Action = func(c *cli.Context) (err error) {
		Process(host, dirPath)
		return nil
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

func CopyDir(srcDirPath, dstDirPath string) error {
	srcDirPath = filepath.Clean(srcDirPath)
	dstDirPath = filepath.Clean(dstDirPath)
	return filepath.Walk(srcDirPath, func(path string, info fs.FileInfo, err error) error {
		path = filepath.Clean(path)
		if err != nil {
			return err
		}
		newPath := strings.Replace(path, srcDirPath, dstDirPath, 1)
		if info.IsDir() {
			err = os.MkdirAll(newPath, 0600)
			if err != nil {
				return err
			}
		} else {
			srcFile, err := os.Open(path)
			defer func() {
				_ = srcFile.Close()
			}()
			if err != nil {
				return err
			}
			dstFile, err := os.Create(strings.Replace(path, srcDirPath, dstDirPath, 1))
			defer func() {
				_ = dstFile.Close()
			}()
			_, err = io.Copy(dstFile, srcFile)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// Process
// srcDirPath最后不能跟有`/`
// TODO 自动化修改后端包名为main，将mainFunc修改为main
func Process(host, srcDirPath string) {
	genDirPath := filepath.Join(filepath.Dir(srcDirPath), filepath.Base(srcDirPath)+"_gen")
	err := CopyDir(srcDirPath, genDirPath)
	if err != nil {
		panic(err)
	}
	// 读取main.go
	mainFilePath := filepath.Join(genDirPath, "main.go")
	if !IsFileExist(mainFilePath) {
		panic(errors.New("文件`main.go`不存在"))
	}
	mainFileBs, err := os.ReadFile(mainFilePath)
	if err != nil {
		panic(err)
	}
	mainFileLines := strings.Split(string(mainFileBs), "\n")
	// 获取后端模块的路径
	handlerModuleMarkPath := FindAndParseHandlerModuleMarks(mainFileLines)
	originalModulePath := filepath.Join(srcDirPath, handlerModuleMarkPath)
	// 检查或创建XDF
	CheckOrGenerateXDF(originalModulePath)
	// 修改main.go中的包名
	ModifyModulePath(handlerModuleMarkPath, mainFileLines)
	mainFile, err := os.OpenFile(mainFilePath, os.O_WRONLY, 0600)
	defer func() {
		_ = mainFile.Close()
	}()
	if err != nil {
		panic(err)
	}
	for _, line := range mainFileLines {
		_, err := mainFile.WriteString(line + "\n")
		if err != nil {
			panic(err)
		}
	}
	// 生成ximGenerate包
	sigsMap, err := ReadSigsMapFromXDFs(originalModulePath)
	err = GenerateModule(host, filepath.Join(genDirPath, ximGenerate), sigsMap)
	if err != nil {
		panic(err)
	}
}

func ReadSigsMapFromXDFs(dirPath string) (map[string][]*HandlerFuncSig, error) {
	sigsMap := make(map[string][]*HandlerFuncSig)
	err := filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(info.Name(), ".ximd") {
			bs, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var sigs []*HandlerFuncSig
			err = json.Unmarshal(bs, &sigs)
			if err != nil {
				return err
			}
			sigsMap[strings.TrimSuffix(info.Name(), ".ximd")] = sigs
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return sigsMap, nil
}

// CheckOrGenerateXDF 检查或创建XDF
// 如果XDF校验文件存在，就检查，如果不一致，重新生成XDF
// 如果XDF校验文件不存在，生成XDF
func CheckOrGenerateXDF(dirPath string) {
	// 遍历查找是否有XDF校验文件
	var XDFSumFilePath string
	err := filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".ximd.sum") {
			XDFSumFilePath = path
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	// 如果已有XDFSum
	if XDFSumFilePath != "" {
		// 读取其内容
		bs, err := os.ReadFile(XDFSumFilePath)
		if err != nil {
			panic(err)
		}
		hashStrA := string(bs)
		// 计算XDF的校验和
		hashStrB, err := XDFSum(dirPath)
		if err != nil {
			panic(err)
		}
		// 对比两个md5是否一致，如果不一致，重新计算并写入
		if hashStrA != hashStrB {
			err = GenerateAllXDF(dirPath, XDFSumFilePath)
			if err != nil {
				panic(err)
			}
		}
	} else {
		// 如果没有校验文件，生成
		err := GenerateAllXDF(dirPath, filepath.Join(dirPath, ".ximd.sum"))
		if err != nil {
			panic(err)
		}
	}
}

func GenerateAllXDF(originalModulePath string, XDFSumFilePath string) error {
	err := filepath.Walk(originalModulePath, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			err = ScanAndGenXDF(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
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
	var buf bytes.Buffer
	err := filepath.Walk(originalModulePath, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(info.Name(), ".ximd") {
			bs, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			buf.Write(bs)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	hash := md5.Sum(buf.Bytes())
	hashStr := hex.EncodeToString(hash[:])
	return hashStr, nil
}

// ModifyModulePath
// originalModuleNameWithPrefix: xim:HandlerModule后面的那个
// **WARNING**: 该方法会修改lines
func ModifyModulePath(originalModuleNameWithPrefix string, lines []string) {
	originalModuleNameWithPrefix = strings.TrimPrefix(originalModuleNameWithPrefix, "./")
	re := regexp.MustCompile("^\\s*backend \"(?P<Path>\\w+(/\\w+)*)\"")
	for i, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 0 {
			continue
		}
		groups := re.SubexpNames()
		var originalModulePath string
		for i, groupName := range groups {
			if groupName == "Path" {
				originalModulePath = matches[i]
				break
			}
		}
		newModulePath := strings.Replace(originalModulePath, originalModuleNameWithPrefix, ximGenerate, 1)
		lines[i] = "backend \"" + newModulePath + "\""
	}
}

func GenerateModule(host, targetPath string, sigsMap map[string][]*HandlerFuncSig) error {
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
		err := os.WriteFile(filepath.Join(targetPath, filename), []byte(GenerateFileContentFromSigs(host, packageTemplate, importTemplate, funcTemplate, sigs)), 0600)
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

func GenerateFileContentFromSigs(host, packageTemplate, importTemplate, funcTemplate string, sigs []*HandlerFuncSig) string {
	var contentBuilder strings.Builder
	packageStr := strings.Replace(packageTemplate, "%PackageName%", ximGenerate, 1)
	importStr := importTemplate
	var funcsBuilder strings.Builder
	for _, sig := range sigs {
		funcStr := strings.Replace(funcTemplate, "%FuncName%", sig.FuncName, 1)
		funcStr = strings.Replace(funcStr, "%ResultType%", sig.ResultType, 1)
		URI := host + sig.URN
		var queryStrBuilder strings.Builder
		var argsBuilder strings.Builder
		var i int
		for argName, argType := range sig.Args {
			argsBuilder.WriteString(argName + " " + argType)
			queryStrBuilder.WriteString(argName)
			queryStrBuilder.WriteString("=")
			queryStrBuilder.WriteString("\"+")
			queryStrBuilder.WriteString(argName)
			queryStrBuilder.WriteString("+\"")
			if i != len(sig.Args)-1 {
				argsBuilder.WriteString(", ")
				queryStrBuilder.WriteString("&")
			}
			i++
		}
		funcStr = strings.Replace(funcStr, "%Args%", argsBuilder.String(), 1)
		funcStr = strings.Replace(funcStr, "%URI%", URI+"?"+queryStrBuilder.String(), 1)
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
		funcLine := lines[lineIndex+1]
		sig := AnalyzeHandlerFuncSig(URN, funcLine)
		if sig == nil {
			continue
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
func AnalyzeHandlerFuncSig(URN, line string) *HandlerFuncSig {
	// 匹配函数声明的正则
	// TODO 完善其他类型（目前只能匹配String和处理类型）
	re := regexp.MustCompile("func (?P<funcName>[^0-9\\W]\\w*)\\((?P<args>|(([^0-9\\W]\\w*) (string))(, ?([^0-9\\W]\\w*) (string))*)\\) (?P<resultType>string) {")
	matches := re.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil
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
	argsMap := make(map[string]string)
	for _, argEntry := range argsSlice {
		argEntryList := strings.SplitN(strings.TrimSpace(argEntry), " ", 2)
		argName := argEntryList[0]
		argType := argEntryList[1]
		argsMap[argName] = argType
	}
	return &HandlerFuncSig{
		URN:        URN,
		FuncName:   funcName,
		Args:       argsMap,
		ResultType: resultType,
	}
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
	Args       map[string]string
	ResultType string
}
