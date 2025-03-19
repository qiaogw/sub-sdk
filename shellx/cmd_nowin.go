//go:build !windows
// +build !windows

package shellx

import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os/exec"
	"syscall"
)

type Result struct {
	output string
	err    error
}

// ExecShell 通过 bash 或 sh 执行 Shell 命令，并可设置执行超时时间（通过 ctx 控制）。
// 若超时或取消，将会尝试杀掉进程组及其所有子进程，并返回 DeadlineExceeded 错误。
func ExecShell(ctx context.Context, command string) (string, error) {
	shellPath, err := FindShell()
	if err != nil {
		return "", err
	}

	// 创建 Cmd 结构体
	cmd := exec.Command(shellPath, "-c", command)

	// 设置新进程组，以便后续能杀掉整个组（包括子进程）。
	// 如果 Pgid == 0 且 Setpgid == true，Go 会将子进程的组 ID 设置为子进程 PID。
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// 可以使用 buffered channel，防止 goroutine 在无人接收时阻塞
	resultChan := make(chan Result, 1)

	// 启动子 goroutine 执行命令
	go func() {
		// 同步执行获取输出（stdout + stderr），阻塞直到命令完成
		output, err := cmd.CombinedOutput()
		// 尝试发送到通道；如果主 goroutine已经超时返回，也不会卡住
		select {
		case resultChan <- Result{string(output), err}:
		default:
			// 若主 goroutine超时/取消后不再读取通道，这里避免阻塞。
		}
	}()

	// 在这里等待执行结果或 ctx 超时
	select {
	case <-ctx.Done():
		// ctx.Done() 触发时，说明超时或手动取消
		// 尝试杀掉整个进程组: -cmd.Process.Pid 表示杀掉该进程组
		if cmd.Process != nil && cmd.Process.Pid > 0 {
			pgid := -cmd.Process.Pid
			// 向整个进程组发送 SIGKILL
			_ = syscall.Kill(pgid, syscall.SIGKILL)
		}
		// 最终返回 DeadlineExceeded
		return "", status.Error(codes.DeadlineExceeded, "Deadline exceeded")

	case result := <-resultChan:
		// 正常完成
		return result.Output, result.Err
	}
}

// FindShell 查找系统可用的 Shell，优先使用 bash，然后退回到 sh。
// 若系统不存在 bash 与 sh，则返回错误。
func FindShell() (string, error) {
	bashPath, err := exec.LookPath("bash")
	if err == nil {
		return bashPath, nil
	}
	shPath, err := exec.LookPath("sh")
	if err != nil {
		return "", fmt.Errorf("neither bash nor sh found on the system: %w", err)
	}
	return shPath, nil
}
