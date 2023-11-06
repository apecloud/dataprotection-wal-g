package main

import (
	"github.com/apecloud/dataprotection-wal-g/cmd/sqlserver"
	_ "github.com/denisenkom/go-mssqldb"
)

func main() {
	sqlserver.Execute()
}
