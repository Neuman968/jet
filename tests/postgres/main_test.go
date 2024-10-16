package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-jet/jet/v2/tests/internal/utils/repo"
	"github.com/jackc/pgx/v4/stdlib"
	"os"
	"runtime"
	"testing"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/tests/dbconfig"
	_ "github.com/lib/pq"
	"github.com/pkg/profile"
	"github.com/stretchr/testify/require"

	_ "github.com/jackc/pgx/v4/stdlib"
)

var db *sql.DB
var testRoot string

var source string

const CockroachDB = "COCKROACH_DB"

func init() {
	source = os.Getenv("PG_SOURCE")
}

func sourceIsCockroachDB() bool {
	return source == CockroachDB
}

func skipForCockroachDB(t *testing.T) {
	if sourceIsCockroachDB() {
		t.SkipNow()
	}
}

func TestMain(m *testing.M) {
	defer profile.Start().Stop()

	setTestRoot()

	for _, driverName := range []string{"pgx", "postgres"} {
		fmt.Printf("\nRunning postgres tests for '%s' driver\n", driverName)

		func() {

			connectionString := dbconfig.PostgresConnectString

			if sourceIsCockroachDB() {
				connectionString = dbconfig.CockroachConnectString
			}

			var err error
			db, err = sql.Open(driverName, connectionString)
			if err != nil {
				fmt.Println(err.Error())
				panic("Failed to connect to test db")
			}
			defer db.Close()

			ret := m.Run()

			if ret != 0 {
				os.Exit(ret)
			}
		}()
	}
}

func setTestRoot() {
	testRoot = repo.GetTestsDirPath()
}

var loggedSQL string
var loggedSQLArgs []interface{}
var loggedDebugSQL string

var queryInfo postgres.QueryInfo
var callerFile string
var callerLine int
var callerFunction string

func init() {
	postgres.SetLogger(func(ctx context.Context, statement postgres.PrintableStatement) {
		loggedSQL, loggedSQLArgs = statement.Sql()
		loggedDebugSQL = statement.DebugSql()
	})

	postgres.SetQueryLogger(func(ctx context.Context, info postgres.QueryInfo) {
		queryInfo = info
		callerFile, callerLine, callerFunction = info.Caller()
	})
}

func requireLogged(t *testing.T, statement postgres.Statement) {
	query, args := statement.Sql()
	require.Equal(t, loggedSQL, query)
	require.Equal(t, loggedSQLArgs, args)
	require.Equal(t, loggedDebugSQL, statement.DebugSql())
}

func requireQueryLogged(t *testing.T, statement postgres.Statement, rowsProcessed int64) {
	query, args := statement.Sql()
	queryLogged, argsLogged := queryInfo.Statement.Sql()

	require.Equal(t, query, queryLogged)
	require.Equal(t, args, argsLogged)
	require.Equal(t, queryInfo.RowsProcessed, rowsProcessed)

	pc, file, _, _ := runtime.Caller(1)
	funcDetails := runtime.FuncForPC(pc)
	require.Equal(t, file, callerFile)
	require.NotEmpty(t, callerLine)
	require.Equal(t, funcDetails.Name(), callerFunction)
}

func skipForPgxDriver(t *testing.T) {
	if isPgxDriver() {
		t.SkipNow()
	}
}

func isPgxDriver() bool {
	switch db.Driver().(type) {
	case *stdlib.Driver:
		return true
	}

	return false
}
