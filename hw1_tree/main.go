package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	//"strings"
    //"bufio"
    "sort"
)

func Filter(vs []string, cond func(string)bool) []string {
    res := make([]string, 0)
    for _, v := range vs {
        if cond(v) {
            res = append(res, v)
        }
    }
    return res
}

func SizeFormat(size int64)string {
    if size == 0 {
        return "empty"
    }
    return fmt.Sprintf("%vb", size)
}

func dirSubTree(out io.Writer, path string, printFiles bool, prefix string) error {
    // Open directory
    file, err := os.Open(path)
    defer file.Close()
    if err != nil {
        return fmt.Errorf("Cannot open directory %v (%v)", path, err)
    }

    // Read list of files
    fileList, err := file.Readdirnames(0)
    if err != nil {
        return fmt.Errorf("Cannot read directory %v", path)
    }

    if !printFiles {
        fileList = Filter(fileList, func(s string)bool {
            info, _ := os.Lstat(filepath.Join(path, s))
            return info.IsDir()
        })
    }

    sort.Strings(fileList)
    listLen := len(fileList)

    // Helper prefix functions
    getPrefix := func(i int)string {
        if i < listLen - 1 {
            return prefix + "├───"
        }
        return prefix + "└───"
    }
    getSubPrefix := func(i int)string {
        if i < listLen  - 1 {
            return prefix + "│	"
        }
        return prefix + "	"
    }

    // Scan child files
    for i, child := range fileList {
        fileInfo, err := os.Lstat(filepath.Join(path, child))
        if err != nil {
            return fmt.Errorf("Cannot get file stat")
        }

        if fileInfo.IsDir() {
            fmt.Fprintf(out, "%v%v\n", getPrefix(i), fileInfo.Name())
            dirSubTree(out, filepath.Join(path, fileInfo.Name()), printFiles, getSubPrefix(i))
        } else {
            fmt.Fprintf(out, "%v%v (%v)\n", getPrefix(i), fileInfo.Name(), SizeFormat(fileInfo.Size()))
        }
    }
    return nil
}

func dirTree(out io.Writer, path string, printFiles bool) error {
    fileInfo, err := os.Lstat(path)
    if err != nil {
        return fmt.Errorf("Cannot open directory %v (%v)", path, err)
    }

    if !fileInfo.IsDir() {
        return fmt.Errorf("%v is not a directory", path)
    }

    return dirSubTree(out, path, printFiles, "")
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
