package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const stateDirectoryMode = 0o700
const stateFileMode = 0o600

type stateFileStore struct {
	path string
}

func newStateFileStore(path string) *stateFileStore {
	return &stateFileStore{path: strings.TrimSpace(path)}
}

func defaultPersistentStateFile() string {
	if info, errStat := os.Stat("/CLIProxyAPI/plugins"); errStat == nil && info.IsDir() {
		return "/CLIProxyAPI/plugins/state/codex-quota-guard/state.json"
	}
	dir, errConfig := os.UserConfigDir()
	if errConfig != nil || strings.TrimSpace(dir) == "" {
		dir = "."
	}
	return filepath.Join(dir, "CLIProxyAPI", pluginID, "state.json")
}

func (s *stateFileStore) load() (persistentState, error) {
	if err := validateStatePath(s.path); err != nil {
		return newPersistentState(), err
	}
	var state persistentState
	errRead := s.withLock(func() error {
		raw, errRead := os.ReadFile(s.path)
		if errors.Is(errRead, os.ErrNotExist) {
			state = newPersistentState()
			return nil
		}
		if errRead != nil {
			return fmt.Errorf("read persistent state: %w", errRead)
		}
		if errDecode := json.Unmarshal(raw, &state); errDecode != nil {
			return fmt.Errorf("decode persistent state: %w", errDecode)
		}
		return nil
	})
	if errRead != nil {
		return newPersistentState(), errRead
	}
	if errValidate := validatePersistentState(&state); errValidate != nil {
		return newPersistentState(), errValidate
	}
	return state, nil
}

func (s *stateFileStore) save(state persistentState) error {
	if err := validateStatePath(s.path); err != nil {
		return err
	}
	if errValidate := validatePersistentState(&state); errValidate != nil {
		return errValidate
	}
	state.UpdatedAt = time.Now().UTC()
	raw, errMarshal := json.MarshalIndent(state, "", "  ")
	if errMarshal != nil {
		return fmt.Errorf("encode persistent state: %w", errMarshal)
	}
	raw = append(raw, '\n')
	return s.withLock(func() error {
		dir := filepath.Dir(s.path)
		if errMkdir := os.MkdirAll(dir, stateDirectoryMode); errMkdir != nil {
			return fmt.Errorf("create persistent state directory: %w", errMkdir)
		}
		file, errCreate := os.CreateTemp(dir, ".codex-quota-guard-state-*.tmp")
		if errCreate != nil {
			return fmt.Errorf("create temporary persistent state: %w", errCreate)
		}
		tempPath := file.Name()
		cleanup := func() {
			_ = file.Close()
			_ = os.Remove(tempPath)
		}
		if errChmod := file.Chmod(stateFileMode); errChmod != nil {
			cleanup()
			return fmt.Errorf("chmod temporary persistent state: %w", errChmod)
		}
		if _, errWrite := file.Write(raw); errWrite != nil {
			cleanup()
			return fmt.Errorf("write temporary persistent state: %w", errWrite)
		}
		if errSync := file.Sync(); errSync != nil {
			cleanup()
			return fmt.Errorf("sync temporary persistent state: %w", errSync)
		}
		if errClose := file.Close(); errClose != nil {
			_ = os.Remove(tempPath)
			return fmt.Errorf("close temporary persistent state: %w", errClose)
		}
		if errRename := os.Rename(tempPath, s.path); errRename != nil {
			_ = os.Remove(tempPath)
			return fmt.Errorf("replace persistent state: %w", errRename)
		}
		if dirFile, errOpen := os.Open(dir); errOpen == nil {
			_ = dirFile.Sync()
			_ = dirFile.Close()
		}
		return nil
	})
}

func validateStatePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("state_file is empty")
	}
	clean := filepath.Clean(path)
	if clean == "." || clean == string(filepath.Separator) {
		return fmt.Errorf("state_file must name a file: %q", path)
	}
	return nil
}

func (s *stateFileStore) withLock(fn func() error) error {
	dir := filepath.Dir(s.path)
	if errMkdir := os.MkdirAll(dir, stateDirectoryMode); errMkdir != nil {
		return fmt.Errorf("create persistent state directory: %w", errMkdir)
	}
	lockPath := s.path + ".lock"
	lockFile, errOpen := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, stateFileMode)
	if errOpen != nil {
		return fmt.Errorf("open persistent state lock: %w", errOpen)
	}
	defer lockFile.Close()
	if errLock := lockStateFile(lockFile); errLock != nil {
		return fmt.Errorf("lock persistent state: %w", errLock)
	}
	defer unlockStateFile(lockFile)
	return fn()
}
