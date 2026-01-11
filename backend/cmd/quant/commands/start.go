package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// backendCmd represents the backend command group
var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "백엔드 서버 관련 명령어",
	Long:  `백엔드 서버 시작, 중지 등의 명령어를 제공합니다.`,
}

// frontendCmd represents the frontend command group
var frontendCmd = &cobra.Command{
	Use:   "frontend",
	Short: "프론트엔드 서버 관련 명령어",
	Long:  `프론트엔드 서버 시작, 중지 등의 명령어를 제공합니다.`,
}

// killProcessOnPort kills any process listening on the specified port
func killProcessOnPort(port string) error {
	// lsof로 포트 사용 중인 PID 찾기
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%s", port))
	output, err := cmd.Output()
	if err != nil {
		// 에러면 포트가 사용 중이 아님
		return nil
	}

	// PID 파싱 및 kill
	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, pidStr := range pids {
		if pidStr == "" {
			continue
		}
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		fmt.Printf("🔄 기존 프로세스 종료 중 (PID: %d, Port: %s)...\n", pid, port)

		// kill 프로세스
		killCmd := exec.Command("kill", "-9", pidStr)
		if err := killCmd.Run(); err != nil {
			return fmt.Errorf("프로세스 종료 실패 (PID: %d): %w", pid, err)
		}
	}

	return nil
}

// backendStartCmd starts the backend API server
var backendStartCmd = &cobra.Command{
	Use:   "start",
	Short: "백엔드 API 서버 시작 (포트 8089)",
	Long: `백엔드 API 서버를 시작합니다.

기본 포트: 8089
기존 프로세스가 실행 중이면 자동으로 종료 후 재시작합니다.

Example:
  go run ./cmd/quant backend start
  go run ./cmd/quant backend start --port 8090`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 기존 포트 사용 중인 프로세스 종료
		if err := killProcessOnPort(apiPort); err != nil {
			return err
		}

		// api 명령어를 직접 실행
		return runAPIServer(cmd, args)
	},
}

var frontendPort = "3009"

// frontendStartCmd starts the frontend Next.js server
var frontendStartCmd = &cobra.Command{
	Use:   "start",
	Short: "프론트엔드 Next.js 서버 시작 (포트 3009)",
	Long: `프론트엔드 Next.js 개발 서버를 시작합니다.

기본 포트: 3009
기존 프로세스가 실행 중이면 자동으로 종료 후 재시작합니다.

Example:
  go run ./cmd/quant frontend start
  go run ./cmd/quant frontend start --port 3010`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 기존 포트 사용 중인 프로세스 종료
		if err := killProcessOnPort(frontendPort); err != nil {
			return err
		}

		fmt.Println("=== Aegis v13 Frontend Server ===")
		fmt.Printf("Starting Next.js development server on port %s...\n", frontendPort)

		// frontend 디렉토리 경로
		frontendDir := "../frontend"

		// 현재 디렉토리 확인
		if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
			// backend에서 실행 중이면 상위로 이동
			frontendDir = "../../frontend"
			if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
				return fmt.Errorf("frontend 디렉토리를 찾을 수 없습니다. aegis/v13 프로젝트 루트에서 실행해주세요")
			}
		}

		// pnpm dev 실행
		execCmd := exec.Command("pnpm", "dev", "--port", frontendPort)
		execCmd.Dir = frontendDir
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		fmt.Printf("\n✅ Frontend server starting on http://localhost:%s\n", frontendPort)
		fmt.Println("Press Ctrl+C to stop\n")

		return execCmd.Run()
	},
}

func init() {
	rootCmd.AddCommand(backendCmd)
	rootCmd.AddCommand(frontendCmd)

	// backend 서브커맨드
	backendCmd.AddCommand(backendStartCmd)
	backendStartCmd.Flags().StringVar(&apiPort, "port", "8089", "API 서버 포트")

	// frontend 서브커맨드
	frontendCmd.AddCommand(frontendStartCmd)
	frontendStartCmd.Flags().StringVar(&frontendPort, "port", "3009", "프론트엔드 서버 포트")
}
