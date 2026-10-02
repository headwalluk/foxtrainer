//go:build e2e

package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// marionettePort avoids clashing with any Firefox on the default 2828.
const marionettePort = 28296

// marionette is a minimal client for Firefox's remote protocol: length-prefixed JSON packets.
type marionette struct {
	connection net.Conn
	reader     *bufio.Reader
	messageID  int
}

// withMarionette starts Firefox with Marionette on the e2e profile, runs probe in chrome context, then quits.
func (current *harness) withMarionette(test *testing.T, probe func(client *marionette)) {
	test.Helper()

	profileDir := current.profileDir(test)
	prefsLine := fmt.Sprintf("user_pref(\"marionette.port\", %d);\n", marionettePort)

	prefsFile, openError := os.OpenFile(filepath.Join(profileDir, "prefs.js"), os.O_APPEND|os.O_WRONLY, 0o600)
	if openError != nil {
		test.Fatal(openError)
	}

	_, writeError := prefsFile.WriteString(prefsLine)
	if closeError := prefsFile.Close(); writeError != nil || closeError != nil {
		test.Fatalf("set marionette port: %v %v", writeError, closeError)
	}

	ctx, cancel := context.WithTimeout(test.Context(), 60*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, current.firefox, "--headless", "--marionette", "-remote-allow-system-access", "-no-remote", "-P", "e2e")
	command.Env = hermeticEnvironment(current.homeDir)
	command.Cancel = func() error { return command.Process.Signal(syscall.SIGTERM) }
	command.WaitDelay = 20 * time.Second

	if startError := command.Start(); startError != nil {
		test.Fatal(startError)
	}

	client := connectMarionette(test, ctx)
	client.command(test, "WebDriver:NewSession", map[string]any{})
	client.command(test, "Marionette:SetContext", map[string]any{"value": "chrome"})

	probe(client)

	client.command(test, "Marionette:Quit", map[string]any{})

	if waitError := command.Wait(); waitError != nil {
		test.Logf("firefox exit: %v", waitError)
	}
}

// connectMarionette retries until Firefox is listening, then reads the server greeting.
func connectMarionette(test *testing.T, ctx context.Context) *marionette {
	test.Helper()

	var dialer net.Dialer

	var client *marionette

	for client == nil && ctx.Err() == nil {
		connection, dialError := dialer.DialContext(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(marionettePort))
		if dialError != nil {
			time.Sleep(500 * time.Millisecond)

			continue
		}

		client = &marionette{connection: connection, reader: bufio.NewReader(connection)}
	}

	if client == nil {
		test.Fatal("Marionette did not start")
	}

	client.readPacket(test)

	return client
}

// readPacket reads one "length:json" packet.
func (client *marionette) readPacket(test *testing.T) json.RawMessage {
	test.Helper()

	lengthText, readError := client.reader.ReadString(':')
	if readError != nil {
		test.Fatal(readError)
	}

	length, convertError := strconv.Atoi(lengthText[:len(lengthText)-1])
	if convertError != nil {
		test.Fatal(convertError)
	}

	body := make([]byte, length)
	if _, fullError := io.ReadFull(client.reader, body); fullError != nil {
		test.Fatal(fullError)
	}

	return body
}

// command sends one command and returns its result, failing the test on a protocol error.
func (client *marionette) command(test *testing.T, name string, parameters map[string]any) json.RawMessage {
	test.Helper()

	client.messageID++

	payload, marshalError := json.Marshal([]any{0, client.messageID, name, parameters})
	if marshalError != nil {
		test.Fatal(marshalError)
	}

	if _, writeError := fmt.Fprintf(client.connection, "%d:%s", len(payload), payload); writeError != nil {
		test.Fatal(writeError)
	}

	var response []json.RawMessage
	if unmarshalError := json.Unmarshal(client.readPacket(test), &response); unmarshalError != nil || len(response) != 4 {
		test.Fatalf("bad response to %s: %v", name, unmarshalError)
	}

	if string(response[2]) != "null" {
		test.Fatalf("%s failed: %s", name, response[2])
	}

	return response[3]
}

// script runs privileged JavaScript and decodes its return value into result.
func (client *marionette) script(test *testing.T, source string, result any) {
	test.Helper()

	raw := client.command(test, "WebDriver:ExecuteScript", map[string]any{"script": source, "args": []any{}})

	var wrapper struct {
		Value json.RawMessage `json:"value"`
	}

	if unmarshalError := json.Unmarshal(raw, &wrapper); unmarshalError != nil {
		test.Fatal(unmarshalError)
	}

	if unmarshalError := json.Unmarshal(wrapper.Value, result); unmarshalError != nil {
		test.Fatal(unmarshalError)
	}
}
