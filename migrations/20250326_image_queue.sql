SET FOREIGN_KEY_CHECKS=0;

DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS external_accounts;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS works;
DROP TABLE IF EXISTS work_tags;
DROP TABLE IF EXISTS likes;
DROP TABLE IF EXISTS comments;

SET FOREIGN_KEY_CHECKS=1;

-- ユーザーテーブル
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    nickname VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(512),
    bio TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 外部アカウントテーブル
CREATE TABLE external_accounts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    provider VARCHAR(50) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (provider, external_id)
);

-- タグテーブル
CREATE TABLE tags (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 作品テーブル
CREATE TABLE works (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    file_url VARCHAR(512) NOT NULL,
    thumbnail_url VARCHAR(512),
    code_shared BOOLEAN DEFAULT FALSE,
    code_content TEXT,
    views INT DEFAULT 0,
    user_id INT,
    is_guest BOOLEAN DEFAULT FALSE,
    guest_nickname VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- 作品とタグの関連付けテーブル
CREATE TABLE work_tags (
    work_id INT NOT NULL,
    tag_id INT NOT NULL,
    PRIMARY KEY (work_id, tag_id),
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- いいねテーブル
CREATE TABLE likes (
    user_id INT NOT NULL,
    work_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, work_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE
);

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
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- 画像テーブル
CREATE TABLE images (
    id INT AUTO_INCREMENT PRIMARY KEY,
    work_id INT,
    file_name VARCHAR(255) NOT NULL,
    original_path VARCHAR(512) NOT NULL,
    webp_path VARCHAR(512),
    status ENUM('pending', 'processing', 'processed', 'error') DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE
);

-- Processing作品変換テーブル
CREATE TABLE processing_works (
    id INT AUTO_INCREMENT PRIMARY KEY,
    work_id INT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255),
    pde_path VARCHAR(512) NOT NULL,
    js_path VARCHAR(512),
    canvas_id VARCHAR(255),
    status ENUM('pending', 'processing', 'processed', 'error') DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE
);

-- imagesテーブルの拡張（圧縮情報追加）
ALTER TABLE images
ADD COLUMN original_size BIGINT DEFAULT 0 COMMENT '元のファイルサイズ（バイト）',
ADD COLUMN webp_size BIGINT DEFAULT 0 COMMENT '変換後のWebPファイルサイズ（バイト）',
ADD COLUMN compression_ratio DOUBLE DEFAULT 0 COMMENT '圧縮率（パーセント）',
ADD COLUMN width INT DEFAULT 0 COMMENT '画像の幅（ピクセル）',
ADD COLUMN height INT DEFAULT 0 COMMENT '画像の高さ（ピクセル）';