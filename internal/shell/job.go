package shell

import (
	"bytes"
	"io"
	"os/exec"
	"runtime"
)

var Jobs chan JobFunction

func init() {
	Jobs = make(chan JobFunction, 100)
}

// Jobs are the way plugins can run processes in the background
// A job is simply a process that gets executed asynchronously
// There are callbacks for when the job exits, when the job creates stdout
// and when the job creates stderr

// These jobs run in a separate goroutine but the lua callbacks need to be
// executed in the main thread (where the Lua VM is running) so they are
// put into the jobs channel which gets read by the main loop

// JobFunction is a representation of a job (this data structure is what is loaded
// into the jobs channel)
type JobFunction struct {
	Function func(string, []any)
	Output   string
	Args     []any
}

// A CallbackFile is the data structure that makes it possible to catch stderr and stdout write events
type CallbackFile struct {
	io.Writer

	callback func(string, []any)
	args     []any
}

// Job stores the executing command for the job, and the stdin pipe
type Job struct {
	*exec.Cmd
	Stdin io.WriteCloser
}

func (f *CallbackFile) Write(data []byte) (int, error) {
	// This is either stderr or stdout
	// In either case we create a new job function callback and put it in the jobs channel
	jobFunc := JobFunction{f.callback, string(data), f.args}
	Jobs <- jobFunc
	return f.Writer.Write(data)
}

// JobStart starts a shell command in the background with the given callbacks
// It returns an *exec.Cmd as the job id
func JobStart(cmd string, onStdout, onStderr, onExit func(string, []any), userargs ...any) *Job {
	if runtime.GOOS == "windows" {
		return JobSpawn("cmd", []string{"/v:on", "/c", cmd}, onStdout, onStderr, onExit, userargs...)
	} else {
		return JobSpawn("sh", []string{"-c", cmd}, onStdout, onStderr, onExit, userargs...)
	}
}

// JobSpawn starts a process with args in the background with the given callbacks
// It returns an *exec.Cmd as the job id
func JobSpawn(cmdName string, cmdArgs []string, onStdout, onStderr, onExit func(string, []any), userargs ...any) *Job {
	// Set up everything correctly if the functions have been provided
	proc := exec.Command(cmdName, cmdArgs...)
	var outbuf bytes.Buffer
	if onStdout != nil {
		proc.Stdout = &CallbackFile{&outbuf, onStdout, userargs}
	} else {
		proc.Stdout = &outbuf
	}
	if onStderr != nil {
		proc.Stderr = &CallbackFile{&outbuf, onStderr, userargs}
	} else {
		proc.Stderr = &outbuf
	}
	stdin, _ := proc.StdinPipe()

	go func() {
		// Run the process in the background and create the onExit callback
		proc.Run()
		if onExit != nil {
			jobFunc := JobFunction{onExit, outbuf.String(), userargs}
			Jobs <- jobFunc
		}
	}()

	return &Job{proc, stdin}
}

// JobStop kills a job
func JobStop(j *Job) {
	j.Process.Kill()
}

// JobSend sends the given data into the job's stdin stream
func JobSend(j *Job, data string) {
	j.Stdin.Write([]byte(data))
}
