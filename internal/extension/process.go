package extension

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"
)

// Process wraps a single running extension subprocess.
type Process struct {
	protocol string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	scanner  *bufio.Scanner
	mu       sync.Mutex
	version  string
}

// StartProcess spawns the extension binary and performs a ping handshake.
func StartProcess(binPath, protocol string) (*Process, error) {
	cmd := exec.Command(binPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	// Pipe stderr to log
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting extension %q: %w", protocol, err)
	}

	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			log.Printf("[ext:%s] %s", protocol, s.Text())
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max line

	p := &Process{
		protocol: protocol,
		cmd:      cmd,
		stdin:    stdin,
		scanner:  scanner,
	}

	// Ping handshake with timeout
	resp, err := p.sendWithTimeout(Request{Action: ActionPing}, 5*time.Second)
	if err != nil {
		p.kill()
		return nil, fmt.Errorf("extension %q ping failed: %w", protocol, err)
	}
	if !resp.OK {
		p.kill()
		return nil, fmt.Errorf("extension %q ping returned error: %s", protocol, resp.Error)
	}
	p.version = resp.Version

	log.Printf("Extension %q started (version: %s, pid: %d)", protocol, p.version, cmd.Process.Pid)
	return p, nil
}

// Send sends a request and waits for the response (mutex-serialized).
func (p *Process) Send(req Request) (*Response, error) {
	return p.sendWithTimeout(req, 0)
}

func (p *Process) sendWithTimeout(req Request, timeout time.Duration) (*Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	data = append(data, '\n')

	if _, err := p.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("writing to extension: %w", err)
	}

	type result struct {
		resp *Response
		err  error
	}

	ch := make(chan result, 1)
	go func() {
		if !p.scanner.Scan() {
			if err := p.scanner.Err(); err != nil {
				ch <- result{err: fmt.Errorf("reading from extension: %w", err)}
			} else {
				ch <- result{err: fmt.Errorf("extension %q closed stdout", p.protocol)}
			}
			return
		}
		var resp Response
		if err := json.Unmarshal(p.scanner.Bytes(), &resp); err != nil {
			ch <- result{err: fmt.Errorf("parsing extension response: %w", err)}
			return
		}
		ch <- result{resp: &resp}
	}()

	if timeout > 0 {
		select {
		case r := <-ch:
			return r.resp, r.err
		case <-time.After(timeout):
			return nil, fmt.Errorf("extension %q timed out after %s", p.protocol, timeout)
		}
	}

	r := <-ch
	return r.resp, r.err
}

// Version returns the version reported by the extension during ping.
func (p *Process) Version() string {
	return p.version
}

// IsAlive checks whether the extension process is still running.
func (p *Process) IsAlive() bool {
	if p.cmd.Process == nil {
		return false
	}
	// ProcessState is set after Wait; nil means still running
	return p.cmd.ProcessState == nil
}

// Stop sends a shutdown action, closes stdin, and waits for the process to exit.
func (p *Process) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Best-effort shutdown request
	data, _ := json.Marshal(Request{Action: ActionShutdown})
	data = append(data, '\n')
	p.stdin.Write(data)
	p.stdin.Close()

	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		p.cmd.Process.Kill()
		return fmt.Errorf("extension %q killed after 5s timeout", p.protocol)
	}
}

func (p *Process) kill() {
	p.stdin.Close()
	if p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
	p.cmd.Wait()
}
