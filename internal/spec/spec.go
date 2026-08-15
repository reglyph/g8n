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
	TSType        string
	JSONType      string
	JSONFormat    string
	StringLike    bool
	Constrainable bool
}

// Spec returns the static description of the kind.
func (k Kind) Spec() Spec {
	switch k {
	case KindInt:
		return Spec{Name: "int", GoType: "int", TSType: "number", JSONType: "integer"}
	case KindInt64:
		return Spec{Name: "int64", GoType: "int64", TSType: "number", JSONType: "integer"}
	case KindBool:
		return Spec{Name: "bool", GoType: "bool", TSType: "boolean", JSONType: "boolean"}
	case KindFloat64:
		return Spec{Name: "float64", GoType: "float64", TSType: "number", JSONType: "number"}
	case KindPort:
		return Spec{Name: "port", GoType: "int", TSType: "number", JSONType: "integer"}
	case KindURL:
		return Spec{Name: "url", GoType: "string", TSType: "string", JSONType: "string", JSONFormat: "uri", StringLike: true, Constrainable: true}
	case KindEmail:
		return Spec{Name: "email", GoType: "string", TSType: "string", JSONType: "string", JSONFormat: "email", StringLike: true, Constrainable: true}
	case KindEnum:
		return Spec{Name: "enum", GoType: "string", TSType: "string", JSONType: "string", StringLike: true}
	default:
		return Spec{Name: "string", GoType: "string", TSType: "string", JSONType: "string", StringLike: true, Constrainable: true}
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

// Literal is a parsed and validated default value.
type Literal struct {
	Str   string
	Int   int64
	Float float64
	Bool  bool
}

// ParseLiteral parses s as a literal of the kind.
func (k Kind) ParseLiteral(s string) (Literal, error) {
	var out Literal

	switch k {
	case KindInt:
		n, err := parseSignedInt(s, "int", 0)
		out.Int = n

		return out, err
	case KindInt64:
		n, err := parseSignedInt(s, "int64", int64BitSize)
		out.Int = n

		return out, err
	case KindPort:
		n, err := parsePort(s)
		out.Int = n

		return out, err
	case KindBool:
		b, err := parseBoolLiteral(s)
		out.Bool = b

		return out, err
	case KindFloat64:
		v, err := parseFloatLiteral(s)
		out.Float = v

		return out, err
	case KindURL:
		str, err := parseURLLiteral(s)
		out.Str = str

		return out, err
	case KindEmail:
		str, err := parseEmailLiteral(s)
		out.Str = str

		return out, err
	default:
		out.Str = s

		return out, nil
	}
}

// ValidateLiteral checks that s is a valid literal of the kind.
func (k Kind) ValidateLiteral(s string) error {
	_, err := k.ParseLiteral(s)
	return err
}

func parseSignedInt(s, typ string, bits int) (int64, error) {
	n, err := strconv.ParseInt(s, 10, bits)
	if err != nil {
		return 0, fmt.Errorf("invalid default %q for type %s: %w", s, typ, err)
	}

	return n, nil
}

func parsePort(s string) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("invalid default %q for type port: %w", s, err)
	}

	if n < PortMin || n > PortMax {
		return 0, fmt.Errorf("invalid default %q for type port: %d out of range", s, n)
	}

	return n, nil
}

func parseBoolLiteral(s string) (bool, error) {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf("invalid default %q for type bool: %w", s, err)
	}

	return v, nil
}

func parseFloatLiteral(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid default %q for type float64: %w", s, err)
	}

	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("invalid default %q for type float64: NaN and Infinity are not allowed", s)
	}

	return v, nil
}

func parseURLLiteral(s string) (string, error) {
	if _, err := url.ParseRequestURI(s); err != nil {
		return "", fmt.Errorf("invalid default %q for type url: %w", s, err)
	}

	return s, nil
}

func parseEmailLiteral(s string) (string, error) {
	if !IsValidEmail(s) {
		return "", fmt.Errorf("invalid default %q for type email: value is not a valid email", s)
	}

	return s, nil
}
