package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef int (*cliproxy_host_call_fn)(void*, const char*, const uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_host_free_fn)(void*, size_t);

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	cliproxy_host_call_fn call;
	cliproxy_host_free_fn free_buffer;
} cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cliproxyPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxyPluginFree(void*, size_t);
extern void cliproxyPluginShutdown(void);

static const cliproxy_host_api* stored_host;

static void store_host_api(const cliproxy_host_api* host) {
	stored_host = host;
}

static int call_host_api(const char* method, const uint8_t* request, size_t request_len, cliproxy_buffer* response) {
	if (stored_host == NULL || stored_host->call == NULL) {
		return 1;
	}
	return stored_host->call(stored_host->host_ctx, method, request, request_len, response);
}

static void free_host_buffer(void* ptr, size_t len) {
	if (stored_host != NULL && stored_host->free_buffer != NULL && ptr != NULL) {
		stored_host->free_buffer(ptr, len);
	}
}
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *envelopeError  `json:"error,omitempty"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type lifecycleRequest struct {
	ConfigYAML    []byte `json:"config_yaml"`
	SchemaVersion uint32 `json:"schema_version"`
}

type registration struct {
	SchemaVersion uint32                 `json:"schema_version"`
	Metadata      pluginapi.Metadata     `json:"metadata"`
	Capabilities  registrationCapability `json:"capabilities"`
}

type registrationCapability struct {
	ManagementAPI bool `json:"management_api"`
	UsagePlugin   bool `json:"usage_plugin"`
	Scheduler     bool `json:"scheduler"`
}

var guard = newGuardState()

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return 1
	}
	C.store_host_api(host)
	plugin.abi_version = C.uint32_t(pluginabi.ABIVersion)
	plugin.call = C.cliproxy_plugin_call_fn(C.cliproxyPluginCall)
	plugin.free_buffer = C.cliproxy_plugin_free_fn(C.cliproxyPluginFree)
	plugin.shutdown = C.cliproxy_plugin_shutdown_fn(C.cliproxyPluginShutdown)
	return 0
}

//export cliproxyPluginCall
func cliproxyPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response != nil {
		response.ptr = nil
		response.len = 0
	}
	if method == nil {
		writeResponse(response, errorEnvelope("invalid_method", "method is required"))
		return 1
	}
	var requestBytes []byte
	if request != nil && requestLen > 0 {
		requestBytes = C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
	}
	raw, errHandle := handleMethod(C.GoString(method), requestBytes)
	if errHandle != nil {
		writeResponse(response, errorEnvelope("plugin_error", errHandle.Error()))
		return 1
	}
	writeResponse(response, raw)
	return 0
}

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, _ C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() {
	guard.stop()
}

func handleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		if errConfigure := configure(request); errConfigure != nil {
			return nil, errConfigure
		}
		return okEnvelope(pluginRegistration())
	case pluginabi.MethodPluginQuiesce:
		guard.stop()
		return okEnvelope(map[string]any{})
	case pluginabi.MethodManagementRegister:
		return okEnvelope(managementRegistrationPayload())
	case pluginabi.MethodManagementHandle:
		return handleManagement(request)
	case pluginabi.MethodUsageHandle:
		var record pluginapi.UsageRecord
		if len(request) > 0 {
			if errUnmarshal := json.Unmarshal(request, &record); errUnmarshal != nil {
				return nil, fmt.Errorf("decode usage record: %w", errUnmarshal)
			}
		}
		guard.HandleUsage(context.Background(), record)
		return okEnvelope(map[string]any{})
	case pluginabi.MethodSchedulerPick:
		var pick pluginapi.SchedulerPickRequest
		if len(request) > 0 {
			if errUnmarshal := json.Unmarshal(request, &pick); errUnmarshal != nil {
				return nil, fmt.Errorf("decode scheduler request: %w", errUnmarshal)
			}
		}
		response, errPick := guard.Pick(context.Background(), pick)
		if errPick != nil {
			return nil, errPick
		}
		return okEnvelope(response)
	case pluginabi.MethodPluginShutdown:
		guard.stop()
		return okEnvelope(map[string]any{})
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func pluginRegistration() registration {
	return registration{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             pluginName,
			Version:          pluginVersion,
			Author:           pluginAuthor,
			GitHubRepository: repositoryURL,
			ConfigFields: []pluginapi.ConfigField{
				{Name: "disable_below_remaining_percent", Type: pluginapi.ConfigFieldTypeNumber, Description: "Disable a credential when the Codex 5h window remaining quota drops to this percentage or below."},
				{Name: "resume_above_remaining_percent", Type: pluginapi.ConfigFieldTypeNumber, Description: "Re-enable the credential once the 5h window remaining quota recovers to at least this percentage."},
				{Name: "enforce_weekly_window", Type: pluginapi.ConfigFieldTypeBoolean, Description: "Also honour the Codex 7d window when deciding whether a credential stays enabled."},
				{Name: "weekly_disable_below_remaining_percent", Type: pluginapi.ConfigFieldTypeNumber, Description: "Disable a credential when the Codex 7d window remaining quota drops to this percentage or below."},
				{Name: "weekly_resume_above_remaining_percent", Type: pluginapi.ConfigFieldTypeNumber, Description: "Re-enable the credential once the 7d window remaining quota recovers to at least this percentage."},
				{Name: "poll_interval", Type: pluginapi.ConfigFieldTypeString, Description: "How often the quota sweep runs, for example 60s."},
				{Name: "initial_delay", Type: pluginapi.ConfigFieldTypeString, Description: "Delay before the first sweep after the plugin starts, for example 15s."},
				{Name: "dry_run", Type: pluginapi.ConfigFieldTypeBoolean, Description: "Report the decisions without writing credential state."},
				{Name: "keep_at_least_one", Type: pluginapi.ConfigFieldTypeBoolean, Description: "Default false. When true, keep one provider credential available by protecting the one with the most remaining quota."},
				{Name: "rate_limit_fallback", Type: pluginapi.ConfigFieldTypeString, Description: "Fallback block duration when a Codex 429 omits reset headers."},
				{Name: "warmup_enabled", Type: pluginapi.ConfigFieldTypeBoolean, Description: "Enable scheduled Codex quota warm-up requests."},
				{Name: "warmup_model", Type: pluginapi.ConfigFieldTypeString, Description: "Model used for scheduled warm-up requests."},
				{Name: "warmup_prompt", Type: pluginapi.ConfigFieldTypeString, Description: "Prompt used for scheduled warm-up requests."},
				{Name: "warmup_timezone", Type: pluginapi.ConfigFieldTypeString, Description: "IANA timezone used by warm-up schedules."},
				{Name: "warmup_schedules", Type: pluginapi.ConfigFieldTypeArray, Description: "Daily warm-up schedule entries with id, at (HH:MM), and enabled."},
			},
		},
		Capabilities: registrationCapability{
			ManagementAPI: true,
			UsagePlugin:   true,
			Scheduler:     true,
		},
	}
}

// hostCallRaw performs one host callback and returns the raw RPC envelope bytes.
// The C bridge lives in this file so the statically linked helper shares the same
// translation unit as the stored host api pointer.
func hostCallRaw(method string, payload []byte) ([]byte, error) {
	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))

	var response C.cliproxy_buffer
	var requestPtr *C.uint8_t
	if len(payload) > 0 {
		cPayload := C.CBytes(payload)
		if cPayload == nil {
			return nil, fmt.Errorf("allocate host callback payload %s", method)
		}
		defer C.free(cPayload)
		requestPtr = (*C.uint8_t)(cPayload)
	}
	callCode := C.call_host_api(cMethod, requestPtr, C.size_t(len(payload)), &response)
	var rawResponse []byte
	if response.ptr != nil && response.len > 0 {
		rawResponse = C.GoBytes(response.ptr, C.int(response.len))
	}
	if response.ptr != nil {
		C.free_host_buffer(response.ptr, response.len)
	}
	if len(rawResponse) == 0 {
		return nil, fmt.Errorf("host callback %s returned no response, code=%d", method, int(callCode))
	}
	if callCode != 0 {
		return rawResponse, fmt.Errorf("host callback %s returned code=%d", method, int(callCode))
	}
	return rawResponse, nil
}

func okEnvelope(value any) ([]byte, error) {
	raw, errMarshal := json.Marshal(value)
	if errMarshal != nil {
		return nil, errMarshal
	}
	return json.Marshal(envelope{OK: true, Result: raw})
}

func errorEnvelope(code, message string) []byte {
	raw, errMarshal := json.Marshal(envelope{OK: false, Error: &envelopeError{Code: code, Message: message}})
	if errMarshal != nil {
		return []byte(`{"ok":false,"error":{"code":"plugin_error","message":"encode error"}}`)
	}
	return raw
}

func writeResponse(response *C.cliproxy_buffer, raw []byte) {
	if response == nil || len(raw) == 0 {
		return
	}
	ptr := C.CBytes(raw)
	if ptr == nil {
		return
	}
	response.ptr = ptr
	response.len = C.size_t(len(raw))
}
