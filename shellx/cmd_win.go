package shellx

import (
	"bytes"
	"context"
	"fmt"
	"github.com/qiaogw/sub-sdk/cryptx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os/exec"
	"strconv"
	"syscall"
)

type Result struct {
	output string
	err    error
}

// ExecShell 在 Windows 平台上执行指定的 Shell 命令，并可通过 ctx 控制超时或取消。
// 如果在 ctx 超时时间内命令未执行完毕，则会尝试杀死该进程和它的子进程，并返回 DeadlineExceeded 错误。
// 命令输出默认视为 GBK 编码，将被自动转换为 UTF-8 返回。
func ExecShell(ctx context.Context, command string) (string, error) {
	if len(command) == 0 {
		return "", fmt.Errorf("command is empty")
	}

	// 使用 cmd /C 执行完整命令
	cmd := exec.Command("cmd", "/C", command)
	// 隐藏命令行窗口
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// 将输出重定向到同一个 buffer 中，包含 stdout + stderr
	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	// 先启动命令，以便获取到 cmd.Process
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// 使用 goroutine 等待命令执行完毕，将执行结果通过通道传递
	done := make(chan error, 1)
	go func() {
		// Wait 会阻塞直到命令执行完毕
		done <- cmd.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		// ctx 触发取消或超时，需要强制结束命令以及所有子进程
		if cmd.Process != nil {
			pid := cmd.Process.Pid
			if pid > 0 {
				// /F：强制结束进程   /T：结束该进程及其子进程
				killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
				// 此处可以捕获错误记录日志，但不影响后续流程
				_ = killCmd.Run()

				_ = cmd.Process.Kill() // 再次 Kill，确保主进程退出
			}
		}
		// 等待 goroutine 返回，回收资源，防止僵尸进程
		<-done
		return "", status.Error(codes.DeadlineExceeded, "Deadline exceeded")

	case err := <-done:
		// 走到这里表示命令执行完毕(成功或失败)，err 代表 cmd.Wait() 的结果
		outStr := ConvertEncoding(outputBuf.String())
		if err != nil {
			// 若你需要区分是命令内部错误还是其他问题，可在这里作进一步处理
			return outStr, err
		}
		return outStr, nil
	}
}

// ConvertEncoding 将 Windows 上命令行输出(通常是 GBK 编码)转换为 UTF-8。
// 如果转换失败则原样返回输出。
func ConvertEncoding(outputGBK string) string {
	outputUTF8, ok := cryptx.GBK2UTF8(outputGBK)
	if ok {
		return outputUTF8
	}
	return outputGBK
}
