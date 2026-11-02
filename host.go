package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

type authListResponse struct {
	Files []pluginapi.HostAuthFileEntry `json:"files"`
}

type authListEntry = pluginapi.HostAuthFileEntry

// callHostRaw performs one host callback and decodes the RPC envelope.
func callHostRaw(method string, payload any) (json.RawMessage, error) {
	rawPayload, errMarshal := json.Marshal(payload)
	if errMarshal != nil {
		return nil, fmt.Errorf("marshal host callback payload %s: %w", method, errMarshal)
	}
	rawResponse, errRaw := hostCallRaw(method, rawPayload)
	if errRaw != nil && len(rawResponse) == 0 {
		return nil, errRaw
	}
	var env envelope
	if errUnmarshal := json.Unmarshal(rawResponse, &env); errUnmarshal != nil {
		return nil, fmt.Errorf("decode host callback envelope %s: %w", method, errUnmarshal)
	}
	if !env.OK {
		if env.Error != nil {
			return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		return nil, fmt.Errorf("host callback %s failed", method)
	}
	return append(json.RawMessage(nil), env.Result...), nil
}

// callHost bounds a host callback with a timeout. A callback that outlives the
// timeout keeps running in the host, so the result channel is buffered and the
// goroutine always releases the host buffer.
func callHost(method string, payload any, timeout time.Duration) (json.RawMessage, error) {
	if timeout <= 0 {
		return callHostRaw(method, payload)
	}
	type outcome struct {
		result json.RawMessage
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, errCall := callHostRaw(method, payload)
		done <- outcome{result: result, err: errCall}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case got := <-done:
		return got.result, got.err
	case <-timer.C:
		return nil, fmt.Errorf("host callback %s exceeded %s", method, timeout)
	}
}

func hostLog(level, message string, fields map[string]any) {
	payload := map[string]any{"level": level, "message": message}
	if len(fields) > 0 {
		payload["fields"] = fields
	}
	_, _ = callHost(pluginabi.MethodHostLog, payload, 5*time.Second)
}

func hostAuthList(timeout time.Duration) (authListResponse, error) {
	result, errCall := callHost(pluginabi.MethodHostAuthList, map[string]any{}, timeout)
	if errCall != nil {
		return authListResponse{}, errCall
	}
	var resp authListResponse
	if errUnmarshal := json.Unmarshal(result, &resp); errUnmarshal != nil {
		return authListResponse{}, fmt.Errorf("decode host.auth.list result: %w", errUnmarshal)
	}
	return resp, nil
}

func hostAuthGet(authIndex string, timeout time.Duration) (pluginapi.HostAuthGetResponse, error) {
	result, errCall := callHost(pluginabi.MethodHostAuthGet, pluginapi.HostAuthGetRequest{AuthIndex: authIndex}, timeout)
	if errCall != nil {
		return pluginapi.HostAuthGetResponse{}, errCall
	}
	var resp pluginapi.HostAuthGetResponse
	if errUnmarshal := json.Unmarshal(result, &resp); errUnmarshal != nil {
		return pluginapi.HostAuthGetResponse{}, fmt.Errorf("decode host.auth.get result: %w", errUnmarshal)
	}
	return resp, nil
}

func hostAuthGetRuntime(authIndex string, timeout time.Duration) (pluginapi.HostAuthGetRuntimeResponse, error) {
	result, errCall := callHost(pluginabi.MethodHostAuthGetRuntime, pluginapi.HostAuthGetRequest{AuthIndex: authIndex}, timeout)
	if errCall != nil {
		return pluginapi.HostAuthGetRuntimeResponse{}, errCall
	}
	var resp pluginapi.HostAuthGetRuntimeResponse
	if errUnmarshal := json.Unmarshal(result, &resp); errUnmarshal != nil {
		return pluginapi.HostAuthGetRuntimeResponse{}, fmt.Errorf("decode host.auth.get_runtime result: %w", errUnmarshal)
	}
	return resp, nil
}

func hostAuthSave(name string, raw json.RawMessage, timeout time.Duration) error {
	_, errCall := callHost(pluginabi.MethodHostAuthSave, pluginapi.HostAuthSaveRequest{Name: name, JSON: raw}, timeout)
	return errCall
}

// writeAuthFileAtomic replaces a credential file through a temporary file and a
// rename, so the auth directory watcher never observes a half written document.
// The temporary file deliberately avoids the .json suffix because the host
// treats every .json file in the auth directory as a credential.
func writeAuthFileAtomic(path string, payload []byte) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("credential has no backing file path")
	}
	dir := filepath.Dir(path)
	mode := os.FileMode(0o600)
	if info, errStat := os.Stat(path); errStat == nil {
		mode = info.Mode().Perm()
	}
	temp, errCreate := os.CreateTemp(dir, ".codex-quota-guard-*.tmp")
	if errCreate != nil {
		return fmt.Errorf("create temporary credential file: %w", errCreate)
	}
	tempName := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}
	if _, errWrite := temp.Write(payload); errWrite != nil {
		cleanup()
		return fmt.Errorf("write temporary credential file: %w", errWrite)
	}
	if errSync := temp.Sync(); errSync != nil {
		cleanup()
		return fmt.Errorf("sync temporary credential file: %w", errSync)
	}
	if errClose := temp.Close(); errClose != nil {
		_ = os.Remove(tempName)
		return fmt.Errorf("close temporary credential file: %w", errClose)
	}
	if errChmod := os.Chmod(tempName, mode); errChmod != nil {
		_ = os.Remove(tempName)
		return fmt.Errorf("chmod temporary credential file: %w", errChmod)
	}
	if errRename := os.Rename(tempName, path); errRename != nil {
		_ = os.Remove(tempName)
		return fmt.Errorf("replace credential file: %w", errRename)
	}
	return nil
}

