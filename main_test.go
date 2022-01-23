package main

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

func TestFindAndParseHandlerFuncMarks(t *testing.T) {
	bytes, err := ioutil.ReadFile("./find_func_marks.txt")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	lines := strings.Split(string(bytes), "\n")
	marks := FindAndParseHandlerFuncMarks(lines)
	fmt.Println(marks)
	if !reflect.DeepEqual(marks, map[int]string{1: "/A", 3: "/B", 5: "/C", 7: "/D", 14: "/H\"/H"}) {
		t.Fail()
	}
}

func TestAnalyzeHandlerFuncSig(t *testing.T) {
	sig, err := AnalyzeHandlerFuncSig("example.com/get", "func ConcatStrings(a string, b string) string {")
	if err != nil {
		fmt.Println(err)
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
}

func TestGenerateXDF(t *testing.T) {
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
}

func TestFindAndParseHandlerModuleMarks(t *testing.T) {
	bytes, err := ioutil.ReadFile("./find_module_mark.go.txt")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	lines := strings.Split(string(bytes), "\n")
	path := FindAndParseHandlerModuleMarks(lines)
	fmt.Println(path)
	if path != "./backend" {
		t.Fail()
	}
}

func TestGenerateFileContentFromSigs(t *testing.T) {
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
	str := GenerateFileContentFromSigs(packageTemplate, importTemplate, funcTemplate, []*HandlerFuncSig{
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
}

func TestGenerateModule(t *testing.T) {
	err := GenerateModule("./tempGen", map[string][]*HandlerFuncSig{
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
}

func TestModifyModulePath(t *testing.T) {
	lines := []string{
		"backend \"main/backend\"",
	}
	ModifyModulePath("./backend", lines)
	fmt.Println(lines)
	if !reflect.DeepEqual(lines, []string{
		"backend \"main/ximGenerate\"",
	}) {
		t.Fail()
	}
}

func TestProcess(t *testing.T) {

}
