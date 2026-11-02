//go:build !linux

package main

import "os"

func lockStateFile(_ *os.File) error { return nil }

func unlockStateFile(_ *os.File) {}
