package ocr

import (
	"context"
	"fmt"
	"github.com/nekoimi/get-magnet/internal/config"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

type Server struct {
	bin     string
	cmdArgs []string
	cmd     *exec.Cmd
}

func NewServer() *Server {
	return &Server{
		bin:     config.Get().OcrBin,
		cmdArgs: []string{"--address", "127.0.0.1", "--port", fmt.Sprintf("%d", config.OcrPort), "--full"},
	}
}

func (s *Server) Start(ctx context.Context) {
	// 启动 ocr 服务作为子进程
	s.cmd = exec.Command(s.bin, s.cmdArgs...)

	ocrLogger := log.StandardLogger()
	s.cmd.Stdout = ocrLogger.Writer()
	s.cmd.Stderr = ocrLogger.Writer()

	err := s.cmd.Start()
	if err != nil {
		panic("启动OCR服务异常：" + err.Error())
	}

	log.Infof("OCR服务启动成功...")

	err = s.cmd.Wait()
	if err != nil {
		panic("OCR服务异常退出：" + err.Error())
	}
}

func (s *Server) Stop() {
	err := s.cmd.Process.Kill()
	if err != nil {
		log.Warnf("停止OCR服务异常：%s", err.Error())
	}

	log.Debugln("OCR服务停止")
}
