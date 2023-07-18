package resultparser_test

import (
	"os"
	"testing"

	"github.com/squarefactory/benchmark-api/resultparser"
	"github.com/stretchr/testify/require"
)

func TestWriteResultsToCSV(t *testing.T) {

	tempInputFile := "/tmp/benchmark.log"
	defer os.Remove(tempInputFile)

	wrongPermissionsFile := "/tmp/wrong_permissions.log"
	defer os.Remove(tempInputFile)

	cleanData := `HPL_AI 1 1 1 1 0.001 10.0 1 1 9.5
	HPL_AI 2 2 2 2 0.002 20.0 1 1 19.0`
	permissionData := `HPL_AI 1 1 1 1 0.001 10.0 1 1 9.5
	HPL_AI 2 2 2 2 0.002 20.0 1 1 20`
	err := os.WriteFile(tempInputFile, []byte(cleanData), 0644)
	require.NoError(t, err)
	err = os.WriteFile(wrongPermissionsFile, []byte(permissionData), 0000)
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Positive test",
			input:   "/tmp/benchmark.log",
			wantErr: false,
		},
		{
			name:    "Incorrect permission",
			input:   "/tmp/wrong_permissions.log",
			wantErr: false,
		},
		{
			name:    "File does not exist",
			input:   "/tmp/non_existing_file.txt",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := resultparser.WriteResultsToCSV(tt.input); (err != nil) != tt.wantErr {
				t.Errorf("WriteResultsToCSV() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}