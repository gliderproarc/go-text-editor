package history

// KillRing stores a small history of killed text entries.
type KillRing struct {
	entries []string
	pos     int
}

const killRingMax = 10

// Push adds killed text to the ring.
func (k *KillRing) Push(s string) {
	if s == "" {
		return
	}
	if k.entries == nil {
		k.entries = make([]string, 0, killRingMax)
	}
	if len(k.entries) < killRingMax {
		k.entries = append(k.entries, "")
	}
	copy(k.entries[1:], k.entries[:len(k.entries)-1])
	k.entries[0] = s
	k.pos = 0
}

// Set stores the killed text.
func (k *KillRing) Set(s string) { k.Push(s) }

// Rotate moves to the next entry in the ring.
func (k *KillRing) Rotate() bool {
	if len(k.entries) <= 1 {
		return false
	}
	k.pos = (k.pos + 1) % len(k.entries)
	return true
}

// Current returns the current killed text.
func (k *KillRing) Current() string {
	if len(k.entries) == 0 {
		return ""
	}
	return k.entries[k.pos]
}

// Next returns the next entry that Rotate would select.
func (k *KillRing) Next() string {
	if len(k.entries) <= 1 {
		return ""
	}
	next := (k.pos + 1) % len(k.entries)
	return k.entries[next]
}

// Get returns the current killed text.
func (k *KillRing) Get() string { return k.Current() }

// Len returns the number of entries in the ring.
func (k *KillRing) Len() int { return len(k.entries) }

// HasData reports whether the ring contains text.
func (k *KillRing) HasData() bool { return len(k.entries) > 0 }
