package env

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type probe struct {
	Host string `env:"PROBE_HOST" envDefault:"localhost"`
	Port int    `env:"PROBE_PORT" envDefault:"8080"`
}

func setEnv(t *testing.T, k, v string) {
	t.Helper()
	t.Setenv(k, v)
}

func TestParse_StructTags(t *testing.T) {
	setEnv(t, "PROBE_HOST", "db.internal")
	setEnv(t, "PROBE_PORT", "15432")

	var p probe
	require.NoError(t, Parse(&p))
	assert.Equal(t, "db.internal", p.Host)
	assert.Equal(t, 15432, p.Port)
}

func TestParseAs_Generic(t *testing.T) {
	setEnv(t, "PROBE_PORT", "9090")
	p, err := ParseAs[probe]()
	require.NoError(t, err)
	assert.Equal(t, 9090, p.Port)
}

func TestGet_Has(t *testing.T) {
	assert.Empty(t, Get("ENV_PKG_MISSING_1"))
	assert.False(t, Has("ENV_PKG_MISSING_1"))

	setEnv(t, "ENV_PKG_SET", "x")
	assert.Equal(t, "x", Get("ENV_PKG_SET"))
	assert.True(t, Has("ENV_PKG_SET"))
}

func TestGetInt(t *testing.T) {
	assert.Equal(t, 42, GetInt("ENV_PKG_MISSING_2", 42))
	setEnv(t, "ENV_PKG_INT", "7")
	assert.Equal(t, 7, GetInt("ENV_PKG_INT", 42))
	setEnv(t, "ENV_PKG_INT", "abc")
	assert.Equal(t, 42, GetInt("ENV_PKG_INT", 42), "非法值回退默认")
}

func TestGetInt64(t *testing.T) {
	setEnv(t, "ENV_PKG_I64", "9007199254740993")
	assert.Equal(t, int64(9007199254740993), GetInt64("ENV_PKG_I64", 1))
	assert.Equal(t, int64(1), GetInt64("ENV_PKG_MISSING_3", 1))
}

func TestGetBool(t *testing.T) {
	assert.Equal(t, true, GetBool("ENV_PKG_MISSING_4", true))
	setEnv(t, "ENV_PKG_BOOL", "t")
	assert.True(t, GetBool("ENV_PKG_BOOL", false))
	setEnv(t, "ENV_PKG_BOOL", "notabool")
	assert.False(t, GetBool("ENV_PKG_BOOL", false), "非法值回退默认")
}

func TestGetDuration(t *testing.T) {
	assert.Equal(t, 30*time.Second, GetDuration("ENV_PKG_MISSING_5", 30*time.Second))
	setEnv(t, "ENV_PKG_DUR", "2m")
	assert.Equal(t, 2*time.Minute, GetDuration("ENV_PKG_DUR", time.Second))
	setEnv(t, "ENV_PKG_DUR", "junk")
	assert.Equal(t, time.Second, GetDuration("ENV_PKG_DUR", time.Second))
}

func TestMust(t *testing.T) {
	assert.NotPanics(t, func() {
		t.Setenv("ENV_PKG_REQ", "v")
		assert.Equal(t, "v", Must("ENV_PKG_REQ"))
	})
	assert.PanicsWithValue(t, "env: required variable ENV_PKG_MISSING_6 is not set", func() {
		Must("ENV_PKG_MISSING_6")
	})
}
