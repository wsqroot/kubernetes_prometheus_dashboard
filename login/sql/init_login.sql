CREATE DATABASE IF NOT EXISTS login DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE login;

CREATE TABLE IF NOT EXISTS kubernetes_user (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL UNIQUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS kubernetes_passwd (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  password VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_user_id (user_id),
  CONSTRAINT fk_user_passwd FOREIGN KEY (user_id) REFERENCES kubernetes_user(id) ON DELETE CASCADE
);

INSERT INTO kubernetes_user (username)
VALUES ('admin'), ('operator')
ON DUPLICATE KEY UPDATE username = VALUES(username);

INSERT INTO kubernetes_passwd (user_id, password)
SELECT id, 'Admin@123' FROM kubernetes_user WHERE username = 'admin'
ON DUPLICATE KEY UPDATE password = VALUES(password);

INSERT INTO kubernetes_passwd (user_id, password)
SELECT id, 'Operator@123' FROM kubernetes_user WHERE username = 'operator'
ON DUPLICATE KEY UPDATE password = VALUES(password);

CREATE DATABASE IF NOT EXISTS kubeconfig DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE kubeconfig;

CREATE TABLE IF NOT EXISTS imported_kubeconfig (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  source_file VARCHAR(255) NOT NULL,
  content LONGTEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
