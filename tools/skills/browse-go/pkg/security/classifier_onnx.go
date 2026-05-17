package security

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// ONNXClassifier bridges to a Python ONNX inference process.
// It loads the TestSavantAI BERT-small model via a subprocess that runs
// a Python script using onnxruntime + transformers.
//
// Architecture note: This mirrors the TS original's design where
// security-classifier.ts runs in a non-compiled Bun script context.
// In Go, we achieve the same isolation by spawning a Python subprocess.
type ONNXClassifier struct {
	name       LayerName
	modelName  string
	hfURL      string
	modelFiles []string
	modelDir   string

	mu       sync.RWMutex
	state    string // "uninitialized" | "loading" | "loaded" | "failed"
	loadErr  string
	cmd      *os.Process
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stdinEnc *json.Encoder
	stdoutDec *json.Decoder
}

// NewTestSavantClassifier creates the TestSavantAI ONNX classifier.
func NewTestSavantClassifier() *ONNXClassifier {
	home, _ := os.UserHomeDir()
	modelsDir := filepath.Join(home, ".gstack", "models")
	return &ONNXClassifier{
		name:       LayerTestsavantContent,
		modelName:  "testsavant-small",
		hfURL:      "https://huggingface.co/testsavantai/prompt-injection-defender-small-v0-onnx/resolve/main",
		modelFiles: []string{"config.json", "tokenizer.json", "tokenizer_config.json", "special_tokens_map.json", "vocab.txt"},
		modelDir:   filepath.Join(modelsDir, "testsavant-small"),
		state:      "uninitialized",
	}
}

// NewDebertaClassifier creates the optional DeBERTa-v3 ensemble classifier.
func NewDebertaClassifier() *ONNXClassifier {
	home, _ := os.UserHomeDir()
	modelsDir := filepath.Join(home, ".gstack", "models")
	return &ONNXClassifier{
		name:       LayerDebertaContent,
		modelName:  "deberta-v3-injection",
		hfURL:      "https://huggingface.co/protectai/deberta-v3-base-injection-onnx/resolve/main",
		modelFiles: []string{"config.json", "tokenizer.json", "tokenizer_config.json", "special_tokens_map.json", "spm.model", "added_tokens.json"},
		modelDir:   filepath.Join(modelsDir, "deberta-v3-injection"),
		state:      "uninitialized",
	}
}

func isDebertaEnabled() bool {
	setting := strings.ToLower(os.Getenv("GSTACK_SECURITY_ENSEMBLE"))
	for _, s := range strings.Split(setting, ",") {
		if strings.TrimSpace(s) == "deberta" {
			return true
		}
	}
	return false
}

func (o *ONNXClassifier) Name() LayerName { return o.name }

func (o *ONNXClassifier) Status() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	switch o.state {
	case "loaded":
		return "ok"
	case "failed":
		return "degraded"
	default:
		return "off"
	}
}

// Load downloads the model (if needed) and starts the Python inference subprocess.
func (o *ONNXClassifier) Load(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if os.Getenv("GSTACK_SECURITY_OFF") == "1" {
		o.state = "failed"
		o.loadErr = "GSTACK_SECURITY_OFF=1"
		return nil
	}

	if o.state == "loaded" {
		return nil
	}
	if o.state == "loading" {
		// Another goroutine is loading; caller should retry or use a sync.Once pattern.
		return fmt.Errorf("classifier %s is already loading", o.modelName)
	}

	o.state = "loading"

	// Ensure model files are staged.
	if err := o.ensureModelStaged(); err != nil {
		o.state = "failed"
		o.loadErr = err.Error()
		return err
	}

	// Check Python + onnxruntime availability.
	if !isPythonAvailable() {
		o.state = "failed"
		o.loadErr = "python3 not available"
		return fmt.Errorf("python3 not available")
	}

	// Start the Python inference subprocess.
	if err := o.startPythonProcess(); err != nil {
		o.state = "failed"
		o.loadErr = err.Error()
		return err
	}

	o.state = "loaded"
	return nil
}

func (o *ONNXClassifier) ensureModelStaged() error {
	onnxDir := filepath.Join(o.modelDir, "onnx")
	if err := os.MkdirAll(onnxDir, 0750); err != nil {
		return err
	}

	for _, f := range o.modelFiles {
		dst := filepath.Join(o.modelDir, f)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := downloadFile(o.hfURL+"/"+f, dst); err != nil {
			return fmt.Errorf("download %s: %w", f, err)
		}
	}

	modelDst := filepath.Join(onnxDir, "model.onnx")
	if _, err := os.Stat(modelDst); err != nil {
		sizeDesc := "112MB"
		if o.name == LayerDebertaContent {
			sizeDesc = "721MB"
		}
		if err := downloadFile(o.hfURL+"/model.onnx", modelDst); err != nil {
			return fmt.Errorf("download model.onnx (%s): %w", sizeDesc, err)
		}
	}

	return nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()
	return os.Rename(tmp, dest)
}

