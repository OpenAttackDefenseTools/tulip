package converters

import (
	"fmt"
	"github.com/vmihailenco/msgpack/v5"
	"os"
	"os/exec"
	"sync"
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

func NewProcess(converter string) (*Process, error) {
	cmd := exec.Command(GetPythonPath(), fmt.Sprintf("converters/%s.py", converter))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// TODO: better handling around this?
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start converter for %s: %w", converter, err)
	}

	// TODO: the process may (somehow?) die - we need some kind of recovery mechanism

	return &Process{
		Name: converter,

		Cmd:     cmd,
		Encoder: msgpack.NewEncoder(stdin),
		Decoder: msgpack.NewDecoder(stdout),
	}, nil
}
