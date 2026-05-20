package service

import (
	"fmt"
	"sync"
	"time"
)

// geofencePendingState хранит отложенные переходы входа/выхода для пары пользователь–чекпоинт.
type geofencePendingState struct {
	pendingEnterSince *time.Time
	pendingExitSince  *time.Time
}

type geofenceStateStore struct {
	mu    sync.Mutex
	items map[string]*geofencePendingState
}

func newGeofenceStateStore() *geofenceStateStore {
	return &geofenceStateStore{items: make(map[string]*geofencePendingState)}
}

func geofenceStateKey(userID, checkpointID int) string {
	return fmt.Sprintf("%d:%d", userID, checkpointID)
}

func (s *geofenceStateStore) get(userID, checkpointID int) *geofencePendingState {
	key := geofenceStateKey(userID, checkpointID)
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.items[key]
	if !ok {
		st = &geofencePendingState{}
		s.items[key] = st
	}
	return st
}

func (st *geofencePendingState) clearPendingEnter() {
	st.pendingEnterSince = nil
}

func (st *geofencePendingState) clearPendingExit() {
	st.pendingExitSince = nil
}

func (st *geofencePendingState) markPendingEnter(now time.Time) {
	if st.pendingEnterSince == nil {
		t := now
		st.pendingEnterSince = &t
	}
}

func (st *geofencePendingState) markPendingExit(now time.Time) {
	if st.pendingExitSince == nil {
		t := now
		st.pendingExitSince = &t
	}
}

func (st *geofencePendingState) pendingEnterElapsed(now time.Time, graceSeconds int) bool {
	if st.pendingEnterSince == nil {
		return false
	}
	return now.Sub(*st.pendingEnterSince) >= time.Duration(graceSeconds)*time.Second
}

func (st *geofencePendingState) pendingExitElapsed(now time.Time, graceSeconds int) bool {
	if st.pendingExitSince == nil {
		return false
	}
	return now.Sub(*st.pendingExitSince) >= time.Duration(graceSeconds)*time.Second
}
