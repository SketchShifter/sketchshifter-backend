-- UTF-8mb4を使用する設定
-- データベースのデフォルト文字セットを設定
CREATE DATABASE IF NOT EXISTS processing_platform;
USE processing_platform;
ALTER DATABASE CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- テーブルを削除（存在する場合）
SET FOREIGN_KEY_CHECKS=0;

DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS works;
DROP TABLE IF EXISTS work_tags;
DROP TABLE IF EXISTS likes;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS processing_works;


-- ユーザーテーブル
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    nickname VARCHAR(255) NOT NULL,
    bio TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- タグテーブル
CREATE TABLE tags (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 作品テーブル
CREATE TABLE works (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    file_data LONGBLOB NOT NULL,
    file_type VARCHAR(128) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    thumbnail_data LONGBLOB,
    thumbnail_type VARCHAR(128),
    code_shared BOOLEAN DEFAULT FALSE,
    code_content TEXT,
    views INT DEFAULT 0,
    user_id INT NULL,
    is_guest BOOLEAN DEFAULT FALSE,
    guest_nickname VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 作品とタグの関連付けテーブル
CREATE TABLE work_tags (
    work_id INT NOT NULL,
    tag_id INT NOT NULL,
    PRIMARY KEY (work_id, tag_id),
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- いいねテーブル
CREATE TABLE likes (
    user_id INT NOT NULL,
    work_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, work_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- コメントテーブル
CREATE TABLE comments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    content TEXT NOT NULL,
    work_id INT NOT NULL,
    user_id INT,
    is_guest BOOLEAN DEFAULT FALSE,
    guest_nickname VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Processing作品変換テーブル
CREATE TABLE processing_works (
    id INT AUTO_INCREMENT PRIMARY KEY,
    work_id INT NOT NULL,
    original_name VARCHAR(255),
    pde_content TEXT COMMENT 'PDEファイルの内容を直接保存',
    js_content TEXT COMMENT '変換後のJavaScriptコード',
    canvas_id VARCHAR(255),
    status ENUM('pending', 'processing', 'processed', 'error') DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- インデックスの作成
CREATE INDEX idx_works_user_id ON works(user_id);
CREATE INDEX idx_works_created_at ON works(created_at);
CREATE INDEX idx_works_views ON works(views);
CREATE INDEX idx_comments_work_id ON comments(work_id);
CREATE INDEX idx_processing_works_work_id ON processing_works(work_id);
CREATE INDEX idx_processing_works_status ON processing_works(status);

-- 検索用のフルテキストインデックス
CREATE FULLTEXT INDEX idx_works_title_description ON works(title, description);
CREATE FULLTEXT INDEX idx_tags_name ON tags(name);

ALTER TABLE works 
ADD COLUMN IF NOT EXISTS file_url VARCHAR(255) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS file_public_id VARCHAR(255) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS thumbnail_url VARCHAR(255) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS thumbnail_public_id VARCHAR(255) DEFAULT NULL;

ALTER TABLE works MODIFY file_data LONGBLOB NULL;
ALTER TABLE works MODIFY thumbnail_data LONGBLOB NULL;

ALTER TABLE works 
ADD COLUMN IF NOT EXISTS file_url VARCHAR(255) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS file_public_id VARCHAR(255) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS thumbnail_url VARCHAR(255) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS thumbnail_public_id VARCHAR(255) DEFAULT NULL;

-- 必要に応じて file_data と thumbnail_data をNULL許容に変更
ALTER TABLE works MODIFY file_data LONGBLOB NULL;
ALTER TABLE works MODIFY thumbnail_data LONGBLOB NULL;

SET FOREIGN_KEY_CHECKS=1;


-- システムユーザーの作成（必要な場合）
-- CREATE USER 'processing_app'@'%' IDENTIFIED BY 'your_password_here';
-- GRANT ALL PRIVILEGES ON processing_platform.* TO 'processing_app'@'%';
-- FLUSH PRIVILEGES;