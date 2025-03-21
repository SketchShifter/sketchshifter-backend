#!/bin/bash

# 起動
make prod

# 少し待つ
sleep 5

# マイグレーション
make migrate-up


# ログ
make prod-logs