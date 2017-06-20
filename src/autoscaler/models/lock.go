package models

type Lock struct {
	Owner                   string
	Last_Modified_Timestamp int64
	Ttl                     int
}
