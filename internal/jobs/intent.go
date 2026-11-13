package jobs

import (
	"hash/crc32"
	"strconv"
	"strings"
)

type OpType string

const (
	OpSingleTransfer OpType = "single_transfer"
	OpBulkTransfer   OpType = "bulk_transfer"
	OpHouseTransfer  OpType = "house_transfer"
)

type Recipient struct {
	AccountID string
	Weight    int64
}

type Intent struct {
	Type          OpType
	AccountID     string
	FromAccountID string
	ToAccountID   string
	CurrencyCode  string
	Amount        int64
	Recipients    []Recipient
	Note          string
}

func GenerateJobID(intent Intent) string {
	var b strings.Builder
	b.WriteString(string(intent.Type))
	b.WriteString("|")
	b.WriteString(intent.AccountID)
	b.WriteString("|")
	b.WriteString(intent.FromAccountID)
	b.WriteString("|")
	b.WriteString(intent.ToAccountID)
	b.WriteString("|")
	b.WriteString(intent.CurrencyCode)
	b.WriteString("|")
	b.WriteString(strconv.FormatInt(intent.Amount, 10))
	b.WriteString("|")
	for _, r := range intent.Recipients {
		b.WriteString(r.AccountID)
		b.WriteString(":")
		b.WriteString(strconv.FormatInt(r.Weight, 10))
		b.WriteString(",")
	}

	sum := crc32.ChecksumIEEE([]byte(b.String()))
	return strconv.FormatUint(uint64(sum), 16)
}
