package util

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func NeverOnCreate(_ event.CreateEvent) bool {
	return false
}

func NeverOnDelete(_ event.DeleteEvent) bool {
	return false
}

func NeverOnGeneric(_ event.GenericEvent) bool {
	return false
}

func NeverOnUpdate(_ event.UpdateEvent) bool {
	return false
}

func AlwaysOnCreate(_ event.CreateEvent) bool {
	return true
}

func AlwaysOnDelete(_ event.DeleteEvent) bool {
	return true
}

func AlwaysOnGeneric(_ event.GenericEvent) bool {
	return true
}

func AlwaysOnUpdate(_ event.UpdateEvent) bool {
	return true
}
