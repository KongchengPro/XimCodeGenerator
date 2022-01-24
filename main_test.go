package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFindAndParseHandlerFuncMarks(t *testing.T) {
	bs, err := ioutil.ReadFile("./find_func_marks.txt")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	lines := strings.Split(string(bs), "\n")
	marks := FindAndParseHandlerFuncMarks(lines)
	fmt.Println(marks)
	if !reflect.DeepEqual(marks, map[int]string{1: "/A", 3: "/B", 5: "/C", 7: "/D", 14: "/H\"/H"}) {
		t.Fail()
	}
}

/*func TestAnalyzeHandlerFuncSig(t *testing.T) {
	sig := AnalyzeHandlerFuncSig("example.com/get", "func ConcatStrings(a string, b string) string {")
	if sig == nil {
		t.Fail()
	}
	fmt.Println(sig)
	if !reflect.DeepEqual(sig, &HandlerFuncSig{
		URN:      "example.com/get",
		FuncName: "ConcatStrings",
		ArgTypes: []string{
			"string",
			"string",
		},
		ResultType: "string",
	}) {
		t.Fail()
	}
}*/

/*func TestGenerateXDF(t *testing.T) {
	err := GenerateXDF("./temp.go", []*HandlerFuncSig{
		{
			URN:      "example.com/get",
			FuncName: "ConcatStrings",
			ArgTypes: []string{
				"string",
				"string",
			},
			ResultType: "string",
		},
	})
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
}*/

func TestFindAndParseHandlerModuleMarks(t *testing.T) {
	bs, err := ioutil.ReadFile("./find_module_mark.go.txt")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	lines := strings.Split(string(bs), "\n")
	path := FindAndParseHandlerModuleMarks(lines)
	fmt.Println(path)
	if path != "./backend" {
		t.Fail()
	}
}

/*func TestGenerateFileContentFromSigs(t *testing.T) {
	packageTemplate, err := ReadTemplateFile("package")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	importTemplate, err := ReadTemplateFile("import")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	funcTemplate, err := ReadTemplateFile("func")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	str := GenerateFileContentFromSigs("", packageTemplate, importTemplate, funcTemplate, []*HandlerFuncSig{
		{
			URN:      "example.com/get",
			FuncName: "ConcatStrings",
			ArgTypes: []string{
				"string",
				"string",
			},
			ResultType: "string",
		},
	})
	fmt.Println(str)
}*/

/*func TestGenerateModule(t *testing.T) {
	err := GenerateModule("", "./temp_gen", map[string][]*HandlerFuncSig{
		"main.go": {
			{
				URN:      "example.com/get",
				FuncName: "ConcatStrings",
				ArgTypes: []string{
					"string",
					"string",
				},
				ResultType: "string",
			},
		},
	})
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
}*/

func TestModifyModulePath(t *testing.T) {
	lines := []string{
		"backend \"main/backend\"",
	}
	ModifyModulePath("./backend", lines)
	fmt.Println(lines)
	if !reflect.DeepEqual(lines, []string{
		"backend \"main/" + ximGenerate + "\"",
	}) {
		t.Fail()
	}
}

func TestCopyDir(t *testing.T) {
	err := CopyDir("./test_project/src", "./test_project/gen")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	hashA, err := CheckDirSum("./test_project/src")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	hashB, err := CheckDirSum("./test_project/gen")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	if hashA != hashB {
		t.Fail()
	}
}

func CheckDirSum(dirPath string) (string, error) {
	var buf bytes.Buffer
	err := filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			relPath := strings.TrimPrefix(filepath.Clean(path), filepath.Clean(dirPath))
			//fmt.Println(relPath)
			buf.WriteString(relPath)
		} else {
			//fmt.Println(strings.TrimPrefix(filepath.Clean(path), filepath.Clean(dirPath)))
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

func TestProcess(t *testing.T) {
	Process("localhost:8080", "./test_project/src")
}
