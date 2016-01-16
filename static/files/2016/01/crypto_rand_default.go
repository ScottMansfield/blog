package main

import "fmt"
import "crypto/rand"
//import "os"
//import "runtime/pprof"
import "sync"
import "time"

func randData(n int) []byte {
    b := make([]byte, n)
    rand.Read(b)

    for i := range b {
        b[i] = byte('A') + (b[i] % 26)
    }

    return b
}

func toInt(b [4]byte) int {
    i := int(b[0])
    i |= (int(b[1]) << 8 )
    i |= (int(b[2]) << 16)
    i |= (int(b[3]) << 24)

    return i
}

const numRuns = 1

func main() {
    /*f, err := os.Create("crypto_rand_default.prof")
    if err != nil {
        panic(err.Error())
    }

    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()*/

    start := time.Now()
    for i := 0; i < numRuns; i++ {
        do()
    }
    total := time.Since(start)
    perRun := float64(total) / numRuns

    fmt.Printf("Time per run: %fns\n", perRun)
}

const numRoutines = 10

func do() {
    start := make(chan struct{})
    comm := make(chan []byte)

    var read, write sync.WaitGroup
    read.Add(numRoutines)
    write.Add(numRoutines)

    for i := 0; i < numRoutines; i++ {
        go func() {
            var r [4]byte

            <-start
            for j := 1; j < 10000; j++ {
                _, err := rand.Read(r[:])
                if err != nil {
                    panic(err.Error())
                }
                comm <- randData(toInt(r) % 10000)
            }
            write.Done()
        }()

        go func() {
            var sum int
            <-start
            for c := range comm {
                sum += len(c)
            }
            fmt.Println(sum)
            read.Done()
        }()
    }

    close(start)
    write.Wait()
    close(comm)
    read.Wait()
}
