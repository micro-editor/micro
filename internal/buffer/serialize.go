package buffer

import (
	"bytes"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/util"
)

// The SerializedBuffer holds the types that get serialized when a buffer is saved
// These are used for the savecursor and saveundo options
type SerializedBuffer struct {
	EventHandler *EventHandler
	Cursor       Loc
	ModTime      time.Time
}

// Serialize serializes the buffer to config.ConfigDir/buffers
func (b *Buffer) Serialize() error {
	if !b.Settings["savecursor"].(bool) && !b.Settings["saveundo"].(bool) {
		return nil
	}
	if b.Path == "" {
		return nil
	}

	buffer := SerializedBuffer{
		Cursor:  b.GetActiveCursor().Loc,
		ModTime: b.ModTime,
	}
	if b.Settings["saveundo"].(bool) {
		buffer.EventHandler = b.EventHandler
	}

	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(buffer)
	if err != nil {
		return err
	}

	name, resolveName := util.DetermineEscapePath(filepath.Join(config.ConfigDir, "buffers"), b.AbsPath)
	err = util.SafeWrite(name, buf.Bytes(), true)
	if err != nil {
		return err
	}

	if resolveName != "" {
		err = util.SafeWrite(resolveName, []byte(b.AbsPath), true)
		if err != nil {
			return err
		}
	}

	return nil
}

// Unserialize loads the buffer info from config.ConfigDir/buffers
func (b *Buffer) Unserialize() error {
	// If either savecursor or saveundo is turned on, we need to load the serialized information
	// from ~/.config/micro/buffers
	if b.Path == "" {
		return nil
	}
	name, _ := util.DetermineEscapePath(filepath.Join(config.ConfigDir, "buffers"), b.AbsPath)
	file, err := os.Open(name)
	if err == nil {
		defer file.Close()
		var buffer SerializedBuffer
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&buffer)
		if err != nil {
			return errors.New(err.Error() + "\nYou may want to remove the files in ~/.config/micro/buffers (these files\nstore the information for the 'saveundo' and 'savecursor' options) if\nthis problem persists.\nThis may be caused by upgrading to version 2.0, and removing the 'buffers'\ndirectory will reset the cursor and undo history and solve the problem.")
		}
		if b.Settings["savecursor"].(bool) {
			b.StartCursor = buffer.Cursor
		}

		if b.Settings["saveundo"].(bool) && buffer.EventHandler != nil {
			// We should only use last time's eventhandler if the file wasn't modified by someone else in the meantime
			if b.ModTime == buffer.ModTime {
				b.EventHandler = buffer.EventHandler
				b.EventHandler.cursors = b.cursors
				b.EventHandler.buf = b.SharedBuffer
			}
		}
	}
	return nil
}
