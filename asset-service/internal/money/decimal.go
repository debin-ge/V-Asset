package money

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

const scale = 8

type Decimal struct {
	rat *big.Rat
}

func Zero() Decimal {
	return Decimal{}
}

func FromInt64(v int64) Decimal {
	return Decimal{rat: new(big.Rat).SetInt64(v)}
}

func Parse(value string) (Decimal, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return Zero(), nil
	}

	rat, ok := new(big.Rat).SetString(trimmed)
	if !ok {
		return Zero(), fmt.Errorf("invalid decimal: %s", value)
	}

	return Decimal{rat: rat}, nil
}

func MustParse(value string) Decimal {
	decimal, err := Parse(value)
	if err != nil {
		panic(err)
	}
	return decimal
}

func (d Decimal) clone() *big.Rat {
	if d.rat == nil {
		return new(big.Rat)
	}
	return new(big.Rat).Set(d.rat)
}

func (d Decimal) IsZero() bool {
	return d.clone().Sign() == 0
}

func (d Decimal) Cmp(other Decimal) int {
	return d.clone().Cmp(other.clone())
}

func (d Decimal) Add(other Decimal) Decimal {
	return Decimal{rat: new(big.Rat).Add(d.clone(), other.clone())}
}

func (d Decimal) Sub(other Decimal) Decimal {
	return Decimal{rat: new(big.Rat).Sub(d.clone(), other.clone())}
}

func (d Decimal) Mul(other Decimal) Decimal {
	return Decimal{rat: new(big.Rat).Mul(d.clone(), other.clone())}
}

func (d Decimal) DivInt64(v int64) Decimal {
	if v == 0 {
		return Zero()
	}
	return Decimal{rat: new(big.Rat).Quo(d.clone(), new(big.Rat).SetInt64(v))}
}

func (d Decimal) Ceil(scale int) Decimal {
	if scale < 0 {
		return d
	}
	if d.rat == nil || d.rat.Sign() == 0 {
		return Zero()
	}

	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	scaled := new(big.Rat).Mul(d.clone(), new(big.Rat).SetInt(factor))

	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.QuoRem(scaled.Num(), scaled.Denom(), remainder)
	if remainder.Sign() != 0 && scaled.Sign() > 0 {
		quotient.Add(quotient, big.NewInt(1))
	}

	return Decimal{rat: new(big.Rat).SetFrac(quotient, factor)}
}

func (d Decimal) Abs() Decimal {
	rat := d.clone()
	if rat.Sign() < 0 {
		rat.Neg(rat)
	}
	return Decimal{rat: rat}
}

func (d Decimal) Neg() Decimal {
	return Decimal{rat: new(big.Rat).Neg(d.clone())}
}

func (d Decimal) String() string {
	if d.rat == nil {
		return "0"
	}

	value := d.rat.FloatString(scale)
	value = strings.TrimRight(value, "0")
	value = strings.TrimRight(value, ".")
	if value == "" || value == "-0" {
		return "0"
	}
	return value
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Decimal) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*d = Zero()
		return nil
	}

	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		parsed, parseErr := Parse(stringValue)
		if parseErr != nil {
			return parseErr
		}
		*d = parsed
		return nil
	}

	var numericValue json.Number
	if err := json.Unmarshal(data, &numericValue); err != nil {
		return err
	}
	parsed, err := Parse(numericValue.String())
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
}

func (d *Decimal) Scan(src interface{}) error {
	switch value := src.(type) {
	case nil:
		*d = Zero()
		return nil
	case int64:
		*d = FromInt64(value)
		return nil
	case float64:
		parsed, err := Parse(fmt.Sprintf("%.8f", value))
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	case []byte:
		parsed, err := Parse(string(value))
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	case string:
		parsed, err := Parse(value)
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	default:
		return fmt.Errorf("unsupported decimal scan type %T", src)
	}
}
