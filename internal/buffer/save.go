package buffer

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"
	"unicode"

	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
	"golang.org/x/text/transform"
)

// LargeFileThreshold is the number of bytes when fastdirty is forced
// because hashing is too slow
const LargeFileThreshold = 50000

type wrappedFile struct {
	name        string
	writeCloser io.WriteCloser
	withSudo    bool
	screenb     bool
	cmd         *exec.Cmd
	sigChan     chan os.Signal
}

type saveResponse struct {
	size int
	err  error
}

type saveRequest struct {
	buf              *Buffer
	path             string
	withSudo         bool
	newFile          bool
	saveResponseChan chan saveResponse
}

var saveRequestChan chan saveRequest
var backupRequestChan chan backupRequest

func init() {
	// Both saveRequestChan and backupRequestChan need to be non-buffered
	// so the save/backup goroutine receives both save and backup requests
	// in the same order the main goroutine sends them.
	saveRequestChan = make(chan saveRequest)
	backupRequestChan = make(chan backupRequest)

	go func() {
		duration := backupSeconds * float64(time.Second)
		backupTicker := time.NewTicker(time.Duration(duration))

		for {
			select {
			case sr := <-saveRequestChan:
				size, err := sr.buf.safeWrite(sr.path, sr.withSudo, sr.newFile)
				sr.saveResponseChan <- saveResponse{size, err}
			case br := <-backupRequestChan:
				handleBackupRequest(br)
			case <-backupTicker.C:
				periodicBackup()
			}
		}
	}()
}

func openFile(name string, withSudo bool) (wrappedFile, error) {
	var err error
	var writeCloser io.WriteCloser
	var screenb bool
	var cmd *exec.Cmd
	var sigChan chan os.Signal

	if withSudo {
		conv := "notrunc"
		// TODO: both platforms do not support dd with conv=fsync yet
		if !(runtime.GOOS == "illumos" || runtime.GOOS == "netbsd") {
			conv += ",fsync"
		}

		cmd = exec.Command(config.GlobalSettings["sucmd"].(string), "dd", "bs=4k", "conv="+conv, "of="+name)
		writeCloser, err = cmd.StdinPipe()
		if err != nil {
			return wrappedFile{}, err
		}

		sigChan = make(chan os.Signal, 1)
		signal.Reset(os.Interrupt)
		signal.Notify(sigChan, os.Interrupt)

		screenb = screen.TempFini()
		// need to start the process now, otherwise when we flush the file
		// contents to its stdin it might hang because the kernel's pipe size
		// is too small to handle the full file contents all at once
		err = cmd.Start()
		if err != nil {
			screen.TempStart(screenb)

			signal.Notify(util.Sigterm, os.Interrupt)
			signal.Stop(sigChan)

			return wrappedFile{}, err
		}
	} else {
		writeCloser, err = os.OpenFile(name, os.O_WRONLY|os.O_CREATE, util.FileMode)
		if err != nil {
			return wrappedFile{}, err
		}
	}

	return wrappedFile{name, writeCloser, withSudo, screenb, cmd, sigChan}, nil
}

func (wf wrappedFile) Truncate() error {
	if wf.withSudo {
		// we don't need to stop the screen here, since it is still stopped
		// by openFile()
		// truncate might not be available on every platfom, so use dd instead
		cmd := exec.Command(config.GlobalSettings["sucmd"].(string), "dd", "count=0", "of="+wf.name)
		return cmd.Run()
	}
	return wf.writeCloser.(*os.File).Truncate(0)
}

func (wf wrappedFile) Write(b *SharedBuffer) (int, error) {
	file := bufio.NewWriter(transform.NewWriter(wf.writeCloser, b.encoding.NewEncoder()))

	b.Lock()
	defer b.Unlock()

	if len(b.lines) == 0 {
		return 0, nil
	}

	// end of line
	var eol []byte
	if b.Endings == FFDos {
		eol = []byte{'\r', '\n'}
	} else {
		eol = []byte{'\n'}
	}

	err := wf.Truncate()
	if err != nil {
		return 0, err
	}

	// write lines
	size, err := file.Write(b.lines[0].data)
	if err != nil {
		return 0, err
	}

	for _, l := range b.lines[1:] {
		if _, err = file.Write(eol); err != nil {
			return 0, err
		}
		if _, err = file.Write(l.data); err != nil {
			return 0, err
		}
		size += len(eol) + len(l.data)
	}

	err = file.Flush()
	if err == nil && !wf.withSudo {
		// Call Sync() on the file to make sure the content is safely on disk.
		f := wf.writeCloser.(*os.File)
		err = f.Sync()
	}
	return size, err
}

func (wf wrappedFile) Close() error {
	err := wf.writeCloser.Close()
	if wf.withSudo {
		// wait for dd to finish and restart the screen if we used sudo
		err := wf.cmd.Wait()
		screen.TempStart(wf.screenb)

		signal.Notify(util.Sigterm, os.Interrupt)
		signal.Stop(wf.sigChan)

		if err != nil {
			return err
		}
	}
	return err
}

func (b *SharedBuffer) overwriteFile(name string) (int, error) {
	file, err := openFile(name, false)
	if err != nil {
		return 0, err
	}

	size, err := file.Write(b)

	err2 := file.Close()
	if err2 != nil && err == nil {
		err = err2
	}
	return size, err
}

