package spec

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	PortMin = 1
	PortMax = 65535
)

type Kind int

const (
	KindString Kind = iota
	KindInt
	KindInt64
	KindBool
	KindFloat64
	KindPort
	KindURL
	KindEmail
	KindEnum
)

type Spec struct {
	Name          string
	GoType        string
	JSONType      string
	JSONFormat    string
	StringLike    bool
	Constrainable bool
}

func (k Kind) Spec() Spec {
	switch k {
	case KindInt:
		return Spec{Name: "int", GoType: "int", JSONType: "integer"}
	case KindInt64:
		return Spec{Name: "int64", GoType: "int64", JSONType: "integer"}
	case KindBool:
		return Spec{Name: "bool", GoType: "bool", JSONType: "boolean"}
	case KindFloat64:
		return Spec{Name: "float64", GoType: "float64", JSONType: "number"}
	case KindPort:
		return Spec{Name: "port", GoType: "int", JSONType: "integer"}
	case KindURL:
		return Spec{Name: "url", GoType: "string", JSONType: "string", JSONFormat: "uri", StringLike: true, Constrainable: true}
	case KindEmail:
		return Spec{Name: "email", GoType: "string", JSONType: "string", JSONFormat: "email", StringLike: true, Constrainable: true}
	case KindEnum:
		return Spec{Name: "enum", GoType: "string", JSONType: "string", StringLike: true}
	default:
		return Spec{Name: "string", GoType: "string", JSONType: "string", StringLike: true, Constrainable: true}
	}
}

func (k Kind) String() string {
	return k.Spec().Name
}

func ParseKind(name string) (Kind, bool) {
	switch name {
	case "string":
		return KindString, true
	case "int":
		return KindInt, true
	case "int64":
		return KindInt64, true
	case "bool":
		return KindBool, true
	case "float64":
		return KindFloat64, true
	case "port":
		return KindPort, true
	case "url":
		return KindURL, true
	case "email":
		return KindEmail, true
	case "enum":
		return KindEnum, true
	default:
		return KindString, false
	}
}

func (k Kind) IsStringLike() bool {
	return k.Spec().StringLike
}

func (k Kind) IsConstrainable() bool {
	return k.Spec().Constrainable
}

func (k Kind) IsPort() bool {
	return k == KindPort
}

func IsValidEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	dot := strings.LastIndexByte(s, '.')

	return at > 0 && dot > at+1
}

func (k Kind) ValidateLiteral(s string) error {
	switch k {
	case KindInt:
		if _, err := strconv.ParseInt(s, 10, 0); err != nil {
			return fmt.Errorf("invalid default %q for type int: %w", s, err)
		}

		return nil

	case KindInt64:
		if _, err := strconv.ParseInt(s, 10, 64); err != nil {
			return fmt.Errorf("invalid default %q for type int64: %w", s, err)
		}

		return nil

	case KindBool:
		if _, err := strconv.ParseBool(s); err != nil {
			return fmt.Errorf("invalid default %q for type bool: %w", s, err)
		}

		return nil

	case KindFloat64:
		v, err := strconv.ParseFloat(s, 64)

		if err != nil {
			return fmt.Errorf("invalid default %q for type float64: %w", s, err)
		}
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("invalid default %q for type float64: NaN and Infinity are not allowed", s)
		}

		return nil

	case KindPort:
		n, err := strconv.ParseInt(s, 10, 0)

		if err != nil {
			return fmt.Errorf("invalid default %q for type port: %w", s, err)
		}
		if n < PortMin || n > PortMax {
			return fmt.Errorf("invalid default %q for type port: %d out of range", s, n)
		}

		return nil

	case KindEmail:
		if !IsValidEmail(s) {
			return fmt.Errorf("invalid default %q for type email: value is not a valid email", s)
		}

	default:
		return nil
	}

	return nil
}
