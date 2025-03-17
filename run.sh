#!/bin/bash

# 依存関係のインストール
go mod tidy

# アプリケーションの実行
go run cmd/app/main.go
