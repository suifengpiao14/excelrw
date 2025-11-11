package repository_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/suifengpiao14/excelrw/repository"
)

func TestCallbackConfigDDL(t *testing.T) {
	ddl, err := repository.Export_callback_config_table.GenerateDDL()
	require.NoError(t, err)
	println(ddl)
}
