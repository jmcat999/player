package storage

import (
	"context"
	"database/sql"
)

func Migrate(ctx context.Context, db *sql.DB) error {
	for _, statement := range schemaStatements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS admin_users (
		id BIGINT NOT NULL AUTO_INCREMENT,
		username VARCHAR(64) NOT NULL,
		password_hash VARCHAR(120) NOT NULL,
		enabled BOOLEAN NOT NULL,
		created_at DATETIME(6) NOT NULL,
		updated_at DATETIME(6) NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_admin_users_username (username)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS sync_config (
		id BIGINT NOT NULL,
		smb_host VARCHAR(255),
		smb_port INT NOT NULL DEFAULT 445,
		smb_domain VARCHAR(255),
		smb_username VARCHAR(255),
		smb_password VARCHAR(255),
		smb_share VARCHAR(255),
		auto_run BOOLEAN NOT NULL DEFAULT TRUE,
		sync_cron VARCHAR(100),
		skip_today BOOLEAN NOT NULL DEFAULT TRUE,
		auto_sync_main_enabled BOOLEAN,
		auto_sync_main_time VARCHAR(5),
		auto_import_main_enabled BOOLEAN,
		auto_import_main_time VARCHAR(5),
		auto_sync_sub_enabled BOOLEAN,
		auto_sync_sub_time VARCHAR(5),
		auto_import_sub_enabled BOOLEAN,
		auto_import_sub_time VARCHAR(5),
		astrbot_api_key VARCHAR(128),
		share_ttl_minutes INT NOT NULL DEFAULT 60,
		PRIMARY KEY (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS sync_source_config (
		id BIGINT NOT NULL AUTO_INCREMENT,
		source_id VARCHAR(100) NOT NULL,
		source_name VARCHAR(100),
		smb_directory VARCHAR(512),
		smb_file_glob VARCHAR(255),
		smb_recursive BOOLEAN NOT NULL DEFAULT FALSE,
		enabled BOOLEAN NOT NULL DEFAULT TRUE,
		PRIMARY KEY (id),
		UNIQUE KEY uk_sync_source_config_source_id (source_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS imported_server_log_files (
		id BIGINT NOT NULL AUTO_INCREMENT,
		server_id VARCHAR(64) NOT NULL,
		server_name VARCHAR(128) NOT NULL,
		remote_path VARCHAR(1024) NOT NULL,
		remote_path_hash CHAR(64) NOT NULL,
		file_name VARCHAR(255) NOT NULL,
		log_date DATE,
		file_size BIGINT NOT NULL,
		last_modified DATETIME(6) NOT NULL,
		content_hash VARCHAR(64) NOT NULL,
		imported_at DATETIME(6) NOT NULL,
		row_count INT NOT NULL,
		ignored_count INT NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_imported_server_log_files_source_path_hash (server_id, remote_path_hash),
		KEY idx_imported_server_log_files_source_imported (server_id, imported_at),
		KEY idx_imported_server_log_files_log_date (log_date)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS player_server_daily_stats (
		id BIGINT NOT NULL AUTO_INCREMENT,
		server_id VARCHAR(64) NOT NULL,
		server_name VARCHAR(128) NOT NULL,
		stat_date DATE NOT NULL,
		player_name VARCHAR(255) NOT NULL,
		broken_count BIGINT NOT NULL,
		placed_count BIGINT NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_player_server_daily_stats_source_day_player (server_id, stat_date, player_name),
		KEY idx_player_server_daily_stats_day_player (stat_date, player_name),
		KEY idx_player_server_daily_stats_server_day (server_id, stat_date)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS player_server_profiles (
		id BIGINT NOT NULL AUTO_INCREMENT,
		server_id VARCHAR(64) NOT NULL,
		server_name VARCHAR(128) NOT NULL,
		player_name VARCHAR(255) NOT NULL,
		first_seen_at DATETIME(6) NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_player_server_profiles_source_player (server_id, player_name),
		KEY idx_player_server_profiles_source (server_id),
		KEY idx_player_server_profiles_player (player_name),
		KEY idx_player_server_profiles_first (first_seen_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS player_server_log_file_seen (
		id BIGINT NOT NULL AUTO_INCREMENT,
		import_file_id BIGINT NOT NULL,
		player_name VARCHAR(255) NOT NULL,
		first_seen_at DATETIME(6) NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_player_server_log_file_seen_file_player (import_file_id, player_name),
		KEY idx_player_server_log_file_seen_file (import_file_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS player_server_log_file_stats (
		id BIGINT NOT NULL AUTO_INCREMENT,
		import_file_id BIGINT NOT NULL,
		stat_date DATE NOT NULL,
		player_name VARCHAR(255) NOT NULL,
		broken_count BIGINT NOT NULL,
		placed_count BIGINT NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_player_server_log_file_stats_file_day_player (import_file_id, stat_date, player_name),
		KEY idx_player_server_log_file_stats_file (import_file_id),
		KEY idx_player_server_log_file_stats_day_player (stat_date, player_name)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS player_server_log_file_ore_stats (
		id BIGINT NOT NULL AUTO_INCREMENT,
		import_file_id BIGINT NOT NULL,
		player_name VARCHAR(255) NOT NULL,
		ore_type VARCHAR(64) NOT NULL,
		ore_count BIGINT NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_player_server_log_file_ore_file_player_type (import_file_id, player_name, ore_type),
		KEY idx_player_server_log_file_ore_file (import_file_id),
		KEY idx_player_server_log_file_ore_player (player_name),
		KEY idx_player_server_log_file_ore_type (ore_type)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS player_server_log_file_wood_stats (
		id BIGINT NOT NULL AUTO_INCREMENT,
		import_file_id BIGINT NOT NULL,
		player_name VARCHAR(255) NOT NULL,
		wood_type VARCHAR(64) NOT NULL,
		wood_count BIGINT NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_player_server_log_file_wood_file_player_type (import_file_id, player_name, wood_type),
		KEY idx_player_server_log_file_wood_file (import_file_id),
		KEY idx_player_server_log_file_wood_player (player_name),
		KEY idx_player_server_log_file_wood_type (wood_type)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS player_server_log_file_sapling_stats (
		id BIGINT NOT NULL AUTO_INCREMENT,
		import_file_id BIGINT NOT NULL,
		player_name VARCHAR(255) NOT NULL,
		sapling_type VARCHAR(64) NOT NULL,
		sapling_count BIGINT NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_player_server_log_file_sapling_file_player_type (import_file_id, player_name, sapling_type),
		KEY idx_player_server_log_file_sapling_file (import_file_id),
		KEY idx_player_server_log_file_sapling_player (player_name),
		KEY idx_player_server_log_file_sapling_type (sapling_type)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS player_server_log_file_milestones (
		id BIGINT NOT NULL AUTO_INCREMENT,
		import_file_id BIGINT NOT NULL,
		player_name VARCHAR(255) NOT NULL,
		milestone_type VARCHAR(64) NOT NULL,
		first_seen_at DATETIME(6) NOT NULL,
		detail VARCHAR(100),
		PRIMARY KEY (id),
		UNIQUE KEY uk_player_server_log_file_milestone_file_player_type (import_file_id, player_name, milestone_type),
		KEY idx_player_server_log_file_milestone_file (import_file_id),
		KEY idx_player_server_log_file_milestone_player (player_name),
		KEY idx_player_server_log_file_milestone_type (milestone_type)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS auto_task_logs (
		id BIGINT NOT NULL AUTO_INCREMENT,
		created_at DATETIME(6) NOT NULL,
		server_id VARCHAR(64),
		server_name VARCHAR(100),
		task_type VARCHAR(32),
		task_label VARCHAR(100),
		status VARCHAR(32),
		message LONGTEXT,
		file_details LONGTEXT,
		scanned_files INT NOT NULL DEFAULT 0,
		success_files INT NOT NULL DEFAULT 0,
		skipped_files INT NOT NULL DEFAULT 0,
		failed_files INT NOT NULL DEFAULT 0,
		PRIMARY KEY (id),
		KEY idx_auto_task_logs_created (created_at),
		KEY idx_auto_task_logs_server (server_id),
		KEY idx_auto_task_logs_status (status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS player_share_tokens (
		token VARCHAR(64) NOT NULL,
		player_name VARCHAR(255) NOT NULL,
		created_at DATETIME(6) NOT NULL,
		expires_at DATETIME(6) NOT NULL,
		PRIMARY KEY (token),
		KEY idx_player_share_tokens_expires (expires_at),
		KEY idx_player_share_tokens_player (player_name)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS ranking_share_tokens (
		token VARCHAR(64) NOT NULL,
		ranking_type VARCHAR(16) NOT NULL,
		limit_count INT NOT NULL,
		from_date DATE,
		to_date DATE,
		created_at DATETIME(6) NOT NULL,
		expires_at DATETIME(6) NOT NULL,
		PRIMARY KEY (token),
		KEY idx_ranking_share_tokens_expires (expires_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS xray_share_tokens (
		token VARCHAR(64) NOT NULL,
		server_id VARCHAR(64) NOT NULL,
		player_name VARCHAR(255) NOT NULL,
		created_at DATETIME(6) NOT NULL,
		expires_at DATETIME(6) NOT NULL,
		payload_json LONGTEXT NOT NULL,
		PRIMARY KEY (token),
		KEY idx_xray_share_tokens_expires (expires_at),
		KEY idx_xray_share_tokens_player (player_name),
		KEY idx_xray_share_tokens_server_player (server_id, player_name)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

	`CREATE TABLE IF NOT EXISTS xray_analysis_jobs (
		job_id VARCHAR(64) NOT NULL,
		server_id VARCHAR(64) NOT NULL,
		server_name VARCHAR(128) NOT NULL,
		status VARCHAR(32) NOT NULL,
		started_at DATETIME(6) NOT NULL,
		finished_at DATETIME(6),
		from_time DATETIME(6),
		to_time DATETIME(6),
		player_name VARCHAR(255),
		dimension VARCHAR(64),
		scanned_files INT NOT NULL,
		scanned_rows BIGINT NOT NULL,
		failed_files INT NOT NULL,
		player_count INT NOT NULL,
		finding_count INT NOT NULL,
		max_risk_score INT NOT NULL,
		message VARCHAR(2000),
		payload_json LONGTEXT NOT NULL,
		PRIMARY KEY (job_id),
		KEY idx_xray_analysis_jobs_server_started (server_id, started_at),
		KEY idx_xray_analysis_jobs_status (status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
}
