package model

import (
	"time"

	"cloud.google.com/go/civil"
)

const (
	Term3months Term = "3m"
	Term1year   Term = "1y"
	Term2years  Term = "2y"
	Term3years  Term = "3y"
	Term4years  Term = "4y"
	Term5years  Term = "5y"
	Term6years  Term = "6y"
	Term7years  Term = "7y"
	Term8years  Term = "8y"
	Term9years  Term = "9y"
	Term10years Term = "10y"

	TypeListRate        Type = "list"
	TypeAvgRate         Type = "average"
	TypeRatioDiscounted Type = "ratioDiscounted"
)

type Term string
type Type string
type Bank string

type RatioDiscountBoundary struct {
	MinRatio float32
	MaxRatio float32
}

type InterestSet struct {
	Bank                    Bank
	NominalRate             float32
	EffectiveRate           float32
	Term                    Term
	Type                    Type
	RatioDiscountBoundaries *RatioDiscountBoundary
	UnionDiscount           bool
	ChangedOn               civil.Date
	LastCrawledAt           time.Time
}
