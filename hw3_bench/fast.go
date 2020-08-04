package main

import (
	"io"
    "fmt"
    "os"
    "bufio"
    "strings"
)

// вам надо написать более быструю оптимальную этой функции
func readFile() chan []byte {
    ch := make(chan []byte, 1)
    go func(out chan []byte) {
        defer close(out)
        file, err := os.Open(filePath)
        if err != nil {
            panic(err)
        }
        defer file.Close()
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
            b :=make([]byte, len(scanner.Bytes()))
            copy(b, scanner.Bytes())
            out <- b
        }
    }(ch)
    return ch
}
func FastSearch(out io.Writer) {
    var user User
    var foundUsers string
    seenBrowsers := make(map[string]bool)
    in := readFile()
    i := 0
    for userRaw := range in {
        err := user.UnmarshalJSON(userRaw)
        if err != nil {
            panic(err)
        }
        isAndroid := false
        isMSIE := false

        for _, browser := range user.Browsers {
            if strings.Contains(browser, "Android") {
                isAndroid = true
                seenBrowsers[browser] = true
            }
            if strings.Contains(browser, "MSIE") {
                isMSIE = true
                seenBrowsers[browser] = true
            }
        }

        if isAndroid && isMSIE {
            email := strings.ReplaceAll(user.Email, "@", " [at] ")
            foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email)
        }
        i++
    }

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}

func FastSearch2(out io.Writer) {
    var user User
    var foundUsers string
    seenBrowsers := make(map[string]bool)
    file, err := os.Open(filePath)
    defer file.Close()
    if err != nil {
        panic(err)
    }
    scanner := bufio.NewScanner(file)
    i := 0
    for scanner.Scan() {
        userRaw := scanner.Bytes()
        err := user.UnmarshalJSON(userRaw)
        if err != nil {
            panic(err)
        }
        isAndroid := false
        isMSIE := false

        for _, browser := range user.Browsers {
            if strings.Contains(browser, "Android") {
                isAndroid = true
                seenBrowsers[browser] = true
            }
            if strings.Contains(browser, "MSIE") {
                isMSIE = true
                seenBrowsers[browser] = true
            }
        }

        if isAndroid && isMSIE {
            email := strings.ReplaceAll(user.Email, "@", " [at] ")
            foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email)
        }
        i++
    }

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
