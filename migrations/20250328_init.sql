-- SketchShifter プラットフォーム 完全マイグレーションスクリプト
-- このスクリプトはデータベース全体の作成と修正を行います

-- 1. データベースとユーザーの作成（最初のセットアップ時のみ必要）
-- 注意: ルート権限が必要です。必要に応じてコメントアウトしてください
CREATE DATABASE IF NOT EXISTS processing_platform CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'processing_user'@'%' IDENTIFIED BY 'processing_password';
GRANT ALL PRIVILEGES ON processing_platform.* TO 'processing_user'@'%';
FLUSH PRIVILEGES;

-- 2. データベースの選択
USE processing_platform;

-- 3. 既存のテーブルがある場合は削除（初回実行時に必要ない場合はコメントアウト）
SET FOREIGN_KEY_CHECKS=0;

DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS external_accounts;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS works;
DROP TABLE IF EXISTS work_tags;
DROP TABLE IF EXISTS likes;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS images;
DROP TABLE IF EXISTS processing_works;

SET FOREIGN_KEY_CHECKS=1;

-- 4. データベースの文字セットを確認し設定
ALTER DATABASE processing_platform CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 5. セッション変数を設定
SET NAMES utf8mb4;
SET character_set_client = utf8mb4;
SET character_set_connection = utf8mb4;
SET character_set_results = utf8mb4;

-- 6. テーブルの作成
-- 6.1 ユーザーテーブル
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL UNIQUE,
    password VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    nickname VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    avatar_url VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    bio TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6.2 外部アカウントテーブル
CREATE TABLE external_accounts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    provider VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    external_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (provider, external_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6.3 タグテーブル
CREATE TABLE tags (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6.4 作品テーブル
CREATE TABLE works (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    description TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    file_url VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    thumbnail_url VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    code_shared BOOLEAN DEFAULT FALSE,
    code_content TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    views INT DEFAULT 0,
    user_id INT,
    is_guest BOOLEAN DEFAULT FALSE,
    guest_nickname VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6.5 作品とタグの関連付けテーブル
CREATE TABLE work_tags (
    work_id INT NOT NULL,
    tag_id INT NOT NULL,
    PRIMARY KEY (work_id, tag_id),
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6.6 いいねテーブル
CREATE TABLE likes (
    user_id INT NOT NULL,
    work_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, work_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6.7 コメントテーブル
CREATE TABLE comments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    content TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    work_id INT NOT NULL,
    user_id INT,
    is_guest BOOLEAN DEFAULT FALSE,
    guest_nickname VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6.8 画像テーブル
CREATE TABLE images (
    id INT AUTO_INCREMENT PRIMARY KEY,
    work_id INT,
    file_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    original_path VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    webp_path VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    status ENUM('pending', 'processing', 'processed', 'error') DEFAULT 'pending',
    error_message TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    original_size BIGINT DEFAULT 0 COMMENT '元のファイルサイズ（バイト）',
    webp_size BIGINT DEFAULT 0 COMMENT '変換後のWebPファイルサイズ（バイト）',
    compression_ratio DOUBLE DEFAULT 0 COMMENT '圧縮率（パーセント）',
    width INT DEFAULT 0 COMMENT '画像の幅（ピクセル）',
    height INT DEFAULT 0 COMMENT '画像の高さ（ピクセル）',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6.9 Processing作品変換テーブル
CREATE TABLE processing_works (
    id INT AUTO_INCREMENT PRIMARY KEY,
    work_id INT NOT NULL,
    file_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    original_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    pde_content TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    pde_path VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    js_path VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    canvas_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    status ENUM('pending', 'processing', 'processed', 'error') DEFAULT 'pending',
    error_message TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7. タグ重複問題を解決するためのストアドプロシージャ
DELIMITER //
CREATE PROCEDURE add_tag_to_work(IN p_work_id INT, IN p_tag_id INT)
BEGIN
    DECLARE tag_exists INT DEFAULT 0;
    
    -- タグ関連付けが既に存在するか確認
    SELECT COUNT(*) INTO tag_exists 
    FROM work_tags 
    WHERE work_id = p_work_id AND tag_id = p_tag_id;
    
    -- 存在しない場合のみ挿入
    IF tag_exists = 0 THEN
        INSERT INTO work_tags (work_id, tag_id) VALUES (p_work_id, p_tag_id);
    END IF;
END //
DELIMITER ;

-- 8. 既存のwork_tagsテーブルに重複がある場合の修正プロシージャ
DELIMITER //
CREATE PROCEDURE fix_duplicate_work_tags()
BEGIN
    -- 一時テーブルを作成して重複を削除
    CREATE TEMPORARY TABLE temp_work_tags AS
    SELECT DISTINCT work_id, tag_id FROM work_tags;
    
    -- 既存テーブルを削除して重複のないデータだけを挿入
    TRUNCATE TABLE work_tags;
    
    INSERT INTO work_tags (work_id, tag_id)
    SELECT work_id, tag_id FROM temp_work_tags;
    
    -- 一時テーブルをドロップ
    DROP TEMPORARY TABLE IF EXISTS temp_work_tags;
    
    SELECT 'ワークタグテーブルの重複を修正しました' AS Message;
END //
DELIMITER ;

-- 重複修正プロシージャを実行
CALL fix_duplicate_work_tags();

-- 9. 初期テストデータの挿入（オプション）
-- 注意: 本番環境では不要な場合はコメントアウトしてください
INSERT INTO users (email, password, name, nickname, bio)
VALUES 
('admin@example.com', '$2a$10$xVCf9qQOsXqX5inNOGGnmO3NVlBy5xS.qaQ9wj8TUq/UsBKKvhOzK', '管理者', 'Admin', 'システム管理者です'),
('user@example.com', '$2a$10$f0D8hJFuS1G1mY4VjGIwauvEjR1hUEnesZJi1.4o.KuR1a3PZg2eO', '一般ユーザー', 'User', '一般ユーザーです');

-- 10. indexの追加（パフォーマンス向上）
-- 作品検索用インデックス
CREATE INDEX idx_works_title ON works (title(191));
CREATE INDEX idx_works_user_id ON works (user_id);
CREATE INDEX idx_works_created_at ON works (created_at);

-- タグ検索用インデックス
CREATE INDEX idx_tags_name ON tags (name(50));

-- コメント検索用インデックス
CREATE INDEX idx_comments_work_id ON comments (work_id);
CREATE INDEX idx_comments_user_id ON comments (user_id);

-- 11. 重要なシステム情報の表示
SELECT 
    CONCAT('データベース `', DATABASE(), '` に ', COUNT(*), ' テーブルが正常に作成されました。') AS 'マイグレーション結果',
    @@character_set_database AS 'データベース文字セット',
    @@collation_database AS 'データベース照合順序'
FROM information_schema.tables 
WHERE table_schema = DATABASE();

-- 12. 複合確認クエリ：各テーブルの状態を確認
SELECT 'テーブル情報' AS Info, table_name AS 'テーブル名', 
    table_rows AS '行数', 
    table_collation AS '照合順序',
    create_time AS '作成時間',
    engine AS 'エンジン'
FROM information_schema.tables 
WHERE table_schema = DATABASE()
ORDER BY table_name;