package process

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"go.uber.org/zap"
)

type ProcessManager struct {
	cmd            *exec.Cmd
	stdoutPipe     *os.File
	stderrPipe     *os.File
	stdinPipe      *os.File
	mutex          sync.Mutex
	logger         *zap.Logger
	serverTemplate string
	cliTemplate    string
}

func NewProcessManager(logger *zap.Logger) *ProcessManager {
	return &ProcessManager{
		logger:         logger,
		serverTemplate: "llama-server -m {model_path} -ngl {ngl}",
		cliTemplate:    "llama-cli -m {model_path} -ngl {ngl}",
	}
}

func (pm *ProcessManager) SetTemplates(serverTemplate, cliTemplate string) {
	pm.serverTemplate = serverTemplate
	pm.cliTemplate = cliTemplate
}

// StartServerHF starts llama-server with a HuggingFace model using -hf flag
func (pm *ProcessManager) StartServerHF(hfModel, quant string, ngl, ctxSize int) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.stopProcessLocked()

	// Format: -hf namespace/model:quant or -hf namespace/model (if no quant)
	hfArg := hfModel
	if quant != "" {
		hfArg = fmt.Sprintf("%s:%s", hfModel, quant)
	}

	cmdStr := strings.NewReplacer(
		"{model_path}", "",
		"{model_name}", hfModel,
		"{ngl}", fmt.Sprintf("%d", ngl),
		"{ctx_size}", fmt.Sprintf("%d", ctxSize),
	).Replace(pm.serverTemplate)

	// Replace -m with -hf, removing the -m and its argument
	args := strings.Fields(cmdStr)
	var newArgs []string
	skipNext := false
	for i, arg := range args {
		if skipNext {
			// Only skip if this doesn't look like a flag
			if !strings.HasPrefix(arg, "-") {
				skipNext = false
				continue
			}
			skipNext = false
		}
		if arg == "-m" {
			newArgs = append(newArgs, "-hf", hfArg)
			// Only skip next if it exists and isn't a flag
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				skipNext = true
			}
		} else if strings.HasPrefix(arg, "-m=") {
			newArgs = append(newArgs, "-hf", hfArg)
		} else {
			newArgs = append(newArgs, arg)
		}
	}

	if pm.logger != nil {
		pm.logger.Info("Starting server with HF model",
			zap.String("hf_model", hfModel),
			zap.String("quant", quant),
			zap.Strings("args", newArgs),
			zap.Int("ngl", ngl))
	}

	pm.cmd = exec.Command(newArgs[0], newArgs[1:]...)
	pm.cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1", "LLAMA_UNBUFFERED=1")

	stdoutPipe, err := pm.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := pm.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	pm.stdoutPipe = stdoutPipe.(*os.File)
	pm.stderrPipe = stderrPipe.(*os.File)

	if pm.logger != nil {
		pm.logger.Info("Server started with HF model", zap.Int("pid", pm.cmd.Process.Pid))
	}
	return nil
}

// StartCLIHF starts llama-cli with a HuggingFace model using -hf flag
func (pm *ProcessManager) StartCLIHF(hfModel, quant string, ngl, ctxSize int) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.stopProcessLocked()

	// Format: -hf namespace/model:quant or -hf namespace/model (if no quant)
	hfArg := hfModel
	if quant != "" {
		hfArg = fmt.Sprintf("%s:%s", hfModel, quant)
	}

	cmdStr := strings.NewReplacer(
		"{model_path}", "",
		"{model_name}", hfModel,
		"{ngl}", fmt.Sprintf("%d", ngl),
		"{ctx_size}", fmt.Sprintf("%d", ctxSize),
	).Replace(pm.cliTemplate)

	// Replace -m with -hf, removing the -m and its argument
	args := strings.Fields(cmdStr)
	var newArgs []string
	skipNext := false
	for i, arg := range args {
		if skipNext {
			// Only skip if this doesn't look like a flag
			if !strings.HasPrefix(arg, "-") {
				skipNext = false
				continue
			}
			skipNext = false
		}
		if arg == "-m" {
			newArgs = append(newArgs, "-hf", hfArg)
			// Only skip next if it exists and isn't a flag
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				skipNext = true
			}
		} else if strings.HasPrefix(arg, "-m=") {
			newArgs = append(newArgs, "-hf", hfArg)
		} else {
			newArgs = append(newArgs, arg)
		}
	}

	if pm.logger != nil {
		pm.logger.Info("Starting CLI with HF model",
			zap.String("hf_model", hfModel),
			zap.String("quant", quant),
			zap.Strings("args", newArgs),
			zap.Int("ngl", ngl),
			zap.Int("ctx_size", ctxSize))
	}

	pm.cmd = exec.Command(newArgs[0], newArgs[1:]...)
	pm.cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1", "LLAMA_UNBUFFERED=1")

	stdinPipe, err := pm.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdoutPipe, err := pm.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := pm.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	pm.stdinPipe = stdinPipe.(*os.File)
	pm.stdoutPipe = stdoutPipe.(*os.File)
	pm.stderrPipe = stderrPipe.(*os.File)

	if pm.logger != nil {
		pm.logger.Info("CLI started with HF model", zap.Int("pid", pm.cmd.Process.Pid))
	}
	return nil
}

