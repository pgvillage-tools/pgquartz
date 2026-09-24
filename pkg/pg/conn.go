package pg

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/jackc/pgx/v4"
)

var (
	// ErrUnexpectedRole is raised when we are connected to a database with an unexpected role
	ErrUnexpectedRole = errors.New("we are connected to a database with another role then wished for")
)

// Conn is a connection to a PostgreSQL database.
type Conn struct {
	Type       string `yaml:"type"`
	ConnParams Dsn    `yaml:"conn_params"`
	Role       string `yaml:"role"`
	conn       *pgx.Conn
}

// NewConn returns a new Conn for the given connection parameters.
func NewConn(connParams Dsn) (c *Conn) {
	return &Conn{
		ConnParams: connParams,
	}
}

// DbName returns the database to connect to, from the connection parameters, PGDATABASE or the user name.
func (c *Conn) DbName() (dbName string) {
	value, ok := c.ConnParams["dbname"]
	if ok {
		return value
	}
	value = os.Getenv("PGDATABASE")
	if value != "" {
		return value
	}
	return c.UserName()
}

// UserName returns the user to connect as, from the connection parameters, PGUSER or the current OS user.
func (c *Conn) UserName() (userName string) {
	value, ok := c.ConnParams["user"]
	if ok {
		return value
	}
	value = os.Getenv("PGUSER")
	if value != "" {
		return value
	}
	currentUser, err := user.Current()
	if err != nil {
		log.Panic("cannot determine current user")
	}
	return currentUser.Username
}

// connectStringValue uses proper quoting for connect string values
func connectStringValue(objectName string) (escaped string) {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(objectName, "'", "\\'"))
}

// DSN returns the connection string for this connection.
func (c *Conn) DSN() (dsn string) {
	var pairs []string
	for key, value := range c.ConnParams {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, connectStringValue(value)))
	}
	return strings.Join(pairs[:], " ")
}

// Connect connects to the database, unless an open connection already exists.
func (c *Conn) Connect() (err error) {
	if c.conn != nil {
		if !c.conn.IsClosed() {
			return nil
		}
		c.conn = nil
	}
	c.conn, err = pgx.Connect(ctx, c.DSN())
	if err != nil {
		c.conn = nil
		return err
	}
	return nil
}

// CheckExists runs the query and reports whether it returned at least one row.
func (c *Conn) CheckExists(query string, args ...interface{}) (exists bool, err error) {
	err = c.Connect()
	if err != nil {
		return false, err
	}
	var answer string
	err = c.conn.QueryRow(ctx, query, args...).Scan(&answer)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err == nil {
		return true, nil
	}
	return false, err
}

// Exec runs the query without returning any rows.
func (c *Conn) Exec(query string, args ...interface{}) (err error) {
	err = c.Connect()
	if err != nil {
		return err
	}
	_, err = c.conn.Exec(ctx, query, args...)
	return err
}

// GetOneField runs the query and returns the first field of the first row.
func (c *Conn) GetOneField(query string, args ...interface{}) (answer string, err error) {
	err = c.Connect()
	if err != nil {
		return "", err
	}

	err = c.conn.QueryRow(ctx, query, args...).Scan(&answer)
	if err != nil {
		return "", fmt.Errorf("runQueryGetOneField (%s) failed: %v", query, err)
	}
	return answer, nil
}

// GetAll runs the query and returns all rows.
func (c *Conn) GetAll(query string, args ...interface{}) (answer Result, err error) {
	err = c.Connect()
	if err != nil {
		return answer, err
	}
	var cursor pgx.Rows
	if cursor, err = c.conn.Query(ctx, query, args...); err != nil {
		return answer, err
	}
	for _, header := range cursor.FieldDescriptions() {
		answer.header = append(answer.header, string(header.Name))
	}
	for cursor.Next() {
		var row []string
		var cols []interface{}
		if cols, err = cursor.Values(); err != nil {
			return answer, err
		}
		for _, col := range cols {
			row = append(row, fmt.Sprint(col))
		}
		answer.rows = append(answer.rows, row)
	}
	return answer, nil
}

// VerifyRole checks that the database has the expected role (primary or standby).
// It returns ErrUnexpectedRole when the role does not match.
func (c *Conn) VerifyRole(expected string) error {
	if expected == "" {
		expected = c.Role
	}
	if expected == "all" {
		return nil
	}
	if _, ok := ValidRoles[expected]; !ok {
		return fmt.Errorf("invalid role was specified for conn %s", c.ConnParams.String(true))
	}
	const roleQuery = "select case pg_is_in_recovery() when true then 'standby' else 'primary' end"
	if role, err := c.GetOneField(roleQuery); err != nil {
		return err
	} else if role != expected {
		log.Debugf("actual role %s != expected role %s", role, expected)
		return ErrUnexpectedRole
	}
	log.Debugf("actual role is as expected %s", expected)
	return nil
}
