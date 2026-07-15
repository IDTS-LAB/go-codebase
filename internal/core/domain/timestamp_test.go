package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTimestamp(t *testing.T) {
	ts := NewTimestamp()

	assert.False(t, ts.IsZero())
}

func TestTimestampFrom(t *testing.T) {
	now := time.Now().UTC()
	ts := TimestampFrom(now)

	assert.Equal(t, now, ts.Time())
}

func TestTimestampTime(t *testing.T) {
	now := time.Now().UTC()
	ts := TimestampFrom(now)

	assert.Equal(t, now, ts.Time())
}

func TestTimestampString(t *testing.T) {
	now := time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC)
	ts := TimestampFrom(now)

	assert.Equal(t, "2024-01-02T15:04:05Z", ts.String())
}

func TestTimestampIsZero(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		ts := Timestamp{}
		assert.True(t, ts.IsZero())
	})

	t.Run("non-zero value", func(t *testing.T) {
		ts := NewTimestamp()
		assert.False(t, ts.IsZero())
	})
}

func TestTimestampEquals(t *testing.T) {
	now := time.Now().UTC()

	t.Run("equal", func(t *testing.T) {
		a := TimestampFrom(now)
		b := TimestampFrom(now)
		assert.True(t, a.Equals(b))
	})

	t.Run("not equal", func(t *testing.T) {
		a := TimestampFrom(now)
		b := TimestampFrom(now.Add(time.Hour))
		assert.False(t, a.Equals(b))
	})
}

func TestTimestampBefore(t *testing.T) {
	earlier := TimestampFrom(time.Now())
	later := TimestampFrom(time.Now().Add(time.Hour))

	assert.True(t, earlier.Before(later))
	assert.False(t, later.Before(earlier))
}

func TestTimestampAfter(t *testing.T) {
	earlier := TimestampFrom(time.Now())
	later := TimestampFrom(time.Now().Add(time.Hour))

	assert.True(t, later.After(earlier))
	assert.False(t, earlier.After(later))
}

func TestTimestampAdd(t *testing.T) {
	base := TimestampFrom(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	result := base.Add(2 * time.Hour)

	assert.Equal(t, time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC), result.Time())
}

func TestTimestampMarshalText(t *testing.T) {
	now := time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC)
	ts := TimestampFrom(now)

	data, err := ts.MarshalText()

	assert.NoError(t, err)
	assert.Equal(t, []byte("2024-01-02T15:04:05Z"), data)
}

func TestTimestampUnmarshalText(t *testing.T) {
	t.Run("valid text", func(t *testing.T) {
		var ts Timestamp

		err := ts.UnmarshalText([]byte("2024-01-02T15:04:05Z"))

		assert.NoError(t, err)
		assert.Equal(t, 2024, ts.Time().Year())
		assert.Equal(t, time.January, ts.Time().Month())
	})

	t.Run("invalid text", func(t *testing.T) {
		var ts Timestamp

		err := ts.UnmarshalText([]byte("bad"))

		assert.Error(t, err)
	})
}

func TestTimestampValue(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := TimestampFrom(now)

	v, err := ts.Value()

	assert.NoError(t, err)
	assert.Equal(t, now, v)
}

func TestTimestampScan(t *testing.T) {
	t.Run("time.Time input", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		var ts Timestamp

		err := ts.Scan(now)

		assert.NoError(t, err)
		assert.Equal(t, now, ts.Time())
	})

	t.Run("byte slice input", func(t *testing.T) {
		var ts Timestamp

		err := ts.Scan([]byte("2024-01-02T15:04:05Z"))

		assert.NoError(t, err)
		assert.Equal(t, 2024, ts.Time().Year())
	})

	t.Run("string input", func(t *testing.T) {
		var ts Timestamp

		err := ts.Scan("2024-01-02T15:04:05Z")

		assert.NoError(t, err)
		assert.Equal(t, 2024, ts.Time().Year())
	})

	t.Run("unsupported type", func(t *testing.T) {
		var ts Timestamp

		err := ts.Scan(42)

		assert.Error(t, err)
	})
}

func TestTimestampValueAndScan(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := TimestampFrom(now)

	v, err := ts.Value()
	assert.NoError(t, err)

	var scanned Timestamp
	err = scanned.Scan(v)
	assert.NoError(t, err)
	assert.Equal(t, now, scanned.Time())
}
