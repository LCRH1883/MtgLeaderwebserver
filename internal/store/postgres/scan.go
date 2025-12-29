package postgres

import (
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func textOrEmpty(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}

func uuidOrEmpty(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuidBytesToString(u.Bytes)
}

func uuidBytesToString(b [16]byte) string {
	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf[:])
}

func textArrayOrEmpty(a pgtype.FlatArray[string]) []string {
	if a == nil {
		return nil
	}
	return []string(a)
}

func int4Ptr(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	i := int(v.Int32)
	return &i
}
