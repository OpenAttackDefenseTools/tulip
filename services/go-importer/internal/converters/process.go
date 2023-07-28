package converters

import (
	"fmt"
	"github.com/vmihailenco/msgpack/v5"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

var pythonPath *string

func GetPythonPath() string {
	if pythonPath == nil {
		path, err := exec.LookPath("python3")
		if err != nil {
			panic(fmt.Errorf("failed to find python3: %w", err))
		}

		pythonPath = &path
	}

	return *pythonPath
}

type Process struct {
	Name string

	Mutex   sync.RWMutex
	Cmd     *exec.Cmd
	Encoder *msgpack.Encoder
	Decoder *msgpack.Decoder
}

func createCmd(converter string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
	cmd := exec.Command(GetPythonPath(), fmt.Sprintf("converters/%s.py", converter))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// TODO: better handling around this?
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start converter for %s: %w", converter, err)
	}

	return cmd, stdin, stdout, nil
}

func NewProcess(converter string) (*Process, error) {
	cmd, stdin, stdout, err := createCmd(converter)
	if err != nil {
		return nil, fmt.Errorf("failed to create converter: %w", err)
	}

	process := &Process{
		Name: converter,

		Cmd:     cmd,
		Encoder: msgpack.NewEncoder(stdin),
		Decoder: msgpack.NewDecoder(stdout),
	}

	go func() {
		for {
			err := process.Cmd.Wait()
			log.Printf("WARN: Converter for %s died: %s\n", converter, err.Error())

			// TODO: this is not actually enough as access is contested between flows too (some channel magic?)
			process.Mutex.Lock()
			for {
				cmd, stdin, stdout, err := createCmd(converter)
				if err != nil {
					log.Printf("!!! FAILED TO CREATE CONVERTER: %s\n", err.Error())
					time.Sleep(5 * time.Second)
					continue
				}

				// Go doesn't allow recovering the original cmd, so...
				process.Cmd = cmd
				process.Encoder = msgpack.NewEncoder(stdin)
				process.Decoder = msgpack.NewDecoder(stdout)

				break
			}
			process.Mutex.Unlock()
		}
	}()

	return process, nil
}
