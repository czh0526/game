package mysqlstore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

const (
	// DefaultDSN is the data source name for MySQL on port 3308
	DefaultDSN = "root:password@tcp(localhost:3308)/aries?parseTime=true"
)

// Provider implements storage provider interface for MySQL
type Provider struct {
	db *sql.DB

	mu sync.RWMutex
}

// Config MySQL configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// Store represents a storage database (simplified interface for compatibility)
type Store interface {
	Put(key string, value []byte, tags ...Tag) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	Iterator(startKey, endKey string) Iterator
	Query(expression string, options ...QueryOption) Iterator
	Close() error
}

// Iterator represents a storage iterator
type Iterator interface {
	Next() bool
	Item() *Entry
	Error() error
	Close() error
}

// Entry represents a key-value entry
type Entry struct {
	Key   string
	Value []byte
}

// Tag represents a Name + Value pair that can be associated with a key + value pair
type Tag struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// QueryOption represents an option for a Store.Query call
type QueryOption func(opts *QueryOptions)

// QueryOptions represents various options for Query calls in a store
type QueryOptions struct {
	PageSize int
}

// ErrDataNotFound is returned when data is not found
var ErrDataNotFound = errors.New("data not found")

// NewProvider creates a new MySQL storage provider
func NewProvider(dsn string) (*Provider, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	provider := &Provider{
		db: db,
	}

	// Create tables
	if err := provider.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return provider, nil
}

// NewProviderFromConfig creates a new MySQL storage provider from config
func NewProviderFromConfig(config *Config) (*Provider, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		config.User, config.Password, config.Host, config.Port, config.Database)
	return NewProvider(dsn)
}

// OpenStore opens and returns a store for given name
func (p *Provider) OpenStore(name string) (Store, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	store := &mysqlStore{
		db:    p.db,
		name:  name,
		table: fmt.Sprintf("store_%s", strings.ToLower(name)),
	}

	// Create store table if not exists
	if err := store.createTable(); err != nil {
		return nil, fmt.Errorf("failed to create store table: %w", err)
	}

	return store, nil
}

// Close closes all stores created by this provider
func (p *Provider) Close() error {
	return p.db.Close()
}

// SetStoreConfig sets the store configuration
func (p *Provider) SetStoreConfig(name string, config string) error {
	query := `INSERT INTO store_config (name, config) VALUES (?, ?)
	          ON DUPLICATE KEY UPDATE config = VALUES(config)`
	_, err := p.db.Exec(query, name, config)
	if err != nil {
		return fmt.Errorf("failed to set store config: %w", err)
	}

	return nil
}

// GetStoreConfig gets the store configuration
func (p *Provider) GetStoreConfig(name string) (string, error) {
	var configStr string
	query := `SELECT config FROM store_config WHERE name = ?`
	err := p.db.QueryRow(query, name).Scan(&configStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrDataNotFound
		}
		return "", fmt.Errorf("failed to get store config: %w", err)
	}

	return configStr, nil
}

// initTables creates the necessary database tables
func (p *Provider) initTables() error {
	// Create store config table
	_, err := p.db.Exec(`
		CREATE TABLE IF NOT EXISTS store_config (
			name VARCHAR(255) PRIMARY KEY,
			config JSON NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create store_config table: %w", err)
	}

	return nil
}

// mysqlStore implements Store interface
type mysqlStore struct {
	db    *sql.DB
	name  string
	table string
}

// createTable creates the store table
func (s *mysqlStore) createTable() error {
	_, err := s.db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			key VARCHAR(255) PRIMARY KEY,
			value LONGBLOB NOT NULL,
			tags JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`, s.table))

	if err != nil {
		return fmt.Errorf("failed to create table %s: %w", s.table, err)
	}

	return nil
}

// Put stores the key-value pair
func (s *mysqlStore) Put(k string, v []byte, tags ...Tag) error {
	var tagsJSON *string
	if len(tags) > 0 {
		tagsData, err := json.Marshal(tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}
		tagsStr := string(tagsData)
		tagsJSON = &tagsStr
	}

	query := fmt.Sprintf(`INSERT INTO %s (key, value, tags) VALUES (?, ?, ?)
	                     ON DUPLICATE KEY UPDATE value = VALUES(value), tags = VALUES(tags), updated_at = CURRENT_TIMESTAMP`, s.table)

	_, err := s.db.Exec(query, k, v, tagsJSON)
	if err != nil {
		return fmt.Errorf("failed to put value: %w", err)
	}

	return nil
}

// Get retrieves the value for the given key
func (s *mysqlStore) Get(k string) ([]byte, error) {
	var value []byte
	query := fmt.Sprintf(`SELECT value FROM %s WHERE key = ?`, s.table)
	err := s.db.QueryRow(query, k).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDataNotFound
		}
		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	return value, nil
}

// Delete deletes the key-value pair for the given key
func (s *mysqlStore) Delete(k string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE key = ?`, s.table)
	result, err := s.db.Exec(query, k)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		return ErrDataNotFound
	}

	return nil
}

// Iterator returns an iterator for key-value pairs
func (s *mysqlStore) Iterator(startKey, endKey string) Iterator {
	return &mysqlIterator{
		store:    s,
		startKey: startKey,
		endKey:   endKey,
	}
}

// Query executes a query and returns an iterator
func (s *mysqlStore) Query(expression string, options ...QueryOption) Iterator {
	// For simplicity, we'll return all results and filter in iterator
	return &mysqlIterator{
		store:      s,
		expression: expression,
	}
}

// Close closes the store
func (s *mysqlStore) Close() error {
	return nil
}

// mysqlIterator implements Iterator interface
type mysqlIterator struct {
	store      *mysqlStore
	startKey   string
	endKey     string
	expression string
	rows       *sql.Rows
	current    *Entry
	err        error
	closed     bool
}

// Next moves to the next item
func (it *mysqlIterator) Next() bool {
	if it.closed {
		return false
	}

	// Initialize rows if not done
	if it.rows == nil {
		var query string
		var args []interface{}

		if it.expression != "" {
			// Query with tags
			query = fmt.Sprintf(`SELECT key, value FROM %s WHERE JSON_CONTAINS(tags, ?)`, it.store.table)
			args = []interface{}{it.expression}
		} else {
			// Range query
			if it.startKey != "" && it.endKey != "" {
				query = fmt.Sprintf(`SELECT key, value FROM %s WHERE key >= ? AND key < ? ORDER BY key`, it.store.table)
				args = []interface{}{it.startKey, it.endKey}
			} else {
				query = fmt.Sprintf(`SELECT key, value FROM %s ORDER BY key`, it.store.table)
			}
		}

		rows, err := it.store.db.Query(query, args...)
		if err != nil {
			it.err = err
			return false
		}
		it.rows = rows
	}

	// Move to next row
	if !it.rows.Next() {
		return false
	}

	var key string
	var value []byte
	if err := it.rows.Scan(&key, &value); err != nil {
		it.err = err
		return false
	}

	it.current = &Entry{
		Key:   key,
		Value: value,
	}

	return true
}

// Item returns the current item
func (it *mysqlIterator) Item() *Entry {
	return it.current
}

// Error returns any error encountered
func (it *mysqlIterator) Error() error {
	return it.err
}

// Close closes the iterator
func (it *mysqlIterator) Close() error {
	if it.rows != nil {
		it.closed = true
		return it.rows.Close()
	}
	return nil
}
