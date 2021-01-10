package main

import (
//    "fmt"
    "sort"
    "sync"
    "strings"
    "strconv"
    "runtime"
)

const (
    Md5Limit            = 1
    MultiThreadCount    = 6
    PipeLimit           = 100
)

func wrapperSignerMd5(chMd5 chan struct{}, data string)string {
    chMd5 <- struct{}{}
    defer func() {
        <-chMd5
    }()
    return DataSignerMd5(data)
}

func wrapperSignerCrc32(data string)chan string {
    resultCh := make(chan string, 1)
    go func(out chan string, data string) {
        out <- DataSignerCrc32(data)
    }(resultCh, data)
    return resultCh
}

func ExecutePipeline(jobs ...job) {
    out := make(chan interface{}, PipeLimit)
    wg := &sync.WaitGroup{}
    for _, jf := range jobs {
        in := out
        out = make(chan interface{}, PipeLimit)
        wg.Add(1)
        go func(wg *sync.WaitGroup, jobFunc job, in, out chan interface{}) {
            defer wg.Done()
            defer close(out)
            jobFunc(in, out)
        }(wg, jf, in, out)
    }
    wg.Wait()
}

func SingleHash(in, out chan interface{}) {
    chMd5 := make(chan struct{}, Md5Limit)
    wg := &sync.WaitGroup{}
    for dataRow := range in {
        d, ok := dataRow.(int)
        if !ok {
            panic("Cannot covert data to int")
        }
        //fmt.Println("SingleHash data", d)
        data := strconv.Itoa(d)
        wg.Add(1)

        go func(wg *sync.WaitGroup, out chan interface{}, data string, chMd5 chan struct{}) {
            defer wg.Done()
            ch1 := wrapperSignerCrc32(data)
            ch2 := wrapperSignerCrc32(wrapperSignerMd5(chMd5, data))
            out <- (<-ch1 + "~" + <-ch2)
        }(wg, out, data, chMd5)
        runtime.Gosched()
    }
    wg.Wait()
}

func MultiHash(in, out chan interface{}) {
    wg := &sync.WaitGroup{}
    for dataRow := range in {
        data, ok := dataRow.(string)
        if !ok {
            panic("Cannot covert data to string")
        }
        //fmt.Println("MultiHash data", data)
        wg.Add(1)
        go func(wg *sync.WaitGroup, out chan interface{}, data string) {
            defer wg.Done()
            //parts := make([]chan string, MultiThreadCount)
            var parts [MultiThreadCount]chan string
            for i := 0; i < MultiThreadCount; i++ {
                th := strconv.Itoa(i)
                parts[i] = wrapperSignerCrc32(th + data)
            }
            var result string
            for i := 0; i < MultiThreadCount; i++ {
                result += <-parts[i]
            }
            out <- result
        }(wg, out, data)
        runtime.Gosched()
    }
    wg.Wait()
}

func CombineResults(in, out chan interface{}) {
    result := make([]string, 0)
    for dataRow := range in {
        data, ok := dataRow.(string)
        if !ok {
            panic("Cannot convert data to string")
        }
        result = append(result, data)
    }
    sort.Strings(result)
    out <- strings.Join(result, "_")
}
