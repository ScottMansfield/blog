package main

import "bufio"
import "fmt"
import crand "crypto/rand"
import mrand "math/rand"
import "time"

type MRandReader struct {
    src mrand.Source
}
func (r MRandReader) Read(p []byte) (n int, err error) {
    for n < len(p) {
        x := r.src.Int63()
        m := n + 7

        if m > len(p) {
            m = len(p)
        }

        for ; n < m; n++ {
            p[n] = byte(x)
            x >>= 8
        }
    }
    return n, nil
}

type CRandReader struct { }
func (c CRandReader) Read(p []byte) (n int, err error) {
    return crand.Read(p)
}

func main() {
    p := make([]byte, 32)
    mr := bufio.NewReader(MRandReader{src: mrand.NewSource(42)})
    cr := bufio.NewReader(CRandReader{})

    dataCounter := int64(0)
    end := time.After(10 * time.Second)

    outer1: for {
        select {
            case <-end:
                break outer1
            default:
                n, _ := mr.Read(p)
                dataCounter += int64(n)
        }
    }

    fmt.Printf("%v bytes read in 10 seconds from math/rand\n", dataCounter)
    fmt.Printf("%v MB/s\n", float64(dataCounter) / (1024.0 * 1024.0 * 10))

    dataCounter = 0
    end = time.After(10 * time.Second)

    outer2: for {
        select {
            case <-end:
                break outer2
            default:
                n, _ := cr.Read(p)
                dataCounter += int64(n)
        }
    }

    fmt.Printf("%v bytes read in 10 seconds from crypto/rand\n", dataCounter)
    fmt.Printf("%v MB/s\n", float64(dataCounter) / (1024.0 * 1024.0 * 10))
}
