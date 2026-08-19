package orm

import "fmt"

type beforeCreator interface{ BeforeCreate() error }
type afterCreator interface{ AfterCreate() error }
type beforeUpdater interface{ BeforeUpdate() error }
type afterUpdater interface{ AfterUpdate() error }
type beforeDeleter interface{ BeforeDelete() error }
type afterDeleter interface{ AfterDelete() error }

func callHook(model any, name string) error {
	var err error
	switch name {
	case "BeforeCreate":
		if h, ok := model.(beforeCreator); ok {
			err = h.BeforeCreate()
		}
	case "AfterCreate":
		if h, ok := model.(afterCreator); ok {
			err = h.AfterCreate()
		}
	case "BeforeUpdate":
		if h, ok := model.(beforeUpdater); ok {
			err = h.BeforeUpdate()
		}
	case "AfterUpdate":
		if h, ok := model.(afterUpdater); ok {
			err = h.AfterUpdate()
		}
	case "BeforeDelete":
		if h, ok := model.(beforeDeleter); ok {
			err = h.BeforeDelete()
		}
	case "AfterDelete":
		if h, ok := model.(afterDeleter); ok {
			err = h.AfterDelete()
		}
	}
	if err != nil {
		return fmt.Errorf("gai/orm: %s hook: %w", name, err)
	}
	return nil
}
