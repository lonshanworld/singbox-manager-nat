package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
	"singbox-manager/internal/models"
)

type Store struct {
	dbPath string
	db     *sql.DB
}

func NewStore(basePath string) *Store {
	return &Store{
		dbPath: filepath.Join(basePath, "manager.db"),
	}
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) Init() error {
	if err := os.MkdirAll(filepath.Dir(s.dbPath), 0755); err != nil {
		return err
	}

	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return err
	}
	s.db = db

	// create tables
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		username TEXT PRIMARY KEY,
		uuid TEXT,
		port INTEGER,
		speed_limit_mbps INTEGER,
		data_limit_gb INTEGER,
		used_bytes INTEGER,
		last_seen_bytes INTEGER,
		status TEXT,
		created_at DATETIME,
		expire_date DATETIME,
		telegram_id INTEGER
	);
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);
	`)
	if err != nil {
		return err
	}

	// Initialize default settings if not exists
	_, err = s.GetSettings()
	if err != nil {
		// Insert default settings
		setts := models.Settings{ManagerPort: 8080}
		err = s.SaveSettings(setts)
		if err != nil {
			return err
		}
	}

	return nil
}

// Users methods
func (s *Store) GetUsers() ([]models.User, error) {
	rows, err := s.db.Query(`SELECT username, uuid, port, speed_limit_mbps, data_limit_gb, used_bytes, last_seen_bytes, status, created_at, expire_date, telegram_id FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var expireDate sql.NullTime
		var createdAt time.Time
		var telegramID sql.NullInt64
		if err := rows.Scan(&u.Username, &u.UUID, &u.Port, &u.SpeedLimitMbps, &u.DataLimitGB, &u.UsedBytes, &u.LastSeenBytes, &u.Status, &createdAt, &expireDate, &telegramID); err != nil {
			return nil, err
		}
		u.CreatedAt = createdAt
		if expireDate.Valid {
			u.ExpireDate = &expireDate.Time
		}
		if telegramID.Valid {
			u.TelegramID = telegramID.Int64
		}
		users = append(users, u)
	}
	if users == nil {
		users = []models.User{}
	}
	return users, nil
}

func (s *Store) AddUser(user models.User) error {
	// check if user or port exists
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, user.Username).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("user %s already exists", user.Username)
	}

	var exp interface{}
	if user.ExpireDate != nil {
		exp = *user.ExpireDate
	}

	_, err = s.db.Exec(`INSERT INTO users (username, uuid, port, speed_limit_mbps, data_limit_gb, used_bytes, last_seen_bytes, status, created_at, expire_date, telegram_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.Username, user.UUID, user.Port, user.SpeedLimitMbps, user.DataLimitGB, user.UsedBytes, user.LastSeenBytes, user.Status, user.CreatedAt, exp, user.TelegramID)
	return err
}

func (s *Store) DeleteUser(username string) error {
	res, err := s.db.Exec(`DELETE FROM users WHERE username = ?`, username)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("user %s not found", username)
	}
	return nil
}

func (s *Store) UpdateUser(username string, updateFn func(*models.User)) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var u models.User
	var expireDate sql.NullTime
	var createdAt time.Time
	var telegramID sql.NullInt64
	err = tx.QueryRow(`SELECT username, uuid, port, speed_limit_mbps, data_limit_gb, used_bytes, last_seen_bytes, status, created_at, expire_date, telegram_id FROM users WHERE username = ?`, username).
		Scan(&u.Username, &u.UUID, &u.Port, &u.SpeedLimitMbps, &u.DataLimitGB, &u.UsedBytes, &u.LastSeenBytes, &u.Status, &createdAt, &expireDate, &telegramID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("user %s not found", username)
	} else if err != nil {
		return err
	}

	u.CreatedAt = createdAt
	if expireDate.Valid {
		u.ExpireDate = &expireDate.Time
	}
	if telegramID.Valid {
		u.TelegramID = telegramID.Int64
	}

	updateFn(&u)

	var exp interface{}
	if u.ExpireDate != nil {
		exp = *u.ExpireDate
	}

	_, err = tx.Exec(`UPDATE users SET uuid = ?, port = ?, speed_limit_mbps = ?, data_limit_gb = ?, used_bytes = ?, last_seen_bytes = ?, status = ?, created_at = ?, expire_date = ?, telegram_id = ? WHERE username = ?`,
		u.UUID, u.Port, u.SpeedLimitMbps, u.DataLimitGB, u.UsedBytes, u.LastSeenBytes, u.Status, u.CreatedAt, exp, u.TelegramID, u.Username)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Settings methods
func (s *Store) GetSettings() (models.Settings, error) {
	var val string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = 'app_settings'`).Scan(&val)
	if err != nil {
		return models.Settings{}, err
	}
	var setts models.Settings
	if err := json.Unmarshal([]byte(val), &setts); err != nil {
		return models.Settings{}, err
	}
	return setts, nil
}

func (s *Store) SaveSettings(setts models.Settings) error {
	data, err := json.Marshal(setts)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`INSERT INTO settings (key, value) VALUES ('app_settings', ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, string(data))
	return err
}
