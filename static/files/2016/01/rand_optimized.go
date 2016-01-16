package main

import "fmt"
import "math/rand"
import "os"
import "runtime/pprof"
import "sync"
import "time"

var letters = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
func randData(r *rand.Rand, n int) []byte {
    b := make([]byte, n)

    for i := range b {
        b[i] = letters[r.Intn(len(letters))]
    }

    return b
}

const numRuns = 100

func main() {
    f, err := os.Create("rand_optimized.prof")
    if err != nil {
        panic(err.Error())
    }

    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()

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
            r := rand.New(rand.NewSource(time.Now().Unix()))
            <-start
            for j := 1; j < 10000; j++ {
                comm <- randData(r, r.Intn(j))
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
