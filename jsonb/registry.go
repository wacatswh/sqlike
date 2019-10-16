package jsonb

import (
	"encoding"
	"encoding/json"
	"reflect"
	"sync"

	"github.com/si3nloong/sqlike/reflext"
)

// ValueDecoder :
type ValueDecoder func(*Reader, reflect.Value) error

// ValueEncoder :
type ValueEncoder func(*Writer, reflect.Value) error

// Registry :
type Registry struct {
	mutex        sync.Mutex
	typeEncoders map[reflect.Type]ValueEncoder
	typeDecoders map[reflect.Type]ValueDecoder
	kindEncoders map[reflect.Kind]ValueEncoder
	kindDecoders map[reflect.Kind]ValueDecoder
}

var registry = buildRegistry()

func buildRegistry() *Registry {
	rg := NewRegistry()
	Decoder{}.SetDecoders(rg)
	Encoder{}.SetEncoders(rg)
	return rg
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		typeEncoders: make(map[reflect.Type]ValueEncoder),
		typeDecoders: make(map[reflect.Type]ValueDecoder),
		kindEncoders: make(map[reflect.Kind]ValueEncoder),
		kindDecoders: make(map[reflect.Kind]ValueDecoder),
	}
}

// SetTypeEncoder :
func (r *Registry) SetTypeEncoder(t reflect.Type, enc ValueEncoder) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.typeEncoders[t] = enc
}

// SetTypeDecoder :
func (r *Registry) SetTypeDecoder(t reflect.Type, dec ValueDecoder) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.typeDecoders[t] = dec
}

// SetKindEncoder :
func (r *Registry) SetKindEncoder(k reflect.Kind, enc ValueEncoder) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.kindEncoders[k] = enc
}

// SetKindDecoder :
func (r *Registry) SetKindDecoder(k reflect.Kind, dec ValueDecoder) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.kindDecoders[k] = dec
}

// LookupEncoder :
func (r *Registry) LookupEncoder(v reflect.Value) (ValueEncoder, error) {
	var (
		enc ValueEncoder
		ok  bool
	)

	if !v.IsValid() || reflext.IsNull(v) {
		return func(w *Writer, v reflect.Value) error {
			w.WriteString(null)
			return nil
		}, nil
	}

	it := v.Interface()
	if _, ok := it.(Marshaler); ok {
		return marshalerEncoder(), nil
	}

	if _, ok := it.(json.Marshaler); ok {
		return jsonMarshalerEncoder(), nil
	}

	if _, ok := it.(encoding.TextMarshaler); ok {
		return textMarshalerEncoder(), nil
	}

	t := v.Type()
	enc, ok = r.typeEncoders[t]
	if ok {
		return enc, nil
	}

	enc, ok = r.kindEncoders[t.Kind()]
	if ok {
		return enc, nil
	}
	return nil, ErrNoEncoder{Type: t}
}

// LookupDecoder :
func (r *Registry) LookupDecoder(t reflect.Type) (ValueDecoder, error) {
	var (
		dec ValueDecoder
		ok  bool
	)

	it := reflect.TypeOf((*Unmarshaler)(nil)).Elem()
	if t.Implements(it) {
		return unmarshalerDecoder(), nil
	}

	it = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	if t.Implements(it) {
		return jsonUnmarshalerDecoder(), nil
	}

	dec, ok = r.typeDecoders[t]
	if ok {
		return dec, nil
	}

	dec, ok = r.kindDecoders[t.Kind()]
	if ok {
		return dec, nil
	}
	return nil, ErrNoDecoder{Type: t}
}
