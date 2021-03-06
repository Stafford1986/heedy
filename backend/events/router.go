package events

import (
	"errors"
	"fmt"
	"sync"
)

var ErrNotSubscribed = errors.New("Not subscribed")

// Map: NOT THREADSAFE
type Map map[string]*MultiHandler

func (em Map) Fire(e *Event) {
	m, ok := em[e.Event]
	if ok {
		m.Fire(e)
	}
	m, ok = em["*"]
	if ok {
		m.Fire(e)
	}
}

func (em Map) Subscribe(event string, h Handler) error {
	mh, ok := em[event]
	if !ok {
		mh = NewMultiHandler()
		em[event] = mh
	}
	return mh.AddHandler(h)
}

func (em Map) Unsubscribe(event string, h Handler) error {
	mh, ok := em[event]
	if !ok {
		return ErrNotSubscribed
	}
	return mh.RemoveHandler(h)
}

func NewMap() Map {
	return make(Map)
}

// Permitted queries:
//	user
//	app
//	object
//	plugin
//	app-key
// 	plugin-key
// 	user-plugin
//	user-plugin-key

type idKey2 struct {
	id  string
	id2 string
}
type idKey3 struct {
	id  string
	id2 string
	id3 string
}

type Router struct {
	sync.RWMutex

	UserEvents       map[string]Map
	AppEvents map[string]Map
	PluginEvents     map[string]Map
	ObjectEvents     map[string]Map

	UserPlugin    map[idKey2]Map
	AppKey map[idKey2]Map
	PluginKey     map[idKey2]Map

	UserPluginKey map[idKey3]Map

	// If ObjectType is set, we send it over the full thing again,
	// but this time with the given object type
	ObjectType map[string]*Router
}

func NewRouter() *Router {
	// None of the maps are initialized at the beginning, they get set up with subscribe
	return &Router{}
}

func (er *Router) Subscribe(e Event, h Handler) error {
	er.Lock()
	defer er.Unlock()

	if e.Type != "" {
		if er.ObjectType == nil {
			er.ObjectType = make(map[string]*Router)
		}
		em, ok := er.ObjectType[e.Type]
		if !ok {
			em = NewRouter()
			er.ObjectType[e.Type] = em
		}
		// Set the type to empty string
		e.Type = ""
		return em.Subscribe(e, h)
	}
	if e.User != "" && e.Plugin != nil && *e.Plugin != "" && e.Key != "" {
		if er.UserPluginKey == nil {
			er.UserPluginKey = make(map[idKey3]Map)
		}
		em, ok := er.UserPluginKey[idKey3{e.User, *e.Plugin, e.Key}]
		if !ok {
			em = NewMap()
			er.UserPluginKey[idKey3{e.User, *e.Plugin, e.Key}] = em
		}
		return em.Subscribe(e.Event, h)
	}
	if e.Plugin != nil && *e.Plugin != "" && e.Key != "" {
		if er.PluginKey == nil {
			er.PluginKey = make(map[idKey2]Map)
		}
		em, ok := er.PluginKey[idKey2{*e.Plugin, e.Key}]
		if !ok {
			em = NewMap()
			er.PluginKey[idKey2{*e.Plugin, e.Key}] = em
		}
		return em.Subscribe(e.Event, h)
	}
	if e.App != "" && e.Key != "" {
		if er.AppKey == nil {
			er.AppKey = make(map[idKey2]Map)
		}
		em, ok := er.AppKey[idKey2{e.App, e.Key}]
		if !ok {
			em = NewMap()
			er.AppKey[idKey2{e.App, e.Key}] = em
		}
		return em.Subscribe(e.Event, h)
	}
	if e.Plugin != nil && *e.Plugin != "" && e.User != "" {
		if er.UserPlugin == nil {
			er.UserPlugin = make(map[idKey2]Map)
		}
		em, ok := er.UserPlugin[idKey2{e.User, *e.Plugin}]
		if !ok {
			em = NewMap()
			er.UserPlugin[idKey2{e.User, *e.Plugin}] = em
		}
		return em.Subscribe(e.Event, h)
	}
	if e.Object != "" {
		if er.ObjectEvents == nil {
			er.ObjectEvents = make(map[string]Map)
		}
		em, ok := er.ObjectEvents[e.Object]
		if !ok {
			em = NewMap()
			er.ObjectEvents[e.Object] = em
		}
		return em.Subscribe(e.Event, h)
	}
	if e.Plugin != nil && *e.Plugin != "" {
		if er.PluginEvents == nil {
			er.PluginEvents = make(map[string]Map)
		}
		em, ok := er.PluginEvents[*e.Plugin]
		if !ok {
			em = NewMap()
			er.PluginEvents[*e.Plugin] = em
		}
		return em.Subscribe(e.Event, h)
	}
	if e.App != "" {
		if er.AppEvents == nil {
			er.AppEvents = make(map[string]Map)
		}
		em, ok := er.AppEvents[e.App]
		if !ok {
			em = NewMap()
			er.AppEvents[e.App] = em
		}
		return em.Subscribe(e.Event, h)
	}
	if e.User != "" {
		if er.UserEvents == nil {
			er.UserEvents = make(map[string]Map)
		}
		em, ok := er.UserEvents[e.User]
		if !ok {
			em = NewMap()
			er.UserEvents[e.User] = em
		}
		return em.Subscribe(e.Event, h)
	}

	return fmt.Errorf("Could not subscribe to %s", e.String())
}

