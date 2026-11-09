package accounts

type Kind string

const (
	KindCustomerCurrent Kind = "customer_current"
	KindCustomerSavings Kind = "customer_savings"
	KindMerchant        Kind = "merchant"
	KindHouse           Kind = "house"
	KindSuspense        Kind = "suspense"
)

type AuditLogs []string

type BaseAccount struct {
	AccountID    string
	Kind         Kind
	CurrencyCode string
	AuditLogs
}

func (b *BaseAccount) Audit() []string {
	return b.AuditLogs
}

type Account struct {
	BaseAccount
	Balance        int64
	LockedReason   string
	FailedAttempts int32
	IsLocked       bool
}

func (a *Account) Lock(reason string) {
	a.IsLocked = true
	a.LockedReason = reason
	a.AuditLogs = append(a.AuditLogs, "Account locked: "+reason)
}
