package main

import (
	"flag"
	"os"
	"fmt"
	"io/ioutil"
	"github.com/dop251/goja"
	"log"
	"runtime/pprof"
	"time"
	"runtime/debug"
	crand "crypto/rand"
	"math/rand"
	"encoding/binary"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var timelimit = flag.Int("timelimit", 0, "max time to run (in seconds)")


func readSource(filename string) ([]byte, error) {
	if filename == "" || filename == "-" {
		return ioutil.ReadAll(os.Stdin)
	}
	return ioutil.ReadFile(filename)
}

func load(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	p := call.Argument(0).String()
	b, err := readSource(p)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Could not read %s: %v", p, err)))
	}
	v, err := vm.RunScript(p, string(b))
	if err != nil {
		panic(err)
	}
	return v
}

func console_log(call goja.FunctionCall) goja.Value {
	args := make([]interface{}, len(call.Arguments))
	for i, a := range call.Arguments {
		args[i] = a.String()
	}
	log.Print(args...)
	return nil
}

func createConsole(vm *goja.Runtime) *goja.Object {
	o := vm.NewObject()
	o.Set("log", console_log)
	return o
}

func newRandSource() goja.RandSource {
	var seed int64
	if err := binary.Read(crand.Reader, binary.LittleEndian, &seed); err != nil {
		panic(fmt.Errorf("Could not read random bytes: %v", err))
	}
	return rand.New(rand.NewSource(seed)).Float64
}

func run() error {
	filename := flag.Arg(0)
	src, err := readSource(filename)
	if err != nil {
		return err
	}

	if filename == "" || filename == "-" {
		filename = "<stdin>"
	}

	vm := goja.New()
	vm.SetRandSource(newRandSource())

	vm.Set("console", createConsole(vm))
	vm.Set("load", func(call goja.FunctionCall) goja.Value{
		return load(vm, call)
	})

	if *timelimit > 0 {
		time.AfterFunc(time.Duration(*timelimit) * time.Second, func() {
			vm.Interrupt("timeout")
		})
	}

	//log.Println("Compiling...")
	prg, err := goja.Compile(filename, string(src), false)
	if err != nil {
		return err
	}
	//log.Println("Running...")
	_, err = vm.RunProgram(prg)
	//log.Println("Finished.")
	return err
}

func main() {
	defer func() {
		if x := recover(); x != nil {
			debug.Stack()
			panic(x)
		}
	}()
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if err := run(); err != nil {
		//fmt.Printf("err type: %T\n", err)
		switch err := err.(type) {
		case *goja.Exception:
			fmt.Println(err.String())
		case *goja.InterruptedError:
			fmt.Println(err.String())
		default:
			fmt.Println(err)
		}
		os.Exit(64)
	}
}

