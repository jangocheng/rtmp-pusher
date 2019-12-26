package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/urfave/cli/v2"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
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
				time.Sleep(time.Second)
			})

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	// 监听系统退出信号，确保defer执行
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-signalChan
}

func pushStream(ctx context.Context, pushAddr string, pushFile string) error {
	fmt.Printf("start pushing %s: %s\n", pushFile, pushAddr)
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

	err := cmd.Run()
	if err != nil {
		fmt.Printf("push %s failed: %s\n", pushAddr, err.Error())
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