func isPythonAvailable() bool {
	cmd := execCommand("python3", "--version")
	return cmd.Run() == nil
}

func (o *ONNXClassifier) startPythonProcess() error {
	scriptPath, err := o.findClassifierScript()
	if err != nil {
		return err
	}

	cmd := execCommand("python3", scriptPath, "--model-dir", o.modelDir, "--model-name", o.modelName)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return err
	}

	o.cmd = cmd.Process
	o.stdin = stdin
	o.stdout = stdout
	o.stdinEnc = json.NewEncoder(stdin)
	o.stdoutDec = json.NewDecoder(bufio.NewReader(stdout))

	// Send a ping to verify the process is ready.
	if err := o.stdinEnc.Encode(map[string]string{"action": "ping"}); err != nil {
		o.cleanupProcess()
		return fmt.Errorf("ping encode: %w", err)
	}
	var pong map[string]interface{}
	if err := o.stdoutDec.Decode(&pong); err != nil {
		o.cleanupProcess()
		return fmt.Errorf("ping decode: %w", err)
	}
	if pong["status"] != "ready" {
		o.cleanupProcess()
		return fmt.Errorf("classifier not ready: %v", pong)
	}

	return nil
}

func (o *ONNXClassifier) findClassifierScript() (string, error) {
	// Look in multiple locations
	candidates := []string{
		filepath.Join("scripts", "onnx_classifier.py"),
		filepath.Join("..", "scripts", "onnx_classifier.py"),
	}
	// Add path relative to executable
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates,
			filepath.Join(filepath.Dir(exe), "scripts", "onnx_classifier.py"),
			filepath.Join(filepath.Dir(exe), "..", "scripts", "onnx_classifier.py"),
		)
	}
	for _, c := range candidates {
		if abs, err := filepath.Abs(c); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs, nil
			}
		}
	}
	return "", fmt.Errorf("onnx_classifier.py not found")
}

func (o *ONNXClassifier) cleanupProcess() {
	if o.stdin != nil {
		o.stdin.Close()
	}
	if o.stdout != nil {
		o.stdout.Close()
	}
	if o.cmd != nil {
		o.cmd.Kill()
	}
	o.cmd = nil
	o.stdin = nil
	o.stdout = nil
}

// Scan sends text to the Python subprocess and returns the classification result.
func (o *ONNXClassifier) Scan(ctx context.Context, text string) LayerSignal {
	o.mu.RLock()
	state := o.state
	o.mu.RUnlock()

	if state != "loaded" {
		return LayerSignal{Layer: o.name, Confidence: 0, Meta: map[string]interface{}{"degraded": true}}
	}

	req := map[string]interface{}{
		"action": "classify",
		"text":   text,
	}

	done := make(chan struct{})
	var resp map[string]interface{}
	var scanErr error

	go func() {
		defer close(done)
		o.mu.Lock()
		if err := o.stdinEnc.Encode(req); err != nil {
			scanErr = err
			o.mu.Unlock()
			return
		}
		if err := o.stdoutDec.Decode(&resp); err != nil {
			scanErr = err
			o.mu.Unlock()
			return
		}
		o.mu.Unlock()
	}()

	select {
	case <-done:
		if scanErr != nil {
			o.mu.Lock()
			o.state = "failed"
			o.loadErr = scanErr.Error()
			o.mu.Unlock()
			return LayerSignal{Layer: o.name, Confidence: 0, Meta: map[string]interface{}{"degraded": true, "error": scanErr.Error()}}
		}
	case <-ctx.Done():
		return LayerSignal{Layer: o.name, Confidence: 0, Meta: map[string]interface{}{"degraded": true, "error": "timeout"}}
	}

	label, _ := resp["label"].(string)
	score, _ := resp["score"].(float64)

	if label == "INJECTION" {
		return LayerSignal{Layer: o.name, Confidence: score, Meta: map[string]interface{}{"label": label}}
	}
	return LayerSignal{Layer: o.name, Confidence: 0, Meta: map[string]interface{}{"label": label, "safeScore": score}}
}

// execCommand is a thin wrapper for os/exec to allow testing.
var execCommand = defaultExecCommand

func defaultExecCommand(name string, arg ...string) *execCmdWrapper {
	return &execCmdWrapper{Cmd: exec.Command(name, arg...)}
}

// execCmdWrapper wraps exec.Cmd for testability.
type execCmdWrapper struct {
	*exec.Cmd
}

func (e *execCmdWrapper) StdinPipe() (io.WriteCloser, error) { return e.Cmd.StdinPipe() }
func (e *execCmdWrapper) StdoutPipe() (io.ReadCloser, error) { return e.Cmd.StdoutPipe() }
func (e *execCmdWrapper) Start() error                        { return e.Cmd.Start() }
func (e *execCmdWrapper) Run() error                          { return e.Cmd.Run() }

func init() {
	// Warmup note: callers should explicitly call Load() at startup.
	// This init block is empty by design.
}