// Save saves the buffer to its default path
func (b *Buffer) Save() error {
	return b.SaveAs(b.Path)
}

// AutoSave saves the buffer to its default path
func (b *Buffer) AutoSave() error {
	if !b.Modified() {
		return nil
	}
	return b.saveToFile(b.Path, false, true)
}

// SaveAs saves the buffer to a specified path (filename), creating the file if it does not exist
func (b *Buffer) SaveAs(filename string) error {
	return b.saveToFile(filename, false, false)
}

func (b *Buffer) SaveWithSudo() error {
	return b.SaveAsWithSudo(b.Path)
}

func (b *Buffer) SaveAsWithSudo(filename string) error {
	return b.saveToFile(filename, true, false)
}

func (b *Buffer) saveToFile(filename string, withSudo bool, autoSave bool) error {
	var err error
	if b.Type.Readonly {
		return errors.New("Cannot save readonly buffer")
	}
	if b.Type.Scratch {
		return errors.New("Cannot save scratch buffer")
	}

	if !autoSave && b.Settings["rmtrailingws"].(bool) {
		for i, l := range b.lines {
			leftover := util.CharacterCount(bytes.TrimRightFunc(l.data, unicode.IsSpace))

			linelen := util.CharacterCount(l.data)
			b.Remove(Loc{leftover, i}, Loc{linelen, i})
		}

		b.RelocateCursors()
	}

	if b.Settings["eofnewline"].(bool) {
		end := b.End()
		if b.RuneAt(Loc{end.X - 1, end.Y}) != '\n' {
			b.insert(end, []byte{'\n'})
		}
	}

	filename, err = util.ReplaceHome(filename)
	if err != nil {
		return err
	}

	newFile := false
	fileInfo, err := os.Stat(filename)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		newFile = true
	}
	if err == nil && fileInfo.IsDir() {
		return errors.New("Error: " + filename + " is a directory and cannot be saved")
	}
	if err == nil && !fileInfo.Mode().IsRegular() {
		return errors.New("Error: " + filename + " is not a regular file and cannot be saved")
	}

	// Ignore the returned error, since the checks are already performed above
	filename, _ = filepath.EvalSymlinks(filename)
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return err
	}

	// Get the leading path to the file | "." is returned if there's no leading path provided
	if dirname := filepath.Dir(absFilename); dirname != "." {
		// Check if the parent dirs don't exist
		if _, statErr := os.Stat(dirname); errors.Is(statErr, fs.ErrNotExist) {
			// Prompt to make sure they want to create the dirs that are missing
			if b.Settings["mkparents"].(bool) {
				// Create all leading dir(s) since they don't exist
				if mkdirallErr := os.MkdirAll(dirname, os.ModePerm); mkdirallErr != nil {
					// If there was an error creating the dirs
					return mkdirallErr
				}
			} else {
				return errors.New("Parent dirs don't exist, enable 'mkparents' for auto creation")
			}
		}
	}

	saveResponseChan := make(chan saveResponse)
	saveRequestChan <- saveRequest{b, absFilename, withSudo, newFile, saveResponseChan}
	result := <-saveResponseChan
	err = result.err
	if err != nil {
		if errors.Is(err, util.ErrOverwrite) {
			screen.TermMessage(err)
			err = errors.Unwrap(err)

			b.UpdateModTime()
		}
		return err
	}

	if !b.Settings["fastdirty"].(bool) {
		if result.size > LargeFileThreshold {
			// For large files 'fastdirty' needs to be on
			b.Settings["fastdirty"] = true
		} else {
			b.calcHash(&b.origHash)
		}
	}

	newPath := b.Path != filename
	if newPath {
		b.RemoveBackup()
	}

	b.Path = filename
	b.AbsPath = absFilename
	b.isModified = false
	b.UpdateModTime()

	if newPath {
		// need to update glob-based and filetype-based settings
		b.ReloadSettings(true)
	}

	err = b.Serialize()
	return err
}

// safeWrite writes the buffer to a file in a "safe" way, preventing loss of the
// contents of the file if it fails to write the new contents.
// This means that the file is not overwritten directly but by writing to the
// backup file first.
func (b *SharedBuffer) safeWrite(path string, withSudo bool, newFile bool) (int, error) {
	file, err := openFile(path, withSudo)
	if err != nil {
		return 0, err
	}

	defer func() {
		if newFile && err != nil {
			os.Remove(path)
		}
	}()

	// Try to backup first before writing
	backupName, resolveName, err := b.writeBackup(path)
	if err != nil {
		file.Close()
		return 0, err
	}

	// Backup saved, so cancel pending periodic backup, if any
	delete(requestedBackups, b)

	b.forceKeepBackup = true
	size := 0
	{
		// If we failed to write or close, keep the backup we made
		size, err = file.Write(b)
		if err != nil {
			file.Close()
			return 0, util.OverwriteError{err, backupName}
		}

		err = file.Close()
		if err != nil {
			return 0, util.OverwriteError{err, backupName}
		}
	}
	b.forceKeepBackup = false

	if !b.keepBackup() {
		b.removeBackup(backupName, resolveName)
	}

	return size, err
}
