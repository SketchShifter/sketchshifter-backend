#!/bin/bash

# プロジェクトルート
BASE_DIR="sketchshifter_backend"

# 作成したいファイル一覧（階層も含めて）
FILES=(
  "$BASE_DIR/cmd/app/main.go"
  "$BASE_DIR/internal/config/config.go"
  "$BASE_DIR/internal/config/database.go"
  "$BASE_DIR/internal/controllers/auth_controller.go"
  "$BASE_DIR/internal/controllers/comment_controller.go"
  "$BASE_DIR/internal/controllers/health_controller.go"
  "$BASE_DIR/internal/controllers/project_controller.go"
  "$BASE_DIR/internal/controllers/task_controller.go"
  "$BASE_DIR/internal/controllers/user_controller.go"
  "$BASE_DIR/internal/controllers/vote_controller.go"
  "$BASE_DIR/internal/controllers/work_controller.go"
  "$BASE_DIR/internal/middlewares/auth_middleware.go"
  "$BASE_DIR/internal/middlewares/cors.go"
  "$BASE_DIR/internal/middlewares/error_middleware.go"
  "$BASE_DIR/internal/models/models.go"
  "$BASE_DIR/internal/repository/comment_repository.go"
  "$BASE_DIR/internal/repository/project_repository.go"
  "$BASE_DIR/internal/repository/tag_repository.go"
  "$BASE_DIR/internal/repository/task_repository.go"
  "$BASE_DIR/internal/repository/user_repository.go"
  "$BASE_DIR/internal/repository/vote_repository.go"
  "$BASE_DIR/internal/repository/work_repository.go"
  "$BASE_DIR/internal/routes/routes.go"
  "$BASE_DIR/internal/services/auth_service.go"
  "$BASE_DIR/internal/services/cloudinary_service.go"
  "$BASE_DIR/internal/services/comment_service.go"
  "$BASE_DIR/internal/services/project_service.go"
  "$BASE_DIR/internal/services/tag_service.go"
  "$BASE_DIR/internal/services/task_service.go"
  "$BASE_DIR/internal/services/user_service.go"
  "$BASE_DIR/internal/services/vote_service.go"
  "$BASE_DIR/internal/services/work_service.go"
  "$BASE_DIR/internal/utils/file_utils.go"
  "$BASE_DIR/.env.example"
  "$BASE_DIR/.gitignore"
  "$BASE_DIR/docker-compose.yml"
  "$BASE_DIR/Dockerfile"
  "$BASE_DIR/go.mod"
  "$BASE_DIR/go.sum"
  "$BASE_DIR/README.md"
)

# ファイルがなければ作成、親ディレクトリも作成
for FILE in "${FILES[@]}"; do
  DIRNAME="$(dirname "$FILE")"
  if [ ! -d "$DIRNAME" ]; then
    mkdir -p "$DIRNAME"
    echo "Created directory: $DIRNAME"
  fi
  if [ ! -f "$FILE" ]; then
    touch "$FILE"
    echo "Created file: $FILE"
  else
    echo "File already exists: $FILE"
  fi
done
