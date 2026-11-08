//go:build !windows

package service

import "errors"

// These stubs are never invoked off Windows (useWindowsTask is false), but must
// exist so service.go compiles on every platform.

var errNotWindows = errors.New("windows scheduled task not available on this OS")

func installWindowsTask(Params) (string, error) { return "", errNotWindows }
func uninstallWindowsTask(Params) error         { return errNotWindows }
func controlWindowsTask(Params, string) error   { return errNotWindows }
func windowsTaskStatus(Params) (string, error)  { return "", errNotWindows }
