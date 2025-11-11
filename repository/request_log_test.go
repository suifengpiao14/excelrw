package repository_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/suifengpiao14/excelrw/repository"
)

func TestRequestLogTableDDL(t *testing.T) {
	ddl, err := repository.Request_log_table.GenerateDDL()
	require.NoError(t, err)
	fmt.Println(ddl)
}
