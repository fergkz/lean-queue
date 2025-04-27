package InfrastructureRepositories

import (
	"errors"
	"fmt"
	DomainEntities "lean-queue/src/domain/entities"
	"log"
	"strings"
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

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db
}

func parseDateTime(timeStr string) (time.Time, error) {
	t, err := time.Parse("2006-01-02 15:04:05.999999", timeStr)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse(time.RFC3339Nano, timeStr)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse("2006-01-02T15:04:05.999999", timeStr)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("could not parse time string '%s': %w", timeStr, err)
}

func parseNullableDateTime(timeStr sql.NullString) (*time.Time, error) {
	if !timeStr.Valid || timeStr.String == "" {
		return nil, nil
	}

	t, err := parseDateTime(timeStr.String)
	if err != nil {
		return nil, err
	}

	return &t, nil
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
            reserved_info,
            reserve_expires
        ) 
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	var reservedAtStr interface{} = nil
	if message.GetReservedAt() != nil {
		reservedAtStr = message.GetReservedAt().UTC().Format("2006-01-02 15:04:05.999999")
	}

	publishedAtStr := message.GetPublishedAt().UTC().Format("2006-01-02 15:04:05.999999")

	_, err = stmt.Exec(
		message.GetId(),
		message.GetName().GetValue(),
		message.GetMessage().GetValue(),
		publishedAtStr,
		reservedAtStr,
		message.GetReservedBy(),
		message.GetReservedCount(),
		message.GetReservedInfo(),
		message.GetReserveExpires().UTC().Format("2006-01-02 15:04:05.999999"),
	)

	return err
}

func (repository *QueueRepository) GetById(id string) (*DomainEntities.QueueEntity, error) {
	connection := repository.connect()
	defer connection.Close()

	stmt, err := connection.Prepare(`
        SELECT id, name, message, published_at, reserved_at, reserved_by, reserved_count, reserved_info, reserve_expires
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
	var reserveExpiresStr sql.NullString

	err = stmt.QueryRow(id).Scan(
		&messageId,
		&name,
		&message,
		&publishedAtStr,
		&reservedAtStr,
		&reservedBy,
		&reservedCount,
		&reservedInfo,
		&reserveExpiresStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	publishedAt, err := parseDateTime(publishedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse published_at date: %w", err)
	}

	reservedAt, err := parseNullableDateTime(reservedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reserved_at date: %w", err)
	}

	reserveExpires, err := parseNullableDateTime(reserveExpiresStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reserve_expires date: %w", err)
	}

	if reserveExpires == nil {
		defaultExpiry := time.Now().Add(5 * time.Minute)
		reserveExpires = &defaultExpiry
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
		*reserveExpires,
	)

	return queueEntity, err
}

func (repository *QueueRepository) GetAndReserveMessages(
	queueName DomainEntities.QueueNameEntity,
	limit int,
	messagesBefore time.Time,
	updateReservedAt time.Time,
	updateReservedBy string,
	updateReservedInfo *string,
	updateReservedExpires *time.Time,
) ([]DomainEntities.QueueEntity, error) {
	connection := repository.connect()
	defer connection.Close()

	tx, err := connection.Begin()
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(`
        SELECT id, name, message, published_at, reserved_at, reserved_by, reserved_count, reserved_info, reserve_expires
        FROM queue_messages
        WHERE name = ? 
          AND reserve_expires < ?
        ORDER BY published_at ASC
        LIMIT ?
        FOR UPDATE
    `)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	nowTime := time.Now().UTC()
	nowTimeStr := nowTime.Format("2006-01-02 15:04:05.999999")

	rows, err := stmt.Query(queueName.GetValue(), messagesBefore.UTC().Format("2006-01-02 15:04:05.999999"), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []DomainEntities.QueueEntity
	var messageIds []string

	for rows.Next() {
		var messageId string
		var nameStr string
		var messageStr string
		var publishedAtStr string
		var reservedAtStr *string
		var reservedBy *string
		var reservedCount *int
		var reservedInfo *string
		var reserveExpiresStr sql.NullString

		err = rows.Scan(
			&messageId,
			&nameStr,
			&messageStr,
			&publishedAtStr,
			&reservedAtStr,
			&reservedBy,
			&reservedCount,
			&reservedInfo,
			&reserveExpiresStr,
		)
		if err != nil {
			return nil, err
		}

		publishedAt, err := parseDateTime(publishedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse published_at date: %w", err)
		}

		nameEntity, err := DomainEntities.NewQueueName(nameStr)
		if err != nil {
			return nil, err
		}

		messageEntity, err := DomainEntities.NewQueueMessage(messageStr)
		if err != nil {
			return nil, err
		}

		if reservedCount == nil {
			reservedCount = new(int)
			*reservedCount = 0
		}

		*reservedCount = *reservedCount + 1

		queueEntity, err := DomainEntities.NewQueue(
			&messageId,
			*nameEntity,
			*messageEntity,
			publishedAt,
			&updateReservedAt,
			&updateReservedBy,
			reservedCount,
			updateReservedInfo,
			*updateReservedExpires,
		)

		if err != nil {
			return nil, err
		}

		messages = append(messages, *queueEntity)
		messageIds = append(messageIds, messageId)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(messageIds) == 0 {

		if err = tx.Commit(); err != nil {
			return nil, fmt.Errorf("could not commit transaction: %w", err)
		}
		return messages, nil
	}

	placeholders := make([]string, len(messageIds))
	args := make([]interface{}, 0, len(messageIds)+1)

	args = append(args, nowTimeStr)

	for i := range messageIds {
		placeholders[i] = "?"
		args = append(args, messageIds[i])
	}

	updateQuery := fmt.Sprintf(`
        UPDATE queue_messages 
        SET reserved_at = ?,
			reserved_by = ?,
			reserved_info = ?,
            reserved_count = reserved_count + 1,
            reserve_expires = ?
        WHERE id IN (%s)
    `, strings.Join(placeholders, ","))

	expiresAtStr := updateReservedExpires.UTC().Format("2006-01-02 15:04:05.999999")

	args = []interface{}{updateReservedAt.UTC().Format("2006-01-02 15:04:05.999999"), updateReservedBy, updateReservedInfo, expiresAtStr}
	for _, id := range messageIds {
		args = append(args, id)
	}

	_, err = tx.Exec(updateQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update reserved status: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return messages, nil
}

func (repository *QueueRepository) RemoveById(id string) error {
	connection := repository.connect()
	defer connection.Close()

	stmt, err := connection.Prepare(`
		DELETE FROM queue_messages
		WHERE id = ?
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}

	return nil
}
