package main

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

func TestFindMarks(t *testing.T) {
	bytes, err := ioutil.ReadFile("./find_marks.txt")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	lines := strings.Split(string(bytes), "\n")
	marks := FindAndParseMarks(lines)
	fmt.Println(marks)
	if !reflect.DeepEqual(marks, map[int]string{1: "/A", 3: "/B", 5: "/C", 7: "/D", 14: "/H\"/H"}) {
		t.Fail()
	}
}
