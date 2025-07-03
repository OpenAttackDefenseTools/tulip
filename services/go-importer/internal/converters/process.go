package converters

import (
	"fmt"
	"github.com/vmihailenco/msgpack/v5"
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
	Mutex sync.RWMutex

	Name    string
	Cmd     *exec.Cmd
	Encoder *msgpack.Encoder
	Decoder *msgpack.Decoder

	RestartMutex  sync.RWMutex
	Restarting    bool
	RestartWaiter chan bool
}

func (process *Process) createCmd() error {
	process.Cmd = exec.Command(GetPythonPath(), fmt.Sprintf("converters/%s.py", process.Name))

	stdin, err := process.Cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	process.Encoder = msgpack.NewEncoder(stdin)

	stdout, err := process.Cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	process.Decoder = msgpack.NewDecoder(stdout)

	// TODO: better handling around this?
	process.Cmd.Stderr = os.Stderr

	if err := process.Cmd.Start(); err != nil {
		return fmt.Errorf("failed to start converter for %s: %w", process.Name, err)
	}

	return nil
}

func (process *Process) Restart() error {
	process.RestartMutex.Lock()
	process.Restarting = true
	process.RestartMutex.Unlock()

	if err := process.Cmd.Process.Kill(); err != nil {
		return fmt.Errorf("killing converter failed: %w", err)
	}

	return nil
}

func NewProcess(converter string) (*Process, error) {
	process := &Process{
		Name: converter,

		Restarting:    false,
		RestartWaiter: make(chan bool, 1),
	}
	if err := process.createCmd(); err != nil {
		return nil, fmt.Errorf("failed to create converter: %w", err)
	}

	go func() {
		for {
			err := process.Cmd.Wait()

			// This aims to minimize the loss of conversions that happen. If it's a timeout restart, only
			// that specific conversion will be lost. Otherwise, there may be small delay where the converter
			// will be completely broken and leak more conversions (though it shouldn't really randomly crash...?)
			process.RestartMutex.Lock()
			process.Restarting = true
			process.RestartMutex.Unlock()

			log.Printf("WARN: Converter for %s died: %s\n", converter, err.Error())

			process.Mutex.Lock()
			for {
				if err := process.createCmd(); err != nil {
					log.Printf("!!! FAILED TO CREATE CONVERTER: %s\n", err.Error())
					time.Sleep(5 * time.Second)
					continue
				}

				break
			}
			process.Mutex.Unlock()

			process.RestartMutex.Lock()
			process.Restarting = false
			process.RestartMutex.Unlock()

			// This should never be anything else than zero, but don't hang things if it for some reason is
			if len(process.RestartWaiter) == 0 {
				process.RestartWaiter <- true
			}
		}
	}()

	return process, nil
}
