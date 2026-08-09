package spec

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
)

// Port bounds for the port variable kind.
const (
	PortMin = 1
	PortMax = 65535
)

// int64BitSize is the bit size used when parsing int64 literals.
const int64BitSize = 64

// Kind identifies a supported variable type.
type Kind int

// Supported variable kinds.
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

// Spec describes the static properties of a variable kind.
type Spec struct {
	Name          string
	GoType        string
	JSONType      string
	JSONFormat    string
	StringLike    bool
	Constrainable bool
}

// Spec returns the static description of the kind.
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

// ParseKind resolves a type name to a Kind and reports whether the name is known.
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

// IsStringLike reports whether the kind is a string-like type.
func (k Kind) IsStringLike() bool {
	return k.Spec().StringLike
}

// IsConstrainable reports whether the kind accepts prefix and regex constraints.
func (k Kind) IsConstrainable() bool {
	return k.Spec().Constrainable
}

// IsPort reports whether the kind is a TCP port.
func (k Kind) IsPort() bool {
	return k == KindPort
}

// IsValidEmail reports whether s looks like an email address.
func IsValidEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	dot := strings.LastIndexByte(s, '.')

	return at > 0 && dot > at+1
}

// ValidateLiteral checks that s is a valid literal of the kind.
func (k Kind) ValidateLiteral(s string) error {
	switch k {
	case KindInt:
		return parseSignedInt(s, "int", 0)
	case KindInt64:
		return parseSignedInt(s, "int64", int64BitSize)
	case KindPort:
		return parsePort(s)
	case KindBool:
		return parseBoolLiteral(s)
	case KindFloat64:
		return parseFloatLiteral(s)
	case KindURL:
		return parseURLLiteral(s)
	case KindEmail:
		return parseEmailLiteral(s)
	default:
		return nil
	}
}

func parseSignedInt(s, typ string, bits int) error {
	if _, err := strconv.ParseInt(s, 10, bits); err != nil {
		return fmt.Errorf("invalid default %q for type %s: %w", s, typ, err)
	}

	return nil
}

func parsePort(s string) error {
	n, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return fmt.Errorf("invalid default %q for type port: %w", s, err)
	}

	if n < PortMin || n > PortMax {
		return fmt.Errorf("invalid default %q for type port: %d out of range", s, n)
	}

	return nil
}

func parseBoolLiteral(s string) error {
	if _, err := strconv.ParseBool(s); err != nil {
		return fmt.Errorf("invalid default %q for type bool: %w", s, err)
	}

	return nil
}

func parseFloatLiteral(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invalid default %q for type float64: %w", s, err)
	}

	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("invalid default %q for type float64: NaN and Infinity are not allowed", s)
	}

	return nil
}

func parseURLLiteral(s string) error {
	if _, err := url.ParseRequestURI(s); err != nil {
		return fmt.Errorf("invalid default %q for type url: %w", s, err)
	}

	return nil
}

func parseEmailLiteral(s string) error {
	if !IsValidEmail(s) {
		return fmt.Errorf("invalid default %q for type email: value is not a valid email", s)
	}

	return nil
}
