package ocr

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

const Port = 9898

type Server struct {
	bin     string
	cmdArgs []string
	cmd     *exec.Cmd
}

func NewOcrServer(bin string) *Server {
	return &Server{
		bin:     bin,
		cmdArgs: []string{"--address", "127.0.0.1", "--port", fmt.Sprintf("%d", Port), "--full"},
	}
}

func (s *Server) Run() {
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

func (s *Server) Close() error {
	err := s.cmd.Process.Kill()
	if err != nil {
		log.Errorf("停止OCR服务异常：%s", err.Error())
		return err
	}

	log.Infoln("OCR服务停止...")
	return nil
}
