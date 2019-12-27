package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	logPath = os.Getenv("LOG_PATH")
)

var rtmpFileFlag = &cli.StringFlag{
	Name:    "rtmp",
	Aliases: []string{"r"},
	Usage:   "Load configuration from `FILE`",
}

var videoFileFlag = &cli.StringFlag{
	Name:    "video",
	Aliases: []string{"v"},
	Usage:   "Load configuration from `FILE`",
}

func main() {
	var rtmpStreams int
	app := &cli.App{
		Flags: []cli.Flag{rtmpFileFlag, videoFileFlag},
		Action: func(c *cli.Context) error {
			rtmpFilePath := c.String(rtmpFileFlag.Name)
			videoFilePath := c.String(videoFileFlag.Name)
			if rtmpFilePath == "" || videoFilePath == "" {
				return fmt.Errorf("invalid arg")
			}

			rangeFileLine(rtmpFilePath, func(rtmpAddr string) {
				if rtmpAddr == "" {
					return
				}

				go pushStream(context.TODO(), rtmpAddr, videoFilePath)
				rtmpStreams++
				time.Sleep(time.Second)
			})

			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.WithError(err).Fatalf("run error")
	}
	if rtmpStreams <= 0 {
		log.Infof("no rtmp stream to push")
		return
	}

	// 监听系统退出信号
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	s := <-signalChan
	log.Infof("quit with %s", s.String())
}

func pushStream(ctx context.Context, pushAddr string, pushFile string) error {
	log.Infof("start pushing %s: %s", pushFile, pushAddr)
	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-re",
		"-stream_loop", "-1",
		"-i", pushFile,
		"-vcodec", "copy",
		"-acodec", "aac",
		"-ar", "44100",
		"-f", "flv",
		pushAddr,
	)
	errOut := bytes.Buffer{}
	cmd.Stdout = &errOut
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		log.Errorf("push %s failed: %s, %s", pushAddr, err.Error(), errOut.String())
		return err
	}

	return nil
}

func rangeFileLine(filePath string, f func(line string)) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := bufio.NewReader(file)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		f(line)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func setupLogger() {
	logFilePath := filepath.Join(logPath, "main.log")
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.WithError(err).Fatalf("open file error: %s")
	}
	log.SetOutput(file)
}
