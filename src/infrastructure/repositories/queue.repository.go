package InfrastructureRepositories

import (
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

	_, err = stmt.Exec(
		message.GetId(),
		message.GetName().GetValue(),
		message.GetMessage().GetValue(),
		message.GetPublishedAt().Unix(),
		message.GetReservedAt().Unix(),
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
	var publishedAtUnix int64
	var reservedAtUnix int64
	var reservedBy string
	var reservedCount int
	var reservedInfo string

	err = stmt.QueryRow(id).Scan(
		&messageId,
		&name,
		&message,
		&publishedAtUnix,
		&reservedAtUnix,
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

	publishedAt := time.Unix(publishedAtUnix, 0)
	reservedAt := time.Unix(reservedAtUnix, 0)

	nameEntity, err := DomainEntities.NewQueueName(name)
	if err != nil {
		return nil, err
	}

	messageEntity, err := DomainEntities.NewQueueMessage(message)
	if err != nil {
		return nil, err
	}

	queueEntity := DomainEntities.NewQueue(
		&messageId,
		*nameEntity,
		*messageEntity,
		publishedAt,
		&reservedAt,
		&reservedBy,
		&reservedCount,
		&reservedInfo,
	)

	return queueEntity, nil
}

func (repository *QueueRepository) GetNextByName(name DomainEntities.QueueNameEntity, afterTime time.Time, limit int) ([]DomainEntities.QueueEntity, error) {
	connection := repository.connect()
	defer connection.Close()

	stmt, err := connection.Prepare(`
        SELECT id, name, message, published_at, reserved_at, reserved_by, reserved_count, reserved_info
        FROM queue_messages
        WHERE name = ? AND (reserved_at < ? OR reserved_at = 0)
        ORDER BY published_at ASC
        LIMIT ?
    `)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(name.GetValue(), afterTime.Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []DomainEntities.QueueEntity

	for rows.Next() {
		var messageId string
		var nameStr string
		var messageStr string
		var publishedAtUnix int64
		var reservedAtUnix int64
		var reservedBy string
		var reservedCount int
		var reservedInfo string

		err = rows.Scan(
			&messageId,
			&nameStr,
			&messageStr,
			&publishedAtUnix,
			&reservedAtUnix,
			&reservedBy,
			&reservedCount,
			&reservedInfo,
		)
		if err != nil {
			return nil, err
		}

		publishedAt := time.Unix(publishedAtUnix, 0)
		reservedAt := time.Unix(reservedAtUnix, 0)

		nameEntity, err := DomainEntities.NewQueueName(nameStr)
		if err != nil {
			return nil, err
		}

		messageEntity, err := DomainEntities.NewQueueMessage(messageStr)
		if err != nil {
			return nil, err
		}

		queueEntity := DomainEntities.NewQueue(
			&messageId,
			*nameEntity,
			*messageEntity,
			publishedAt,
			&reservedAt,
			&reservedBy,
			&reservedCount,
			&reservedInfo,
		)

		messages = append(messages, *queueEntity)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
