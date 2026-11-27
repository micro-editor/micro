package shell

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"

	shellquote "github.com/kballard/go-shellquote"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
)

// ExecCommand executes a command using exec
// It returns any output/errors
func ExecCommand(name string, arg ...string) (string, error) {
	var err error
	cmd := exec.Command(name, arg...)
	outputBytes := &bytes.Buffer{}
	cmd.Stdout = outputBytes
	cmd.Stderr = outputBytes
	err = cmd.Start()
	if err != nil {
		return "", err
	}
	err = cmd.Wait() // wait for command to finish
	outstring := outputBytes.String()
	return outstring, err
}

// RunCommand executes a shell command and returns the output/error
func RunCommand(input string) (string, error) {
	args, err := shellquote.Split(input)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		return "", errors.New("No arguments")
	}
	inputCmd := args[0]

	return ExecCommand(inputCmd, args[1:]...)
}

// RunBackgroundShell runs a shell command in the background
// It returns a function which will run the command and returns a string
// message result
func RunBackgroundShell(input string) (func() string, error) {
	args, err := shellquote.Split(input)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, errors.New("No arguments")
	}
	inputCmd := args[0]
	return func() string {
		output, err := RunCommand(input)

		str := output
		if err != nil {
			str = fmt.Sprint(inputCmd, " exited with error: ", err, ": ", output)
		}
		return str
	}, nil
}

// RunInteractiveShell logs that it is deprecated, but only once per session
var warnRunInteractiveShell sync.Once

// RunInteractiveShell runs a shellcommand interactively
//
// Deprecated: Use RunInteractiveCommand, which takes the arguments separately
// so that they don't need to be quoted.
func RunInteractiveShell(input string, wait bool, getOutput bool) (string, error) {
	warnRunInteractiveShell.Do(func() {
		log.Println("RunInteractiveShell is deprecated: use RunInteractiveCommand")
	})

	args, err := shellquote.Split(input)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		return "", errors.New("No arguments")
	}
	return RunInteractiveCommand(wait, getOutput, args[0], args[1:]...)
}

// RunInteractiveCommand temporarily closes micro and runs an executable with
// the given arguments in the terminal. If wait is true, it waits for the user
// to press enter before returning. If getOutput is true, it also returns what
// the command wrote to stdout.
func RunInteractiveCommand(wait bool, getOutput bool, name string, arg ...string) (string, error) {
	// Shut down the screen because we're going to interact directly with the shell
	screenb := screen.TempFini()

	// Set up everything for the command
	outputBytes := &bytes.Buffer{}
	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin
	if getOutput {
		cmd.Stdout = io.MultiWriter(os.Stdout, outputBytes)
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr

	var err error

	// This is a trap for Ctrl-C so that it doesn't kill micro
	// micro is killed if the signal is ignored only on Windows, so it is
	// received
	c := make(chan os.Signal, 1)
	signal.Reset(os.Interrupt)
	signal.Notify(c, os.Interrupt)
	err = cmd.Start()
	if err == nil {
		err = cmd.Wait()
		if wait {
			// This is just so we don't return right away and let the user press enter to return
			screen.TermMessage("")
		}
	} else {
		screen.TermMessage(err)
	}

	output := outputBytes.String()

	// Start the screen back up
	screen.TempStart(screenb)

	signal.Notify(util.Sigterm, os.Interrupt)
	signal.Stop(c)

	return output, err
}

// UserCommand runs the shell command
// The openTerm argument specifies whether a terminal should be opened (for viewing output
// or interacting with stdin)
// func UserCommand(input string, openTerm bool, waitToFinish bool) string {
// 	if !openTerm {
// 		RunBackgroundShell(input)
// 		return ""
// 	} else {
// 		output, _ := RunInteractiveShell(input, waitToFinish, false)
// 		return output
// 	}
// }
