package metrics

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type DuplicateMetric string

func (err DuplicateMetric) Error() string {
	return fmt.Sprintf("duplicate metric: %s", string(err))
}

type Registry interface {

	Each(func(string, interface{}))

	Get(string) interface{}

	GetOrRegister(string, interface{}) interface{}

	Register(string, interface{}) error

	RunHealthchecks()

	Unregister(string)

	UnregisterAll()
}

type StandardRegistry struct {
	metrics map[string]interface{}
	mutex   sync.Mutex
}

func NewRegistry() Registry {
	return &StandardRegistry{metrics: make(map[string]interface{})}
}

func (r *StandardRegistry) Each(f func(string, interface{})) {
	for name, i := range r.registered() {
		f(name, i)
	}
}

func (r *StandardRegistry) Get(name string) interface{} {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.metrics[name]
}

func (r *StandardRegistry) GetOrRegister(name string, i interface{}) interface{} {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if metric, ok := r.metrics[name]; ok {
		return metric
	}
	if v := reflect.ValueOf(i); v.Kind() == reflect.Func {
		i = v.Call(nil)[0].Interface()
	}
	r.register(name, i)
	return i
}

func (r *StandardRegistry) Register(name string, i interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.register(name, i)
}

func (r *StandardRegistry) RunHealthchecks() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, i := range r.metrics {
		if h, ok := i.(Healthcheck); ok {
			h.Check()
		}
	}
}

func (r *StandardRegistry) Unregister(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.metrics, name)
}

func (r *StandardRegistry) UnregisterAll() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for name, _ := range r.metrics {
		delete(r.metrics, name)
	}
}

func (r *StandardRegistry) register(name string, i interface{}) error {
	if _, ok := r.metrics[name]; ok {
		return DuplicateMetric(name)
	}
	switch i.(type) {
	case Counter, Gauge, GaugeFloat64, Healthcheck, Histogram, Meter, Timer:
		r.metrics[name] = i
	}
	return nil
}

func (r *StandardRegistry) registered() map[string]interface{} {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	metrics := make(map[string]interface{}, len(r.metrics))
	for name, i := range r.metrics {
		metrics[name] = i
	}
	return metrics
}

type PrefixedRegistry struct {
	underlying Registry
	prefix     string
}

func NewPrefixedRegistry(prefix string) Registry {
	return &PrefixedRegistry{
		underlying: NewRegistry(),
		prefix:     prefix,
	}
}

func NewPrefixedChildRegistry(parent Registry, prefix string) Registry {
	return &PrefixedRegistry{
		underlying: parent,
		prefix:     prefix,
	}
}

func (r *PrefixedRegistry) Each(fn func(string, interface{})) {
	wrappedFn := func(prefix string) func(string, interface{}) {
		return func(name string, iface interface{}) {
			if strings.HasPrefix(name, prefix) {
				fn(name, iface)
			} else {
				return
			}
		}
	}

	baseRegistry, prefix := findPrefix(r, "")
	baseRegistry.Each(wrappedFn(prefix))
}

func findPrefix(registry Registry, prefix string) (Registry, string) {
	switch r := registry.(type) {
	case *PrefixedRegistry:
		return findPrefix(r.underlying, r.prefix+prefix)
	case *StandardRegistry:
		return r, prefix
	}
	return nil, ""
}

func (r *PrefixedRegistry) Get(name string) interface{} {
	realName := r.prefix + name
	return r.underlying.Get(realName)
}

func (r *PrefixedRegistry) GetOrRegister(name string, metric interface{}) interface{} {
	realName := r.prefix + name
	return r.underlying.GetOrRegister(realName, metric)
}

func (r *PrefixedRegistry) Register(name string, metric interface{}) error {
	realName := r.prefix + name
	return r.underlying.Register(realName, metric)
}

func (r *PrefixedRegistry) RunHealthchecks() {
	r.underlying.RunHealthchecks()
}

func (r *PrefixedRegistry) Unregister(name string) {
	realName := r.prefix + name
	r.underlying.Unregister(realName)
}

func (r *PrefixedRegistry) UnregisterAll() {
	r.underlying.UnregisterAll()
}

var DefaultRegistry Registry = NewRegistry()

func Each(f func(string, interface{})) {
	DefaultRegistry.Each(f)
}

func Get(name string) interface{} {
	return DefaultRegistry.Get(name)
}

func GetOrRegister(name string, i interface{}) interface{} {
	return DefaultRegistry.GetOrRegister(name, i)
}

func Register(name string, i interface{}) error {
	return DefaultRegistry.Register(name, i)
}

func MustRegister(name string, i interface{}) {
	if err := Register(name, i); err != nil {
		panic(err)
	}
}

func RunHealthchecks() {
	DefaultRegistry.RunHealthchecks()
}

func Unregister(name string) {
	DefaultRegistry.Unregister(name)
}
