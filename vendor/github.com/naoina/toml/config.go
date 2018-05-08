package toml

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	stringutil "github.com/naoina/go-stringutil"
	"github.com/naoina/toml/ast"
)

type Config struct {

	NormFieldName func(typ reflect.Type, keyOrField string) string

	FieldToKey func(typ reflect.Type, field string) string

	MissingField func(typ reflect.Type, key string) error
}

var DefaultConfig = Config{
	NormFieldName: defaultNormFieldName,
	FieldToKey:    snakeCase,
}

func defaultNormFieldName(typ reflect.Type, s string) string {
	return strings.Replace(strings.ToLower(s), "_", "", -1)
}

func snakeCase(typ reflect.Type, s string) string {
	return stringutil.ToSnakeCase(s)
}

func defaultMissingField(typ reflect.Type, key string) error {
	return fmt.Errorf("field corresponding to `%s' is not defined in %v", key, typ)
}

func NewEncoder(w io.Writer) *Encoder {
	return DefaultConfig.NewEncoder(w)
}

func Marshal(v interface{}) ([]byte, error) {
	return DefaultConfig.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return DefaultConfig.Unmarshal(data, v)
}

func UnmarshalTable(t *ast.Table, v interface{}) error {
	return DefaultConfig.UnmarshalTable(t, v)
}

func NewDecoder(r io.Reader) *Decoder {
	return DefaultConfig.NewDecoder(r)
}