func hostHTTPDo(request pluginapi.HTTPRequest, timeout time.Duration) (pluginapi.HTTPResponse, error) {
	result, errCall := callHost(pluginabi.MethodHostHTTPDo, request, timeout)
	if errCall != nil {
		return pluginapi.HTTPResponse{}, errCall
	}
	var resp pluginapi.HTTPResponse
	if errUnmarshal := json.Unmarshal(result, &resp); errUnmarshal != nil {
		return pluginapi.HTTPResponse{}, fmt.Errorf("decode host.http.do result: %w", errUnmarshal)
	}
	return resp, nil
}

// authDocument keeps every untouched credential field byte-for-byte intact so a
// guard-driven write never rewrites tokens or unrelated metadata.
type authDocument struct {
	fields map[string]json.RawMessage
}

func parseAuthDocument(raw []byte) (*authDocument, error) {
	fields := make(map[string]json.RawMessage)
	if errUnmarshal := json.Unmarshal(raw, &fields); errUnmarshal != nil {
		return nil, fmt.Errorf("decode auth document: %w", errUnmarshal)
	}
	return &authDocument{fields: fields}, nil
}

func (d *authDocument) stringValue(key string) string {
	raw, ok := d.fields[key]
	if !ok {
		return ""
	}
	var value string
	if errUnmarshal := json.Unmarshal(raw, &value); errUnmarshal != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func (d *authDocument) boolValue(key string) bool {
	raw, ok := d.fields[key]
	if !ok {
		return false
	}
	var value bool
	if errUnmarshal := json.Unmarshal(raw, &value); errUnmarshal != nil {
		return false
	}
	return value
}

func (d *authDocument) rawValue(key string) json.RawMessage {
	return d.fields[key]
}

func (d *authDocument) setBool(key string, value bool) {
	if value {
		d.fields[key] = json.RawMessage("true")
		return
	}
	d.fields[key] = json.RawMessage("false")
}

func (d *authDocument) setRaw(key string, value json.RawMessage) {
	if len(value) == 0 {
		delete(d.fields, key)
		return
	}
	d.fields[key] = value
}

func (d *authDocument) remove(key string) {
	delete(d.fields, key)
}

func (d *authDocument) marshal() (json.RawMessage, error) {
	raw, errMarshal := json.Marshal(d.fields)
	if errMarshal != nil {
		return nil, fmt.Errorf("encode auth document: %w", errMarshal)
	}
	return json.RawMessage(raw), nil
}
