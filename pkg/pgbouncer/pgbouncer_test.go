package pgbouncer

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jackc/pgx"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeExecutor struct{ mock.Mock }

func (e fakeExecutor) Query(ctx context.Context, query string, params ...interface{}) (*pgx.Rows, error) {
	args := e.Called(ctx, query, params)
	return args.Get(0).(*pgx.Rows), args.Error(1)
}

func (e fakeExecutor) Execute(ctx context.Context, query string, params ...interface{}) error {
	args := e.Called(ctx, query, params)
	return args.Error(0)
}

func makeTempFile(t *testing.T, prefix string) *os.File {
	tempFile, err := ioutil.TempFile("", prefix)
	require.Nil(t, err, "failed to create temporary file")

	return tempFile
}

func TestGenerateConfig(t *testing.T) {
	t.Run("errors with invalid config template", func(t *testing.T) {
		bouncer := &PgBouncer{
			ConfigFile:         "/etc/pgbouncer/pgbouncer.ini",
			ConfigTemplateFile: "/this/does/not/exist",
		}

		err := bouncer.GenerateConfig("curly.db.ams.gc.cx")

		assert.Error(t, err, "expected config generation to fail")
		assert.Equal(t,
			"failed to read PgBouncer config template file: open /this/does/not/exist: no such file or directory",
			err.Error(),
		)
	})

	t.Run("writes config with host when successful", func(t *testing.T) {
		tempConfigFile := makeTempFile(t, "pgbouncer-config-")
		defer os.Remove(tempConfigFile.Name())

		bouncer := &PgBouncer{
			ConfigFile:         tempConfigFile.Name(),
			ConfigTemplateFile: "./testdata/pgbouncer.ini.template",
		}

		err := bouncer.GenerateConfig("curly.db.ams.gc.cx")
		assert.Nil(t, err, "failed to generate config")

		configBuffer, _ := ioutil.ReadFile(tempConfigFile.Name())
		assert.Contains(t, string(configBuffer),
			"postgres = host=curly.db.ams.gc.cx", "expected host to be in generated config")
	})
}

func TestPause(t *testing.T) {
	testCases := []struct {
		name        string
		psqlError   error                   // error returned from PsqlExecutor
		assertError func(*testing.T, error) // assertions on the Pause() error
	}{
		{
			"when pause is successful",
			nil,
			func(t *testing.T, err error) {
				assert.Nil(t, err, "expected Pause to return no error")
			},
		},
		{
			"when pause fails",
			errors.New("timeout"),
			func(t *testing.T, err error) {
				assert.Equal(t, "failed to pause PgBouncer: timeout", err.Error())
			},
		},
		// If PgBouncer is already paused then we'll receive a specific error message. Verify
		// that the Pause command will succeed in this case, as it has no work to do.
		{
			"when already paused",
			pgx.PgError{Code: "08P01", Message: "already suspended/paused"},
			func(t *testing.T, err error) {
				assert.Nil(t, err, "expected Pause to return no error")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var noParams []interface{}
			exec := new(fakeExecutor)
			bouncer := &PgBouncer{Executor: exec}

			exec.
				On("Execute", context.TODO(), "PAUSE;", noParams).
				Return(tc.psqlError)
			err := bouncer.Pause(context.TODO())

			exec.AssertExpectations(t)
			tc.assertError(t, err)
		})
	}
}

func TestReload(t *testing.T) {
	testCases := []struct {
		name        string
		psqlError   error                   // error returned from PsqlExecutor
		assertError func(*testing.T, error) // assertions on the Reload() error
	}{
		{
			"when reload is successful",
			nil,
			func(t *testing.T, err error) {
				assert.Nil(t, err, "expected Reload to return no error")
			},
		},
		{
			"when reload is successful",
			errors.New("timeout"),
			func(t *testing.T, err error) {
				assert.Equal(t, "failed to reload PgBouncer: timeout", err.Error())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var noParams []interface{}
			exec := new(fakeExecutor)
			bouncer := &PgBouncer{Executor: exec}

			exec.
				On("Execute", context.TODO(), "RELOAD;", noParams).
				Return(tc.psqlError)
			err := bouncer.Reload(context.TODO())

			exec.AssertExpectations(t)
			tc.assertError(t, err)
		})
	}
}

func TestResume(t *testing.T) {
	testCases := []struct {
		name        string
		psqlError   error                   // error returned from PsqlExecutor
		assertError func(*testing.T, error) // assertions on the Resume() error
	}{
		{
			"when resume is successful",
			nil,
			func(t *testing.T, err error) {
				assert.Nil(t, err, "expected Resume to return no error")
			},
		},
		{
			"when reload is successful",
			errors.New("timeout"),
			func(t *testing.T, err error) {
				assert.Equal(t, "failed to resume PgBouncer: timeout", err.Error())
			},
		},
		{
			"when already resumed",
			pgx.PgError{Code: "08P01", Message: "Pooler is not paused/suspended"},
			func(t *testing.T, err error) {
				assert.Nil(t, err, "expected Resume to return no error")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var noParams []interface{}
			exec := new(fakeExecutor)
			bouncer := &PgBouncer{Executor: exec}

			exec.
				On("Execute", context.TODO(), "RESUME;", noParams).
				Return(tc.psqlError)
			err := bouncer.Resume(context.TODO())

			exec.AssertExpectations(t)
			tc.assertError(t, err)
		})
	}
}