func (pm *ProcessManager) StartServer(modelPath, modelName string, ngl, ctxSize int) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.stopProcessLocked()

	cmdStr := strings.NewReplacer(
		"{model_path}", modelPath,
		"{model_name}", modelName,
		"{ngl}", fmt.Sprintf("%d", ngl),
		"{ctx_size}", fmt.Sprintf("%d", ctxSize),
	).Replace(pm.serverTemplate)

	if pm.logger != nil {
		pm.logger.Info("Starting server",
			zap.String("model", modelName),
			zap.String("command", cmdStr),
			zap.Int("ngl", ngl))
	}

	args := strings.Fields(cmdStr)
	pm.cmd = exec.Command(args[0], args[1:]...)
	pm.cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1", "LLAMA_UNBUFFERED=1")

	stdoutPipe, err := pm.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := pm.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	pm.stdoutPipe = stdoutPipe.(*os.File)
	pm.stderrPipe = stderrPipe.(*os.File)

	if pm.logger != nil {
		pm.logger.Info("CLI started", zap.Int("pid", pm.cmd.Process.Pid))
	}
	return nil
}

func (pm *ProcessManager) StartCLI(modelPath, modelName string, ngl, ctxSize int) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.stopProcessLocked()

	cmdStr := strings.NewReplacer(
		"{model_path}", modelPath,
		"{model_name}", modelName,
		"{ngl}", fmt.Sprintf("%d", ngl),
		"{ctx_size}", fmt.Sprintf("%d", ctxSize),
	).Replace(pm.cliTemplate)

	if pm.logger != nil {
		pm.logger.Info("Starting CLI",
			zap.String("model", modelName),
			zap.String("command", cmdStr),
			zap.Int("ngl", ngl),
			zap.Int("ctx_size", ctxSize))
	}

	args := strings.Fields(cmdStr)
	pm.cmd = exec.Command(args[0], args[1:]...)
	pm.cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1", "LLAMA_UNBUFFERED=1")

	stdinPipe, err := pm.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdoutPipe, err := pm.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := pm.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	pm.stdinPipe = stdinPipe.(*os.File)
	pm.stdoutPipe = stdoutPipe.(*os.File)
	pm.stderrPipe = stderrPipe.(*os.File)

	if pm.logger != nil {
		pm.logger.Info("CLI started", zap.Int("pid", pm.cmd.Process.Pid))
	}
	return nil
}

func (pm *ProcessManager) Stop() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.stopProcessLocked()
}

func (pm *ProcessManager) stopProcessLocked() {
	if pm.cmd != nil && pm.cmd.Process != nil {
		if pm.logger != nil {
			pm.logger.Info("Stopping process", zap.Int("pid", pm.cmd.Process.Pid))
		}
		pm.cmd.Process.Kill()
		pm.cmd.Wait()
		pm.cmd = nil
	}

	if pm.stdinPipe != nil {
		pm.stdinPipe.Close()
		pm.stdinPipe = nil
	}

	if pm.stdoutPipe != nil {
		pm.stdoutPipe.Close()
		pm.stdoutPipe = nil
	}

	if pm.stderrPipe != nil {
		pm.stderrPipe.Close()
		pm.stderrPipe = nil
	}
}

func (pm *ProcessManager) IsRunning() bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	return pm.cmd != nil && pm.cmd.Process != nil
}

func (pm *ProcessManager) GetOutputPipes() (*os.File, *os.File) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	return pm.stdoutPipe, pm.stderrPipe
}

func (pm *ProcessManager) GetStdinPipe() *os.File {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	return pm.stdinPipe
}

func (pm *ProcessManager) WriteToStdin(data []byte) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	if pm.stdinPipe == nil {
		return fmt.Errorf("stdin pipe not available")
	}
	_, err := pm.stdinPipe.Write(data)
	return err
}
