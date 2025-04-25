package InfrastructureRepositories

import (
	"time"
)

func (repository *QueueRepository) MigrateSchema() error {
	connection := repository.connect()
	defer connection.Close()

	_, err := connection.Exec(`
    CREATE TABLE IF NOT EXISTS schema_migrations (
        version INT PRIMARY KEY,
        applied_at BIGINT NOT NULL
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`)

	if err != nil {
		return err
	}

	var currentVersion int
	err = connection.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		return err
	}

	migrations := map[int]string{
		1: `CREATE TABLE IF NOT EXISTS queue_messages (
            id VARCHAR(255) NOT NULL,
            name VARCHAR(255) NOT NULL,
            message TEXT NOT NULL,
            published_at BIGINT NOT NULL,
            reserved_at BIGINT DEFAULT 0,
            reserved_by VARCHAR(255) NULL,
            reserved_count INT DEFAULT 0,
            reserved_info TEXT NULL,
            PRIMARY KEY (id),
            INDEX idx_name_reserved_at (name, reserved_at)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
	}

	tx, err := connection.Begin()
	if err != nil {
		return err
	}

	for version, migration := range migrations {
		if version > currentVersion {
			_, err = tx.Exec(migration)
			if err != nil {
				tx.Rollback()
				return err
			}

			_, err = tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
				version, time.Now().Unix())
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}