func (er *Router) Unsubscribe(e Event, h Handler) error {
	er.Lock()
	defer er.Unlock()
	if e.Type != "" {
		if er.ObjectType == nil {
			return ErrNotSubscribed
		}
		em, ok := er.ObjectType[e.Type]
		if !ok {
			return ErrNotSubscribed
		}
		// Set the type to empty string
		e.Type = ""
		return em.Unsubscribe(e, h)
	}
	if e.User != "" && e.Plugin != nil && *e.Plugin != "" && e.Key != "" {
		if er.UserPluginKey == nil {
			return ErrNotSubscribed
		}
		em, ok := er.UserPluginKey[idKey3{e.User, *e.Plugin, e.Key}]
		if !ok {
			return ErrNotSubscribed
		}
		return em.Unsubscribe(e.Event, h)
	}
	if e.Plugin != nil && *e.Plugin != "" && e.Key != "" {
		if er.PluginKey == nil {
			return ErrNotSubscribed
		}
		em, ok := er.PluginKey[idKey2{*e.Plugin, e.Key}]
		if !ok {
			return ErrNotSubscribed
		}
		return em.Unsubscribe(e.Event, h)
	}
	if e.App != "" && e.Key != "" {
		if er.AppKey == nil {
			return ErrNotSubscribed
		}
		em, ok := er.AppKey[idKey2{e.App, e.Key}]
		if !ok {
			return ErrNotSubscribed
		}
		return em.Unsubscribe(e.Event, h)
	}
	if e.Plugin != nil && *e.Plugin != "" && e.User != "" {
		if er.UserPlugin == nil {
			return ErrNotSubscribed
		}
		em, ok := er.UserPlugin[idKey2{e.User, *e.Plugin}]
		if !ok {
			return ErrNotSubscribed
		}
		return em.Unsubscribe(e.Event, h)
	}
	if e.Object != "" {
		if er.ObjectEvents == nil {
			return ErrNotSubscribed
		}
		em, ok := er.ObjectEvents[e.Object]
		if !ok {
			return ErrNotSubscribed
		}
		return em.Unsubscribe(e.Event, h)
	}
	if e.Plugin != nil && *e.Plugin != "" {
		if er.PluginEvents == nil {
			return ErrNotSubscribed
		}
		em, ok := er.PluginEvents[*e.Plugin]
		if !ok {
			return ErrNotSubscribed
		}
		return em.Unsubscribe(e.Event, h)
	}
	if e.App != "" {
		if er.AppEvents == nil {
			return ErrNotSubscribed
		}
		em, ok := er.AppEvents[e.App]
		if !ok {
			return ErrNotSubscribed
		}
		return em.Unsubscribe(e.Event, h)
	}
	if e.User != "" {
		if er.UserEvents == nil {
			return ErrNotSubscribed
		}
		em, ok := er.UserEvents[e.User]
		if !ok {
			return ErrNotSubscribed
		}
		return em.Unsubscribe(e.Event, h)
	}

	return fmt.Errorf("Could not unsubscribe from %s", e.String())
}

func (er *Router) Fire(e *Event) {
	er.RLock()
	defer er.RUnlock()

	// User Subscriptions
	if er.UserEvents != nil {
		h, ok := er.UserEvents[e.User]
		if ok {
			h.Fire(e)
		}
		h, ok = er.UserEvents["*"]
		if ok {
			h.Fire(e)
		}
	}

	// App Subscriptions
	if e.App == "" {
		return
	}
	if er.AppEvents != nil {
		h, ok := er.AppEvents[e.App]
		if ok {
			h.Fire(e)
		}
		h, ok = er.AppEvents["*"]
		if ok {
			h.Fire(e)
		}
	}
	if e.Plugin != nil && *e.Plugin != "" {
		if er.PluginEvents != nil {
			h, ok := er.PluginEvents[*e.Plugin]
			if ok {
				h.Fire(e)
			}
		}
		if er.UserPlugin != nil {
			h, ok := er.UserPlugin[idKey2{e.User, *e.Plugin}]
			if ok {
				h.Fire(e)
			}
		}
	}

	// Object Subscriptions
	if e.Object == "" {
		return
	}
	if er.ObjectEvents != nil {
		h, ok := er.ObjectEvents[e.Object]
		if ok {
			h.Fire(e)
		}
		h, ok = er.ObjectEvents["*"]
		if ok {
			h.Fire(e)
		}
	}

	// This will always be nil in the Type router
	if er.ObjectType != nil {
		h, ok := er.ObjectType[e.Type]
		if ok {
			h.Fire(e)
		}
	}

	if e.Key == "" {
		return
	}
	if er.AppKey != nil {
		h, ok := er.AppKey[idKey2{e.App, e.Key}]
		if ok {
			h.Fire(e)
		}
	}
	if e.Plugin == nil || *e.Plugin == "" {
		return
	}
	if er.PluginKey != nil {
		h, ok := er.PluginKey[idKey2{*e.Plugin, e.Key}]
		if ok {
			h.Fire(e)
		}
	}
	if er.UserPluginKey != nil {
		h, ok := er.UserPluginKey[idKey3{e.User, *e.Plugin, e.Key}]
		if ok {
			h.Fire(e)
		}
	}
}
