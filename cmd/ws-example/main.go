package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	napcat "github.com/ckikoo/napcat_ws"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	wsURL := flag.String("url", "", "NapCat websocket url")
	flag.Parse()

	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	bot := napcat.New(*wsURL, napcat.WithRetryDelay(5*time.Second))
	bot.OnError(func(err error) {
		logger.Error("bot error", zap.Error(err))
	})
	bot.OnPanic(func(p napcat.PanicInfo) {
		logger.Error("panic", zap.Any("recovered", p.Recovered), zap.ByteString("stack", p.Stack))
	})
	bot.OnPrivate(func(e *napcat.PrivateMessageEvent) {
		switch {
		case e.HasText():
			logger.Info("private message",
				zap.String("nick", e.Sender.Nickname),
				zap.Int64("user_id", e.UserID),
				zap.String("text", e.GetText()),
			)

			infos, err := bot.GetGroupMemberList(ctx, 654586770)
			fmt.Printf("infos: %+v\n", infos)
			fmt.Printf("err: %+v\n", err)
		}
	})

	bot.OnPrivateFile(func(pme *napcat.PrivateMessageEvent) {
		fmt.Printf("pme: %+v\n", pme)
		name, fileID, size, ok := pme.GetFile()
		logger.Info("private file",
			zap.String("nick", pme.Sender.Nickname),
			zap.Int64("user_id", pme.UserID),
			zap.Bool("ok", ok),
			zap.String("name", name),
			zap.String("file_id", fileID),
			zap.Int64("size", size),
		)

		go func(fileID, name string) {
			downloadCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			downloadURL, err := bot.GetPrivateFileURL(downloadCtx, fileID)
			if err != nil {
				logger.Error("get private file url", zap.Error(err), zap.String("file_id", fileID))
				return
			}

			savedPath, err := downloadToFile(downloadCtx, downloadURL, name)
			if err != nil {
				logger.Error("download private file", zap.Error(err), zap.String("url", downloadURL))
				return
			}

			logger.Info("downloaded private file",
				zap.String("path", savedPath),
				zap.String("url", downloadURL),
			)
		}(fileID, name)
	})

	bot.OnGroup(func(e *napcat.GroupMessageEvent) {
		name := e.Sender.Card
		if name == "" {
			name = e.Sender.Nickname
		}
		logger.Info("group message",
			zap.Int64("group_id", e.GroupID),
			zap.String("name", name),
			zap.Int64("user_id", e.UserID),
			zap.String("text", e.GetText()),
			zap.Bool("at_me", e.IsAtMe()),
		)
	})
	bot.OnGroupFile(func(pme *napcat.GroupMessageEvent) {
		fmt.Printf("pme: %+v\n", pme)
		name, fileID, size, ok := pme.GetFile()
		logger.Info("private file",
			zap.String("nick", pme.Sender.Nickname),
			zap.Int64("user_id", pme.UserID),
			zap.Bool("ok", ok),
			zap.String("name", name),
			zap.String("file_id", fileID),
			zap.Int64("size", size),
		)

		go func(fileID, name string) {
			downloadCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			downloadURL, err := bot.GetGroupFileURL(downloadCtx, pme.GroupID, fileID)
			if err != nil {
				logger.Error("get private file url", zap.Error(err), zap.String("file_id", fileID))
				return
			}

			savedPath, err := downloadToFile(downloadCtx, downloadURL, name)
			if err != nil {
				logger.Error("download private file", zap.Error(err), zap.String("url", downloadURL))
				return
			}

			logger.Info("downloaded private file",
				zap.String("path", savedPath),
				zap.String("url", downloadURL),
			)
		}(fileID, name)
	})

	if err := bot.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal("bot exit", zap.Error(err))
	}
}

func downloadToFile(ctx context.Context, url string, name string) (string, error) {
	if url == "" {
		return "", errors.New("empty url")
	}

	fileName := sanitizeFileName(name)
	if fileName == "" {
		fileName = "file"
	}

	if err := os.MkdirAll("downloads", 0o755); err != nil {
		return "", err
	}

	dstPath := filepath.Join("downloads", fileName)
	if _, err := os.Stat(dstPath); err == nil {
		ext := filepath.Ext(fileName)
		base := strings.TrimSuffix(fileName, ext)
		dstPath = filepath.Join("downloads", base+"_"+time.Now().Format("20060102_150405")+ext)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("unexpected http status: " + resp.Status)
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return dstPath, nil
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	return name
}
