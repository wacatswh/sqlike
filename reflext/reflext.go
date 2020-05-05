package reflext

import (
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type Structer interface {
	Fields() []StructFielder
	Properties() []StructFielder
	LookUpFieldByName(name string) (StructFielder, bool)
	GetByTraversal(index []int) StructFielder
}

type StructFielder interface {
	Name() string // New name of struct field
	Type() reflect.Type
	Index() []int
	Tag() StructTag
	Parent() StructFielder
	ParentByTraversal(cb func(StructFielder) bool) StructFielder
	Children() []StructFielder
	IsNullable() bool
	IsEmbedded() bool
}

// StructTag :
type StructTag struct {
	originalName string
	name         string
	opts         map[string]string
}

// Name :
func (st StructTag) Name() string {
	return st.name
}

// OriginalName :
func (st StructTag) OriginalName() string {
	return st.originalName
}

// Get :
func (st StructTag) Get(key string) string {
	return st.opts[key]
}

// LookUp :
func (st StructTag) LookUp(key string) (val string, exist bool) {
	val, exist = st.opts[key]
	return
}

// StructField :
type StructField struct {
	id       string
	idx      []int
	name     string
	path     string
	t        reflect.Type
	null     bool
	tag      StructTag
	embed    bool
	parent   StructFielder
	children []StructFielder
}

var _ StructFielder = (*StructField)(nil)

// Name :
func (sf *StructField) Name() string {
	return sf.path
}

// Type :
func (sf *StructField) Type() reflect.Type {
	return sf.t
}

// Tag :
func (sf *StructField) Tag() StructTag {
	return sf.tag
}

// Index :
func (sf *StructField) Index() []int {
	return sf.idx
}

// Parent :
func (sf *StructField) Parent() StructFielder {
	return sf.parent
}

func (sf *StructField) Children() []StructFielder {
	return sf.children
}

func (sf *StructField) IsNullable() bool {
	return sf.null
}

func (sf *StructField) IsEmbedded() bool {
	return sf.embed
}

// ParentByTraversal :
func (sf *StructField) ParentByTraversal(cb func(StructFielder) bool) StructFielder {
	prnt := sf.parent
	for prnt != nil {
		if cb(prnt) {
			break
		}
		prnt = prnt.Parent()
	}
	return prnt
}

// Struct :
type Struct struct {
	tree       StructFielder
	fields     Fields // all fields belong to this struct
	properties Fields // available properties in sequence
	indexes    map[string]StructFielder
	names      map[string]StructFielder
}

var _ Structer = (*Struct)(nil)

// Fields :
func (s *Struct) Fields() []StructFielder {
	return s.fields
}

// Properties :
func (s *Struct) Properties() []StructFielder {
	return s.properties
}

// LookUpFieldByName :
func (s *Struct) LookUpFieldByName(name string) (StructFielder, bool) {
	x, ok := s.names[name]
	return x, ok
}

// GetByTraversal :
func (s *Struct) GetByTraversal(index []int) StructFielder {
	if len(index) == 0 {
		return nil
	}

	tree := s.tree
	for _, i := range index {
		children := tree.Children()
		if i >= len(children) || children[i] == nil {
			return nil
		}
		tree = children[i]
	}
	return tree
}

// Fields :
type Fields []StructFielder

func (x Fields) Len() int { return len(x) }

func (x Fields) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func (x Fields) Less(i, j int) bool {
	for k, xik := range x[i].Index() {
		if k >= len(x[j].Index()) {
			return false
		}
		if xik != x[j].Index()[k] {
			return xik < x[j].Index()[k]
		}
	}
	return len(x[i].Index()) < len(x[j].Index())
}

type typeQueue struct {
	t  reflect.Type
	sf *StructField
	pp string // parent path
}

func getCodec(t reflect.Type, tagName string, fmtFunc FormatFunc) *Struct {
	fields := make([]StructFielder, 0)

	root := &StructField{}
	queue := []typeQueue{}
	queue = append(queue, typeQueue{Deref(t), root, ""})

	for len(queue) > 0 {
		q := queue[0]
		q.sf.children = make([]StructFielder, 0)

		for i := 0; i < q.t.NumField(); i++ {
			f := q.t.Field(i)

			// skip unexported fields
			if len(f.PkgPath) != 0 && !f.Anonymous {
				continue
			}

			tag := parseTag(f, tagName, fmtFunc)
			if tag.name == "-" {
				continue
			}

			sf := &StructField{
				id:       strings.TrimLeft(q.sf.id+"."+strconv.Itoa(i), "."),
				name:     f.Name,
				path:     tag.name,
				null:     q.sf.null || IsNullable(f.Type),
				t:        f.Type,
				tag:      tag,
				children: make([]StructFielder, 0),
			}

			if len(q.sf.Index()) > 0 {
				sf.parent = q.sf
			}

			if sf.path == "" {
				sf.path = sf.tag.name
			}

			if q.pp != "" {
				sf.path = q.pp + "." + sf.path
			}

			ft := Deref(f.Type)
			q.sf.children = append(q.sf.children, sf)
			sf.idx = appendSlice(q.sf.idx, i)
			sf.embed = ft.Kind() == reflect.Struct && f.Anonymous

			if ft.Kind() == reflect.Struct {
				// check recursive, prevent infinite loop
				if q.t == ft {
					goto nextStep
				}

				// embedded struct
				path := sf.path
				if f.Anonymous {
					if sf.tag.OriginalName() == "" {
						path = q.pp
					}
					// queue = append(queue, typeQueue{ft, sf, path})
				}

				queue = append(queue, typeQueue{ft, sf, path})
			}

		nextStep:
			fields = append(fields, sf)
		}

		queue = queue[1:]
	}

	codec := &Struct{
		tree:       root,
		fields:     fields,
		properties: make([]StructFielder, 0, len(fields)),
		indexes:    make(map[string]StructFielder),
		names:      make(map[string]StructFielder),
	}

	sort.Sort(codec.fields)

	for _, sf := range codec.fields {
		codec.indexes[sf.(*StructField).id] = sf
		if sf.Name() != "" && !sf.IsEmbedded() {
			codec.names[sf.Name()] = sf
			prnt := sf.ParentByTraversal(func(f StructFielder) bool {
				return !f.IsEmbedded()
			})
			if len(sf.Index()) > 1 &&
				sf.Parent() != nil && prnt != nil {
				continue
			}
			// not nested embedded struct or embedded struct
			codec.properties = append(codec.properties, sf)
		}
	}

	return codec
}

func appendSlice(s []int, i int) []int {
	x := make([]int, len(s)+1)
	copy(x, s)
	x[len(x)-1] = i
	return x
}

func parseTag(f reflect.StructField, tagName string, fmtFunc FormatFunc) (st StructTag) {
	parts := strings.Split(f.Tag.Get(tagName), ",")
	name := strings.TrimSpace(parts[0])
	st.originalName = name
	if name == "" {
		name = f.Name
		if fmtFunc != nil {
			name = fmtFunc(name)
		}
	}
	st.name = name
	st.opts = make(map[string]string)
	if len(parts) > 1 {
		for _, opt := range parts[1:] {
			opt = strings.TrimSpace(opt)
			if strings.Contains(opt, "=") {
				kv := strings.SplitN(opt, "=", 2)
				k := strings.TrimSpace(strings.ToLower(kv[0]))
				st.opts[k] = strings.TrimSpace(kv[1])
				continue
			}
			opt = strings.ToLower(opt)
			st.opts[opt] = ""
		}
	}
	return
}
