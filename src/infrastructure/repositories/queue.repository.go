package InfrastructureRepositories

import (
	"errors"
	"fmt"
	DomainEntities "lean-queue/src/domain/entities"
	"log"
	"time"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type QueueRepository struct {
	dbHost     string
	dbPort     string
	dbUser     string
	dbPassword string
	dbName     string
}

func NewQueueRepository(
	dbHost string,
	dbPort string,
	dbUser string,
	dbPassword string,
	dbName string,
) *QueueRepository {

	repository := &QueueRepository{
		dbHost:     dbHost,
		dbPort:     dbPort,
		dbUser:     dbUser,
		dbPassword: dbPassword,
		dbName:     dbName,
	}

	if err := repository.MigrateSchema(); err != nil {
		log.Printf("Warning: Failed to migrate database schema: %v", err)
	}

	return repository
}

func (repository *QueueRepository) connect() *sql.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		repository.dbUser,
		repository.dbPassword,
		repository.dbHost,
		repository.dbPort,
		repository.dbName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db
}

func (repository *QueueRepository) Save(message DomainEntities.QueueEntity) error {
	connection := repository.connect()
	defer connection.Close()

	if connection == nil {
		return errors.New("database connection failed")
	}

	fmt.Printf("Message full representation: %+v\n", message)

	stmt, err := connection.Prepare(`
        INSERT INTO queue_messages (
            id,
            name,
            message,
            published_at,
            reserved_at,
            reserved_by,
            reserved_count,
            reserved_info
        ) 
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	var reservedAtStr interface{} = nil
	if message.GetReservedAt() != nil {
		reservedAtStr = message.GetReservedAt().Format("2006-01-02 15:04:05.999999-07:00")
	}

	publishedAtStr := message.GetPublishedAt().Format("2006-01-02 15:04:05.999999-07:00")

	_, err = stmt.Exec(
		message.GetId(),
		message.GetName().GetValue(),
		message.GetMessage().GetValue(),
		publishedAtStr,
		reservedAtStr,
		message.GetReservedBy(),
		message.GetReservedCount(),
		message.GetReservedInfo(),
	)

	return err
}

func (repository *QueueRepository) GetById(id string) (*DomainEntities.QueueEntity, error) {
	connection := repository.connect()
	defer connection.Close()

	stmt, err := connection.Prepare(`
        SELECT id, name, message, published_at, reserved_at, reserved_by, reserved_count, reserved_info
        FROM queue_messages
        WHERE id = ?
    `)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var messageId string
	var name string
	var message string
	var publishedAtStr string
	var reservedAtStr sql.NullString
	var reservedBy string
	var reservedCount int
	var reservedInfo string

	err = stmt.QueryRow(id).Scan(
		&messageId,
		&name,
		&message,
		&publishedAtStr,
		&reservedAtStr,
		&reservedBy,
		&reservedCount,
		&reservedInfo,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	publishedAt, err := time.Parse("2006-01-02 15:04:05.999999-07:00", publishedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse published_at date: %w", err)
	}

	var reservedAt *time.Time
	if reservedAtStr.Valid && reservedAtStr.String != "" {
		parsedReservedAt, err := time.Parse("2006-01-02 15:04:05.999999-07:00", reservedAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse reserved_at date: %w", err)
		}
		reservedAt = &parsedReservedAt
	}

	nameEntity, err := DomainEntities.NewQueueName(name)
	if err != nil {
		return nil, err
	}

	messageEntity, err := DomainEntities.NewQueueMessage(message)
	if err != nil {
		return nil, err
	}

	queueEntity, err := DomainEntities.NewQueue(
		&messageId,
		*nameEntity,
		*messageEntity,
		publishedAt,
		reservedAt,
		&reservedBy,
		&reservedCount,
		&reservedInfo,
	)

	return queueEntity, err
}

func (repository *QueueRepository) GetNextByName(name DomainEntities.QueueNameEntity, afterTime time.Time, limit int) ([]DomainEntities.QueueEntity, error) {
	connection := repository.connect()
	defer connection.Close()

	stmt, err := connection.Prepare(`
        SELECT id, name, message, published_at, reserved_at, reserved_by, reserved_count, reserved_info
        FROM queue_messages
        WHERE name = ? AND (reserved_at < ? OR reserved_at IS NULL)
        ORDER BY published_at ASC
        LIMIT ?
    `)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	afterTimeStr := afterTime.Format("2006-01-02 15:04:05.999999-07:00")
	rows, err := stmt.Query(name.GetValue(), afterTimeStr, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []DomainEntities.QueueEntity

	for rows.Next() {
		var messageId string
		var nameStr string
		var messageStr string
		var publishedAtStr string
		var reservedAtStr sql.NullString
		var reservedBy string
		var reservedCount int
		var reservedInfo string

		err = rows.Scan(
			&messageId,
			&nameStr,
			&messageStr,
			&publishedAtStr,
			&reservedAtStr,
			&reservedBy,
			&reservedCount,
			&reservedInfo,
		)
		if err != nil {
			return nil, err
		}

		publishedAt, err := time.Parse("2006-01-02 15:04:05.999999-07:00", publishedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse published_at date: %w", err)
		}

		var reservedAt *time.Time
		if reservedAtStr.Valid && reservedAtStr.String != "" {
			parsedReservedAt, err := time.Parse("2006-01-02 15:04:05.999999-07:00", reservedAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse reserved_at date: %w", err)
			}
			reservedAt = &parsedReservedAt
		}

		nameEntity, err := DomainEntities.NewQueueName(nameStr)
		if err != nil {
			return nil, err
		}

		messageEntity, err := DomainEntities.NewQueueMessage(messageStr)
		if err != nil {
			return nil, err
		}

		queueEntity, err := DomainEntities.NewQueue(
			&messageId,
			*nameEntity,
			*messageEntity,
			publishedAt,
			reservedAt,
			&reservedBy,
			&reservedCount,
			&reservedInfo,
		)

		if err != nil {
			return nil, err
		}

		messages = append(messages, *queueEntity)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
